package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type LogWriter struct{}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		Get().log(Info, "System", msg)
	}
	return len(p), nil
}

func SetupGlobalLogHook() {
	mw := io.MultiWriter(os.Stderr, &LogWriter{})
	log.SetOutput(mw)
}

type Level string

const (
	Info  Level = "INFO"
	Warn  Level = "WARN"
	Error Level = "ERROR"
)

type Entry struct {
	Time     string `json:"time"`
	Level    Level  `json:"level"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

type Logger struct {
	mu      sync.RWMutex
	entries []Entry
	maxSize int
}

var (
	global *Logger
	once   sync.Once
)

func Get() *Logger {
	once.Do(func() {
		global = &Logger{
			entries: make([]Entry, 0, 200),
			maxSize: 200,
		}
	})
	return global
}

func (l *Logger) log(level Level, category, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, Entry{
		Time:     time.Now().Format("2006-01-02 15:04:05"),
		Level:    level,
		Category: category,
		Message:  msg,
	})
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}
}

func (l *Logger) Infof(category, format string, args ...interface{}) {
	l.log(Info, category, fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(category, format string, args ...interface{}) {
	l.log(Warn, category, fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(category, format string, args ...interface{}) {
	l.log(Error, category, fmt.Sprintf(format, args...))
}

func (l *Logger) GetEntries(limit int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	total := len(l.entries)
	if limit <= 0 || limit > total {
		limit = total
	}
	result := make([]Entry, limit)
	copy(result, l.entries[total-limit:])
	return result
}

func (l *Logger) GetEntriesByCategory(category string, limit int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var filtered []Entry
	for i := len(l.entries) - 1; i >= 0 && (limit == 0 || len(filtered) < limit); i-- {
		if l.entries[i].Category == category {
			filtered = append([]Entry{l.entries[i]}, filtered...)
		}
	}
	return filtered
}

func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
}
