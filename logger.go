package logger

import (
	"sync"
)

type Logger interface {
	SetLevel(level Level)
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
	Panic(message string, args ...any)
	isEnable(level Level) bool
}

type Level uint8

func Parse(levelString string) (Level, bool) {
	for k, v := range logLevelString {
		if v == levelString {
			return k, true
		}
	}
	return INFO, false
}

func (level Level) ToString() (levelString string) {
	if value, ok := logLevelString[level]; ok {
		levelString = value
	}
	return
}

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	PANIC
)

var (
	once                  = sync.Once{}
	currentLogger  Logger = NewSimpleLogger("Unnamed.log")
	logLevelString        = map[Level]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		PANIC: "PANIC",
	}
)

func SetLogger(logger Logger) {
	once.Do(func() {
		currentLogger = logger
	})
}

func SetLevel(level Level) {
	currentLogger.SetLevel(level)
}

func Debug(message string, args ...any) {
	currentLogger.Debug(message, args...)
}

func Info(message string, args ...any) {
	currentLogger.Info(message, args...)
}

func Warn(message string, args ...any) {
	currentLogger.Warn(message, args...)
}

func Error(message string, args ...any) {
	currentLogger.Error(message, args...)
}

func Panic(message string, args ...any) {
	currentLogger.Panic(message, args...)
}
