package fetcher

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ScrpTrx-Go/GoTGParse/internal/config"
	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/model"
	pkg "github.com/ScrpTrx-Go/GoTGParse/pkg/logger"
	"github.com/zelenin/go-tdlib/client"
)

type TDLibFetcher struct {
	client        *client.Client
	me            *client.User
	log           pkg.Logger
	cfg           config.TDLibConfig
	totalFetched  int
	totalFiltered int
	totalErrors   int
}

func NewTDLibFetcher(tdlibClient *client.Client, log pkg.Logger, cfg config.TDLibConfig) (*TDLibFetcher, error) {
	me, err := tdlibClient.GetMe()
	if err != nil {
		return nil, fmt.Errorf("GetMe error: %w", err)
	}
	log.Info("Authorized successfully", "user_id", me.Id, "first_name", me.FirstName)
	log.Info("New TDLibFetcher was created")
	return &TDLibFetcher{
		client: tdlibClient,
		me:     me,
		log:    log,
		cfg:    cfg,
	}, nil
}

func (f *TDLibFetcher) RunFetchPipelene(ctx context.Context, from, to time.Time) <-chan *model.Post {
	out := make(chan *model.Post)
	go func() {
		var wg sync.WaitGroup

		for _, username := range f.cfg.Usernames {
			wg.Add(1)
			go func(username string) {
				defer wg.Done()

				chatID, err := f.FindChat(username)
				if err != nil {
					f.log.Error("Failed to find chat", "username", username, "err", err)
					f.totalErrors++
					return
				}

				resultCh, errCh := f.RunPipeline(ctx, chatID, from, to)
				f.log.Info("Fetch pipeline started", "username", username)

				count := 0
				for {
					select {
					case <-ctx.Done():
						f.log.Warn("Context canceled", "username", username)
						return
					case post, ok := <-resultCh:
						if !ok {
							f.log.Info("Fetch workers completed", "username", username, "count", count)
							return
						}
						post.Username = username
						f.totalFetched++
						count++
						out <- post
					case err, ok := <-errCh:
						if ok {
							f.totalErrors++
							f.log.Error("Pipeline error", "username", username, "err", err)
						}
					}
				}

			}(username)
		}
		wg.Wait()
		close(out)
		f.log.Info("All usernames processed", "total_fetched", f.totalFetched, "total_errors", f.totalErrors)
	}()
	return out
}

func (f *TDLibFetcher) RunPipeline(ctx context.Context, chatID int64, from, to time.Time) (<-chan *model.Post, <-chan error) {
	const numWorkers = 5

	rawOut := make(chan *client.Message)
	postOut := make(chan *model.Post)
	errCh := make(chan error, 1)

	go func() {
		defer close(rawOut)
		defer close(errCh)
		f.log.Info("Producer: GetHistoryByPeriod started", "from", from, "to", to)
		err := f.GetHistoryByPeriod(ctx, chatID, from, to, rawOut)
		if err != nil {
			errCh <- fmt.Errorf("GetHistory failed: %w", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			f.log.Debug("Worker started", "worker", workerID)
			for raw := range rawOut {
				post, ok := f.ValidateMessage(raw)
				if !ok {
					f.totalFiltered++
					continue
				}
				link, err := f.getMessageLink(chatID, post.ID)
				if err != nil {
					f.totalErrors++
					f.log.Error("Failed to get message link", "id", post.ID, "err", err)
				}
				post.Link = link
				select {
				case <-ctx.Done():
					f.log.Warn("Context canceled in worker", "worker", workerID)
					return
				case postOut <- post:
				}
			}
		}(i)
	}

	go func() {
		wg.Wait()
		f.log.Info("All workers finished")
		close(postOut)
	}()

	return postOut, errCh
}

func (f *TDLibFetcher) FindChat(username string) (int64, error) {
	chat, err := f.client.SearchPublicChat(&client.SearchPublicChatRequest{Username: username})
	if err != nil {
		return 0, fmt.Errorf("SearchPublicChat error: %w", err)
	}
	if chat == nil {
		return 0, fmt.Errorf("chat is nil after SearchPublicChat")
	}
	f.log.Info("Chat found", "username", username, "chat_id", chat.Id)
	return chat.Id, nil
}

func (f *TDLibFetcher) GetHistoryByPeriod(ctx context.Context, chatID int64, from, to time.Time, out chan<- *client.Message) error {
	var fromMessageID int64
	stop := false

	for {
		select {
		case <-ctx.Done():
			f.log.Warn("Context cancelled in GetHistoryByPeriod")
			return nil
		default:
		}

		history, err := f.client.GetChatHistory(&client.GetChatHistoryRequest{
			ChatId:        chatID,
			FromMessageId: fromMessageID,
			Offset:        0,
			Limit:         50,
			OnlyLocal:     false,
		})
		if err != nil {
			f.totalErrors++
			f.log.Error("GetChatHistory failed", "chat_id", chatID, "err", err)
			return err
		}
		if len(history.Messages) == 0 || stop {
			f.log.Info("Reached end of history", "chat_id", chatID)
			return nil
		}

		for _, msg := range history.Messages {
			t := time.Unix(int64(msg.Date), 0)
			if t.After(to) {
				continue
			}
			if t.Before(from) {
				stop = true
				break
			}
			select {
			case <-ctx.Done():
				f.log.Warn("Context cancelled while sending message")
				return nil
			case out <- msg:
			}
		}

		fromMessageID = history.Messages[len(history.Messages)-1].Id
	}
}

func (f *TDLibFetcher) getMessageLink(chatID int64, messageID int64) (string, error) {
	req := &client.GetMessageLinkRequest{ChatId: chatID, MessageId: messageID}
	resp, err := f.client.GetMessageLink(req)
	if err != nil {
		return "", fmt.Errorf("GetMessageLink error: %w", err)
	}
	return resp.Link, nil
}

func (f *TDLibFetcher) ValidateMessage(raw *client.Message) (*model.Post, bool) {
	var text string
	switch content := raw.Content.(type) {
	case *client.MessageText:
		text = content.Text.Text
	case *client.MessagePhoto:
		text = content.Caption.Text
	case *client.MessageVideo:
		text = content.Caption.Text
	default:
		f.totalFiltered++
		f.log.Warn("Unsupported message content", "type", fmt.Sprintf("%T", raw.Content))
		return nil, false
	}

	text = strings.TrimSpace(text)
	if text == "" {
		f.totalFiltered++
		return nil, false
	}

	return &model.Post{
		ID:        raw.Id,
		Text:      text,
		Timestamp: time.Unix(int64(raw.Date), 0),
	}, true
}
