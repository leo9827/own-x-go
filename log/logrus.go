package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type Logrus struct {
	std *logrus.Logger
	mu  sync.Mutex
}

var DefaultLogger *Logrus

func New() *Logrus {

	l := &Logrus{
		std: logrus.New(),
	}
	l.SetLevel("debug")
	if DefaultLogger == nil {
		DefaultLogger = l
	}

	return l
}

func (l *Logrus) DecorateRuntimeContext(skip int) *logrus.Entry {
	if pc, file, line, ok := runtime.Caller(skip); ok {
		fName := runtime.FuncForPC(pc).Name()
		path := strings.Split(file, string(os.PathSeparator))
		var position string
		if len(path) > 3 {
			position = fmt.Sprintf("%s:%d", strings.Join(path[len(path)-3:], string(os.PathSeparator)), line)
		} else {
			position = fmt.Sprintf("%s:%d", strings.Join(path, string(os.PathSeparator)), line)
		}
		return l.std.WithField("position", position).WithField("func", fName)
	} else {
		return logrus.NewEntry(l.std)
	}
}

func (l *Logrus) Trace(format string, v ...interface{}) {
	l.DecorateRuntimeContext(2).Tracef(format, v...)
}

func (l *Logrus) Debug(format string, v ...interface{}) {
	l.DecorateRuntimeContext(2).Debugf(format, v...)
}

func (l *Logrus) Info(format string, v ...interface{}) {
	l.DecorateRuntimeContext(2).Infof(format, v...)
}

func (l *Logrus) Warn(format string, v ...interface{}) {
	l.DecorateRuntimeContext(2).Warnf(format, v...)
}

func (l *Logrus) Error(format string, v ...interface{}) {
	l.DecorateRuntimeContext(2).Errorf(format, v...)
}

func (l *Logrus) Fatal(format string, v ...interface{}) {
	l.DecorateRuntimeContext(2).Fatalf(format, v...)
}

func (l *Logrus) Panic(format string, v ...interface{}) {
	l.DecorateRuntimeContext(2).Panicf(format, v...)
}

func (l *Logrus) SetOutput(out io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.std.Out = out
}

func (l *Logrus) SetReportCaller(b bool) {
	l.std.SetReportCaller(b)
}

func (l *Logrus) GetOutput() io.Writer {
	if l.std != nil && l.std.Out != nil {
		return l.std.Out
	}
	return nil
}

func (l *Logrus) GetLevel() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return int(l.std.Level)
}

func (l *Logrus) setLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.std.Level = logrus.Level(level)
}

func (l *Logrus) SetLevel(level string) {
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

func (l *Logrus) SetFormatter(formatter logrus.Formatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.std.Formatter = formatter
}
