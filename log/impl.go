package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type LoggerImpl struct {
	mu sync.Mutex
	l  *logrus.Logger
}

var DefaultLogger *LoggerImpl
var defaultLoggerInit sync.Once

func New() *LoggerImpl {
	l := &LoggerImpl{
		l: logrus.New(),
	}
	l.SetLevel(string(DebugLevel))
	if DefaultLogger == nil {
		defaultLoggerInit.Do(func() {
			DefaultLogger = l
		})
	}
	return l
}

func (l *LoggerImpl) decorate(skip int) *logrus.Entry {
	if pc, file, line, ok := runtime.Caller(skip); ok {
		fName := runtime.FuncForPC(pc).Name()
		path := strings.Split(file, string(os.PathSeparator))
		var position string
		if len(path) > 3 {
			position = fmt.Sprintf("%s:%d", strings.Join(path[len(path)-3:], string(os.PathSeparator)), line)
		} else {
			position = fmt.Sprintf("%s:%d", strings.Join(path, string(os.PathSeparator)), line)
		}
		return l.l.WithField("position", position).WithField("func", fName)
	} else {
		return logrus.NewEntry(l.l)
	}
}

func (l *LoggerImpl) Trace(format string, v ...interface{}) {
	l.decorate(2).Tracef(format, v...)
}

func (l *LoggerImpl) Debug(format string, v ...interface{}) {
	l.decorate(2).Debugf(format, v...)
}

func (l *LoggerImpl) Info(format string, v ...interface{}) {
	l.decorate(2).Infof(format, v...)
}

func (l *LoggerImpl) Warn(format string, v ...interface{}) {
	l.decorate(2).Warnf(format, v...)
}

func (l *LoggerImpl) Error(format string, v ...interface{}) {
	l.decorate(2).Errorf(format, v...)
}

func (l *LoggerImpl) Fatal(format string, v ...interface{}) {
	l.decorate(2).Fatalf(format, v...)
}

func (l *LoggerImpl) Panic(format string, v ...interface{}) {
	l.decorate(2).Panicf(format, v...)
}

func (l *LoggerImpl) setLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Level = logrus.Level(level)
}

func (l *LoggerImpl) SetLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		l.setLevel(LevelDebug)
	case "info":
		l.setLevel(LevelInfo)
	case "warn":
		l.setLevel(LevelWarn)
	case "LevelError":
		l.setLevel(LevelError)
	default:
		l.setLevel(LevelInfo)
	}
}

func (l *LoggerImpl) GetLevel() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return int(l.l.Level)
}

func (l *LoggerImpl) SetOutput(out io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Out = out
}

func (l *LoggerImpl) SetReportCaller(b bool) {
	l.l.SetReportCaller(b)
}

func (l *LoggerImpl) GetOutput() io.Writer {
	if l.l != nil && l.l.Out != nil {
		return l.l.Out
	}
	return nil
}

func (l *LoggerImpl) SetFormatter(formatter logrus.Formatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Formatter = formatter
}
