package contracts

import (
	"context"
	"time"

	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/model"
)

type PostFetcher interface {
	RunFetchPipelene(ctx context.Context, from, to time.Time) <-chan *model.Post
}

type PostAnalyzer interface {
	RunAnalyzePipeline(ctx context.Context, in <-chan *model.Post) <-chan *model.Post
}

type SaverPostgres interface {
	SaveBatch(ctx context.Context, in <-chan *model.Post) error
	GetMinMaxTimestamps(ctx context.Context) (min time.Time, max time.Time, ok bool, err error)
	GetPostsByPeriod(ctx context.Context, from, to time.Time) ([]*model.Post, error)
}

type Reporter interface {
	GenerateFullReport(ctx context.Context, from, to time.Time) error
}
