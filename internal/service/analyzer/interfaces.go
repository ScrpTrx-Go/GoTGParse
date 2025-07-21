package analyzer

import (
	"context"

	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/model"
)

type AnalyzePostWorker interface {
	Run(ctx context.Context, in <-chan *model.Post, out chan<- *model.Post)
}

type MatchersCreator interface {
	CreateMatchers() Matchers
}

type DictionariesCreator interface {
	CreateDictionaries() *Dictionaries
}
