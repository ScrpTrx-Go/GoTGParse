package application

import (
	"context"
	"time"

	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/contracts"
	pkg "github.com/ScrpTrx-Go/GoTGParse/pkg/logger"
)

type App struct {
	Fetcher  contracts.PostFetcher
	Analyzer contracts.PostAnalyzer
	Logger   pkg.Logger
	Db       contracts.SaverPostgres
	Reporter contracts.Reporter
}

func NewApp(fetcher contracts.PostFetcher, analyzer contracts.PostAnalyzer, logger pkg.Logger, db contracts.SaverPostgres, reporter contracts.Reporter) *App {
	return &App{
		Fetcher:  fetcher,
		Analyzer: analyzer,
		Logger:   logger,
		Db:       db,
		Reporter: reporter,
	}
}

func (a *App) Run(ctx context.Context, from, to time.Time) {
	min, max, ok, err := a.Db.GetMinMaxTimestamps(ctx)
	if err != nil {
		a.Logger.Error("Failed to get DB timestamps", "err", err)
		return
	}

	if !ok {
		a.Logger.Warn("Database is empty, fetching all posts", "from", from, "to", to)
		a.fetchAndSave(ctx, from, to)
		return
	}

	if from.Before(min) {
		newTo := min.Add(-time.Nanosecond)
		a.Logger.Info("Loading older posts", "from", from, "to", newTo)
		a.fetchAndSave(ctx, from, newTo)
	}

	if to.After(max) {
		newFrom := max.Add(time.Nanosecond)
		a.Logger.Info("Loading newer posts", "from", newFrom, "to", to)
		a.fetchAndSave(ctx, newFrom, to)
	}
	if err := a.Reporter.GenerateFullReport(ctx, from, to); err != nil {
		a.Logger.Error("Failed to Generate report", "err", err)
		return
	}
}

func (a *App) fetchAndSave(ctx context.Context, from, to time.Time) {
	outFromFetch := a.Fetcher.RunFetchPipelene(ctx, from, to)
	outFromAnalyze := a.Analyzer.RunAnalyzePipeline(ctx, outFromFetch)

	if err := a.Db.SaveBatch(ctx, outFromAnalyze); err != nil {
		a.Logger.Error("Failed to save posts", "err", err)
	}
}
