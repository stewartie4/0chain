package zap

import (
	"0chain.net/core/logging"
	"0chain.net/internal/infrastrcture/service/logger"
	"go.uber.org/zap"
)

type zapLogger struct {
	zap *zap.SugaredLogger
}

func (l *zapLogger) Flush() error {
	return l.zap.Sync()
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.zap.Fatalf(format, args...)
}

func (l *zapLogger) Fatal(args ...interface{}) {
	l.zap.Fatal(args...)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.zap.Infof(format, args...)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.zap.Warnf(format, args...)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.zap.Debugf(format, args...)
}

func (l *zapLogger) Printf(format string, args ...interface{}) {
	l.zap.Infof(format, args...)
}

func (l *zapLogger) Println(args ...interface{}) {
	l.zap.Info(args, "\n")
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.zap.Errorf(format, args...)
}

// New return zap instance of logger
func New() logger.Logger {
	l := logging.Logger

	//l, _ := zap.NewProduction()

	return &zapLogger{
		zap: l.Sugar(),
	}
}
