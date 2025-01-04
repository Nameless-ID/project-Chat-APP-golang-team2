package log

import (
	"auth-service/config"
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitZapLogger(cfg config.Config) (*zap.Logger, error) {
	logLevel := zap.InfoLevel
	if cfg.AppDebug {
		logLevel = zap.DebugLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		MessageKey:     "M",
		CallerKey:      "C",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	file, err := os.Create("../app.log")
	if err != nil {
		log.Panicf("Failed to open log file: %v", err)
		return nil, err
	}

	core := zapcore.NewTee(

		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(file),
			logLevel,
		),

		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			logLevel,
		),
	)

	logger := zap.New(core)
	logger.Info("Logger initialized successfully")

	return logger, nil
}
