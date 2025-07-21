package pkg

import (
	"log"
	"os"
	"strings"

	"github.com/ScrpTrx-Go/GoTGParse/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	sugar *zap.SugaredLogger
}

func NewZapLogger(cfg config.LoggerConfig) (*ZapLogger, error) {

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(cfg.Level))); err != nil {
		log.Printf("Invalid log level %q: %v. Falling back to default level: Info", cfg.Level, err)
		level = zapcore.InfoLevel
	}

	baseEncoderCfg := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	consoleCfg := baseEncoderCfg
	consoleCfg.ConsoleSeparator = " | "

	fileCfg := baseEncoderCfg
	fileCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(consoleCfg)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)

	file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	fileEncoder := zapcore.NewJSONEncoder(fileCfg)
	fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(file), level)

	core := zapcore.NewTee(consoleCore, fileCore)
	logger := zap.New(core, zap.AddStacktrace(zapcore.ErrorLevel))

	return &ZapLogger{sugar: logger.Sugar()}, nil
}

func (l *ZapLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

func (l *ZapLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *ZapLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

func (l *ZapLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

func (l *ZapLogger) WithPackage(name string) Logger {
	return &ZapLogger{sugar: l.sugar.With("package", name)}
}

func (l *ZapLogger) Sync() error {
	return l.sugar.Sync()
}
