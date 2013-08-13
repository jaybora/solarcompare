package logger

import (
	"log"
)

const (
	TRACE = iota
	DEBUG = iota
	INFO  = iota
	FAIL  = iota
)

type Logger struct {
	Level  int
	Module string
}

func NewLogger(level int, module string) Logger {
	return Logger{level, module}
}

func (l *Logger) Tracef(s string, v ...interface{}) {
	if l.Level == TRACE {
		log.Printf(l.Module+":TRACE: "+s, v...)
	}
}
func (l *Logger) Trace(s string) {
	if l.Level == TRACE {
		log.Print(l.Module + ":TRACE: " + s)
	}
}
func (l *Logger) Debugf(s string, v ...interface{}) {
	if l.Level <= DEBUG {
		log.Printf(l.Module+":DEBUG: "+s, v...)
	}
}
func (l *Logger) Debug(s string) {
	if l.Level <= DEBUG {
		log.Print(l.Module + ":DEBUG: " + s)
	}
}
func (l *Logger) Infof(s string, v ...interface{}) {
	if l.Level <= INFO {
		log.Printf(l.Module+":INFO: "+s, v...)
	}
}
func (l *Logger) Info(s string) {
	if l.Level <= INFO {
		log.Print(l.Module + ":INFO: " + s)
	}
}
func (l *Logger) Failf(s string, v ...interface{}) {
	if l.Level <= FAIL {
		log.Printf(l.Module+":FAIL: "+s, v...)
	}
}
func (l *Logger) Fail(s string) {
	if l.Level <= FAIL {
		log.Print(l.Module + ":FAIL: " + s)
	}
}
