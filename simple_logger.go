package logger

import (
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	"os"
	"os/signal"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	reset                  = "\033[0m"
	red                    = "\033[91m"
	green                  = "\033[32m"
	yellow                 = "\033[33m"
	purple                 = "\033[35m"
	consoleLogFormat       = "%s %s%5s" + reset + " " + purple + "%4s" + reset + " " + green + "%-30s" + reset + " : %s"
	logFormat              = "%s %5s %4s %-30s : %s"
	logMaxAge              = 30 * 24 * time.Hour
	logRotationTime        = 24 * time.Hour
	maxMessageBufferLength = 100
	messageChanLength      = 10000
	flushTimeInterval      = time.Second
)

var colorRegex, _ = regexp.Compile("\u001B\\[.*?m")

type SimpleLogger struct {
	logLevel      Level
	logFilename   string
	logFile       *rotatelogs.RotateLogs
	messageBuffer []string
	messageChan   chan string
	signChan      chan os.Signal
}

func NewSimpleLogger(logFilename string) *SimpleLogger {
	logger := &SimpleLogger{
		logLevel:    INFO,
		logFilename: logFilename,
	}
	if logger.isFileLogger() {
		ext := path.Ext(logFilename)
		filename := strings.TrimSuffix(logFilename, ext)
		rotateLogs, err := rotatelogs.New(
			filename+".%Y-%m-%d"+ext,
			rotatelogs.WithMaxAge(logMaxAge),
			rotatelogs.WithRotationTime(logRotationTime),
		)
		if err != nil {
			panic(err)
		}
		logger.logFile = rotateLogs
		logger.messageBuffer = make([]string, 0, maxMessageBufferLength)
		logger.messageChan = make(chan string, messageChanLength)
		logger.signChan = make(chan os.Signal)
		go logger.listenFlush()
	}
	return logger
}

func (logger *SimpleLogger) SetLevel(level Level) {
	logger.logLevel = level
}

func (logger *SimpleLogger) Debug(message string, args ...any) {
	logger.Push(DEBUG, "", message, args...)
}

func (logger *SimpleLogger) Info(message string, args ...any) {
	logger.Push(INFO, "", message, args...)
}

func (logger *SimpleLogger) Warn(message string, args ...any) {
	logger.Push(WARN, "", message, args...)
}

func (logger *SimpleLogger) Error(message string, args ...any) {
	logger.Push(ERROR, "", message, args...)
}

func (logger *SimpleLogger) Panic(message string, args ...any) {
	logger.Push(PANIC, "", message, args...)
	if logger.isFileLogger() {
		time.Sleep(5 * time.Second)
		logger.signChan <- syscall.SIGQUIT
	} else {
		os.Exit(1)
	}
}

func (logger *SimpleLogger) Push(level Level, caller string, message string, args ...any) {
	if logger.isEnabled(level) {
		if len(args) > 0 {
			message = fmt.Sprintf(message, args...)
		}
		now := time.Now().Format("2006-01-02 15:04:05.000")
		pid := strconv.Itoa(os.Getpid())
		if caller == "" {
			_, file, line, _ := runtime.Caller(3)
			caller = fmt.Sprintf("%s:%d", file, line)
		}
		levelColor := green
		levelString := level.ToString()
		switch level {
		case DEBUG, INFO:
			levelColor = green
		case WARN:
			levelColor = yellow
		case ERROR, PANIC:
			levelColor = red
		}
		fmt.Printf(consoleLogFormat+"\n", now, levelColor, levelString, pid, caller, message)
		if logger.isFileLogger() {
			message = fmt.Sprintf(logFormat, now, levelString, pid, colorRegex.ReplaceAllString(caller, ""), colorRegex.ReplaceAllString(message, ""))
			logger.messageChan <- message
		}
	}
}

func (logger *SimpleLogger) isEnabled(level Level) bool {
	return logger.logLevel <= level
}

func (logger *SimpleLogger) isFileLogger() bool {
	return logger.logFilename != ""
}

func (logger *SimpleLogger) flush() {
	builder := strings.Builder{}
	for _, message := range logger.messageBuffer {
		builder.WriteString(message)
		builder.WriteString("\n")
	}
	content := builder.String()
	if content != "" {
		logger.messageBuffer = make([]string, 0, maxMessageBufferLength)
		if logger.logFile != nil {
			_, _ = logger.logFile.Write([]byte(content))
		}
	}
}

func (logger *SimpleLogger) listenFlush() {
	signal.Notify(logger.signChan, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	ticker := time.NewTicker(flushTimeInterval)
	for {
		select {
		case message := <-logger.messageChan:
			logger.messageBuffer = append(logger.messageBuffer, message)
			if len(logger.messageBuffer) == maxMessageBufferLength {
				logger.flush()
			}
		case <-ticker.C:
			logger.flush()
		case <-logger.signChan:
			logger.flush()
			os.Exit(1)
		}
	}
}
