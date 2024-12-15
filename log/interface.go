package log

import (
	"context"
	"io"
)

const (
	LevelPanic = iota
	LevelFatal
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
)

type level string

const (
	DebugLevel level = "debug"
	InfoLevel  level = "info"
	WarnLevel  level = "warn"
	ErrorLevel level = "error"
	FatalLevel level = "fatal"
	PanicLevel level = "panic"
)

type Logger interface {
	Trace(format string, v ...interface{})

	Debug(format string, v ...interface{})

	Info(format string, v ...interface{})

	Warn(format string, v ...interface{})

	Error(format string, v ...interface{})

	Fatal(format string, v ...interface{})

	Panic(format string, v ...interface{})

	SetLevel(level string)

	GetLevel() int

	SetOutput(out io.Writer)

	GetOutput() io.Writer
}
