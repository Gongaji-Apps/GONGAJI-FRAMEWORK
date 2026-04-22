package logger

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

func New() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *Logger) Info(msg string) {
	l.Println(`{"level":"INFO","message":"` + msg + `"}`)
}

func (l *Logger) Error(msg string) {
	l.Println(`{"level":"ERROR","message":"` + msg + `"}`)
}
