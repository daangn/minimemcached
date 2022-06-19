package minimemcached

import "fmt"

type LogLevel string

const (
	All   LogLevel = "all"
	Info  LogLevel = "info"
	Debug LogLevel = "debug"
	Off   LogLevel = "off"
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
	fmt.Println(str)
}
