package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel defines the severity level.
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

var levelNames = map[LogLevel]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

// Logger provides simple structured logging with dev-friendly output.
type Logger struct {
	level     LogLevel
	out       io.Writer
	devMode   bool
	mu        sync.Mutex
	devLogs   []string // Buffer for dev logs
	maxLogs   int
}

// NewLogger creates a new logger instance.
func NewLogger(level LogLevel, devMode bool) *Logger {
	return &Logger{
		level:   level,
		out:     os.Stdout,
		devMode: devMode,
		devLogs: make([]string, 0, 100),
		maxLogs: 100,
	}
}

// Debug logs a debug-level message.
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.log(DebugLevel, msg, keysAndValues...)
}

// Info logs an info-level message.
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.log(InfoLevel, msg, keysAndValues...)
}

// Warn logs a warning-level message.
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.log(WarnLevel, msg, keysAndValues...)
}

// Error logs an error-level message.
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.log(ErrorLevel, msg, keysAndValues...)
}

// log formats and outputs a log message.
func (l *Logger) log(level LogLevel, msg string, keysAndValues ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelName := levelNames[level]

	// Format key-value pairs
	kvStr := ""
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			kvStr += fmt.Sprintf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
		}
	}

	output := fmt.Sprintf("[%s] %s: %s%s\n", timestamp, levelName, msg, kvStr)
	fmt.Fprint(l.out, output)

	// Buffer for dev logs
	if l.devMode {
		logEntry := fmt.Sprintf("%s %s: %s%s", levelName, timestamp, msg, kvStr)
		l.devLogs = append(l.devLogs, logEntry)
		if len(l.devLogs) > l.maxLogs {
			l.devLogs = l.devLogs[1:]
		}
	}
}

// GetDevLogs returns buffered dev logs (most recent).
func (l *Logger) GetDevLogs(count int) []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	if count > len(l.devLogs) {
		count = len(l.devLogs)
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = l.devLogs[len(l.devLogs)-count+i]
	}
	return result
}

// ClearDevLogs clears the dev log buffer.
func (l *Logger) ClearDevLogs() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.devLogs = l.devLogs[:0]
}

// parseLevelString converts string to LogLevel.
func ParseLevelString(s string) LogLevel {
	switch s {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}
