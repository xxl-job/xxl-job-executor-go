package xxl

import (
	"fmt"
	"log/slog"
)

// LogFunc 应用日志
type LogFunc func(req LogReq, res *LogRes) []byte

// Logger 系统日志
type Logger interface {
	Debug(format string, a ...interface{})
	Info(format string, a ...interface{})
	Error(format string, a ...interface{})
}

type logger struct {
}

func (l *logger) Debug(format string, a ...interface{}) {
	slog.Debug(fmt.Sprintf(format, a...))
}

func (l *logger) Info(format string, a ...interface{}) {
	slog.Info(fmt.Sprintf(format, a...))
}

func (l *logger) Error(format string, a ...interface{}) {
	slog.Error(fmt.Sprintf(format, a...))
}
