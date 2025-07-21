package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ScrpTrx-Go/GoTGParse/application"
	"github.com/ScrpTrx-Go/GoTGParse/internal/config"
	"github.com/ScrpTrx-Go/GoTGParse/internal/infra/database"
	fetcher "github.com/ScrpTrx-Go/GoTGParse/internal/infra/telegram"
	"github.com/ScrpTrx-Go/GoTGParse/internal/service/analyzer"
	"github.com/ScrpTrx-Go/GoTGParse/internal/service/reporter"
	pkg "github.com/ScrpTrx-Go/GoTGParse/pkg/logger"
)

func main() {
	config, err := config.LoadConfig("./internal/config/config.yaml")
	if err != nil {
		log.Fatalf("error load config %v", err)
	}

	zaplogger, err := pkg.NewZapLogger(config.Logger)
	if err != nil {
		log.Fatalf("error initialize logger: %v", err)
	}

	if zaplogger != nil {
		defer zaplogger.Sync()
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	tdlibclient, err := fetcher.NewClient(config.TDLib)
	if err != nil {
		zaplogger.Error("tdlibclient", "init error", err)
		return
	}
	defer func() {
		_, err = tdlibclient.Close()
		if err != nil {
			zaplogger.Error("tdlibclient", "close error", err)
		}
	}()

	tdlibFetcher, err := fetcher.NewTDLibFetcher(tdlibclient, zaplogger, config.TDLib)
	if err != nil {
		zaplogger.Error("tdlibfetcher init error", err)
		return
	}

	dictCreator := analyzer.NewDictionariesCreator()
	dictionaries := dictCreator.CreateDictionaries()
	regions := analyzer.GetRegionKeys(dictionaries.RegionsAllias)

	workers := make([]analyzer.AnalyzePostWorker, 0, 5)
	for i := 0; i < 5; i++ {
		matchCreator := analyzer.NewMatcherCreator(dictionaries, regions)
		worker := analyzer.NewAnalyzeWorker(matchCreator, zaplogger, regions, *dictionaries)
		workers = append(workers, worker)
	}

	postPipeline := analyzer.NewPostPipeline(zaplogger, workers)

	db, err := database.NewPostgresPool(zaplogger, config.DatabaseConfig)
	if err != nil {
		zaplogger.Error("failed to init DB", "err", err)
		return
	}
	defer db.Pool.Close()

	newReporter := reporter.NewReporter(zaplogger, db)

	from := time.Date(2025, time.July, 21, 0, 0, 0, 0, time.Local)
	to := time.Date(2025, time.July, 22, 0, 0, 0, 0, time.Local)

	app := application.NewApp(tdlibFetcher, postPipeline, zaplogger, db, newReporter)

	app.Run(ctx, from, to)
}
