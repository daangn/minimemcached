package minimemcached

import (
	"fmt"
	"time"
)

type LogLevel string

const (
	// Info is the default LogLevel of Logger.
	Info LogLevel = "info"
	// Debug helps you debug mini-memcached usages,
	//and logs all command and its result after being processed.
	Debug LogLevel = "debug"
	// Off does not log anything.
	Off LogLevel = "off"
)

// Logger logs certain messages depending on the LogLevel.
// TODO:(@sang-w0o): Selectively accept where log output will go to.
type Logger struct {
	Level LogLevel
}

// newLogger returns a Logger.
func newLogger(level LogLevel) *Logger {
	l := Logger{
		Level: level,
	}
	return &l
}

// Println writes given string to log file.
func (l *Logger) Println(str string) {
	s := fmt.Sprintf("{\"level: \"%s\", \"time\":\"%v\", \"message\":\"%s\"}", l.Level, time.Now(), str)
	fmt.Println(s)
}
