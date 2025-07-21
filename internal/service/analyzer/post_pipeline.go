package analyzer

import (
	"context"
	"sync"

	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/model"
	pkg "github.com/ScrpTrx-Go/GoTGParse/pkg/logger"
)

type PostPipeline struct {
	Log     pkg.Logger
	Workers []AnalyzePostWorker
}

func NewPostPipeline(log pkg.Logger, workers []AnalyzePostWorker) *PostPipeline {
	return &PostPipeline{
		Log:     log,
		Workers: workers,
	}
}

func (p *PostPipeline) RunAnalyzePipeline(ctx context.Context, in <-chan *model.Post) <-chan *model.Post {
	out := make(chan *model.Post)
	var wg sync.WaitGroup

	for _, worker := range p.Workers {
		wg.Add(1)
		go func(w AnalyzePostWorker) {
			defer wg.Done()
			w.Run(ctx, in, out)
		}(worker)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
