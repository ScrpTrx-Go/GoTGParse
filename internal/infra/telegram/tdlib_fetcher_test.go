package fetcher_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ScrpTrx-Go/GoTGParse/internal/config"
	fetcher "github.com/ScrpTrx-Go/GoTGParse/internal/infra/telegram"
	pkg "github.com/ScrpTrx-Go/GoTGParse/pkg/logger"
)

type Period struct {
	from time.Time
	to   time.Time
}

func getProjectPath() string {
	_, filename, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(filename)
	projectDir := filepath.Join(currentDir, "..", "..", "..")
	return projectDir
}
func TestFetch(t *testing.T) {
	dir := getProjectPath()
	configPath := filepath.Join(dir, "internal", "config", "config.yaml")
	loggerPath := filepath.Join(dir, "logs", "fetcher_test_logs")
	tdlibDBPath := filepath.Join(dir, "internal", "infra", "telegram", "data", "tdlib-db")
	tdlibFilesPath := filepath.Join(dir, "internal", "infra", "telegram", "data", "tdlib-files")

	config, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("error load config %v", err)
	}

	config.Logger.FilePath = loggerPath

	zaplogger, err := pkg.NewZapLogger(config.Logger)
	if err != nil {
		t.Fatalf("Error initialize logger: %v", err)
	}
	if zaplogger != nil {
		defer zaplogger.Sync()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config.TDLib.DatabaseDirectory = tdlibDBPath
	config.TDLib.FilesDirectory = tdlibFilesPath

	tdlibclient, _ := fetcher.NewClient(config.TDLib)
	tdlibFetcher, err := fetcher.NewTDLibFetcher(tdlibclient, zaplogger, config.TDLib)
	if err != nil {
		zaplogger.Error("tdlibfetcher error", err)
	}

	sledcomMap := map[int]Period{
		0: {
			from: time.Date(2025, time.July, 15, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.July, 16, 0, 0, 0, 0, time.Local),
		},
		16: {
			from: time.Date(2025, time.June, 15, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.June, 16, 0, 0, 0, 0, time.Local),
		},
		29: {
			from: time.Date(2025, time.May, 22, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.May, 23, 0, 0, 0, 0, time.Local),
		},
		13: {
			from: time.Date(2025, time.April, 19, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.April, 20, 0, 0, 0, 0, time.Local),
		},
		23: {
			from: time.Date(2025, time.March, 10, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.March, 11, 0, 0, 0, 0, time.Local),
		},
		7: {
			from: time.Date(2025, time.February, 15, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.February, 16, 0, 0, 0, 0, time.Local),
		},
		10: {
			from: time.Date(2025, time.January, 2, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.January, 3, 0, 0, 0, 0, time.Local),
		},
	}

	informCentrMap := map[int]Period{
		0: {
			from: time.Date(2025, time.July, 15, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.July, 16, 0, 0, 0, 0, time.Local),
		},
		35: {
			from: time.Date(2025, time.June, 15, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.June, 16, 0, 0, 0, 0, time.Local),
		},
		47: {
			from: time.Date(2025, time.May, 22, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.May, 23, 0, 0, 0, 0, time.Local),
		},
		32: {
			from: time.Date(2025, time.April, 19, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.April, 20, 0, 0, 0, 0, time.Local),
		},
		36: {
			from: time.Date(2025, time.March, 10, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.March, 11, 0, 0, 0, 0, time.Local),
		},
		7: {
			from: time.Date(2025, time.February, 15, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.February, 16, 0, 0, 0, 0, time.Local),
		},
		27: {
			from: time.Date(2025, time.January, 2, 0, 0, 0, 0, time.Local),
			to:   time.Date(2025, time.January, 3, 0, 0, 0, 0, time.Local),
		},
	}
	chatNameSledcom := []string{"sledcom_press"}
	chatNameIC := []string{"infocentrskrf"}
	err = checkMessages(sledcomMap, chatNameSledcom, tdlibFetcher, ctx, zaplogger)
	if err != nil {
		t.Fatalf("checkMessages from Sledcom error: %v", err)
	}
	err = checkMessages(informCentrMap, chatNameIC, tdlibFetcher, ctx, zaplogger)
	if err != nil {
		t.Fatalf("checkMessages from informcentr error: %v", err)
	}
}

func checkMessages(periodAndCountMessages map[int]Period, chatName []string, f *fetcher.TDLibFetcher, ctx context.Context, logger *pkg.ZapLogger) error {
	repeats := make(map[string]struct{})
	for exceptedCount, period := range periodAndCountMessages {
		counter := 0
		out := f.RunFetchPipelene(ctx, period.from, period.to)

		for msg := range out {
			if _, exists := repeats[msg.Text]; exists {
				return fmt.Errorf("repeated text %s", msg.Text)
			}
			logger.Info("message", "text", msg.Text)
			repeats[msg.Text] = struct{}{}
			counter++
		}
		if counter != exceptedCount {
			return fmt.Errorf("excepted count and real count not equal. excepted: %v, got: %v", exceptedCount, counter)
		}
	}
	return nil
}
