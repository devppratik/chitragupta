package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Level represents log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	mu           sync.RWMutex
	currentLevel = LevelInfo
	output       io.Writer = os.Stderr
)

// SetLevel sets global log level
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

// SetOutput sets log output writer
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = w
}

// ParseLevel parses level string
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Debug logs debug message
func Debug(format string, args ...interface{}) {
	mu.RLock()
	level := currentLevel
	out := output
	mu.RUnlock()
	if level <= LevelDebug {
		log.New(out, "[DEBUG] ", log.LstdFlags).Printf(format, args...)
	}
}

// Info logs info message
func Info(format string, args ...interface{}) {
	mu.RLock()
	level := currentLevel
	out := output
	mu.RUnlock()
	if level <= LevelInfo {
		fmt.Fprintf(out, format+"\n", args...)
	}
}

// Warn logs warning message
func Warn(format string, args ...interface{}) {
	mu.RLock()
	level := currentLevel
	out := output
	mu.RUnlock()
	if level <= LevelWarn {
		fmt.Fprintf(out, "⚠️  "+format+"\n", args...)
	}
}

// Error logs error message
func Error(format string, args ...interface{}) {
	mu.RLock()
	level := currentLevel
	out := output
	mu.RUnlock()
	if level <= LevelError {
		fmt.Fprintf(out, "✗ "+format+"\n", args...)
	}
}

// Fatal logs error and exits
func Fatal(format string, args ...interface{}) {
	mu.RLock()
	out := output
	mu.RUnlock()
	fmt.Fprintf(out, "✗ "+format+"\n", args...)
	os.Exit(1)
}

// Success logs success message
func Success(format string, args ...interface{}) {
	mu.RLock()
	level := currentLevel
	out := output
	mu.RUnlock()
	if level <= LevelInfo {
		fmt.Fprintf(out, "✓ "+format+"\n", args...)
	}
}
