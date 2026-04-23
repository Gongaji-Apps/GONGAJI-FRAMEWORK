package logger

import (
	"context"
	"os"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func New() *Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	return &Logger{log}
}

func (l *Logger) WithCtx(ctx context.Context) *logrus.Entry {
	return l.WithFields(logrus.Fields{
		"request_id":     contextx.GetRequestID(ctx),
		"correlation_id": contextx.GetCorrelationID(ctx),
	})
}
