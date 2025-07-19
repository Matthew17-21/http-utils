package httputils

import "fmt"

type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}

type logger struct{}

func NewLogger() Logger {
	return &logger{}
}

func (n logger) Debug(_ string, _ ...any) {}

func (n logger) Info(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func (n logger) Warn(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func (n logger) Error(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

type silentLogger struct{}

func NewSilentLogger() Logger {
	return &silentLogger{}
}

func (n silentLogger) Debug(_ string, _ ...any) {}

func (n silentLogger) Info(_ string, _ ...any) {}

func (n silentLogger) Warn(_ string, _ ...any) {}

func (n silentLogger) Error(_ string, _ ...any) {}

type debugLogger struct {
	logger Logger
}

func NewDebugLogger(logger Logger) Logger {
	return &debugLogger{
		logger: logger,
	}
}

func (n debugLogger) Debug(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func (n debugLogger) Info(format string, args ...any) {
	n.logger.Info(format, args...)
}

func (n debugLogger) Warn(format string, args ...any) {
	n.logger.Warn(format, args...)
}

func (n debugLogger) Error(format string, args ...any) {
	n.logger.Error(format, args...)
}
