package logging

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"blackboxv1/backend/config"
)

type Logger struct{}

func NewLogger(cfg config.Config) *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(&lumberjack.Logger{
				Filename: "./logs/app.log",
				MaxSize: 50,
				MaxBackups: 5,
				MaxAge: 14,
				Compress: false,
			}),
			zapcore.InfoLevel,
		),
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderCfg),
			zapcore.AddSync(zapcore.Lock(os.Stderr)),
			zapcore.InfoLevel,
		),
	)

	logger := zap.New(core)
	return logger
}

