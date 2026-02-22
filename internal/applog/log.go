package applog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	AppLogFileName  = "v2n-coremesh.log"
	XrayLogFileName = "xray.log"
)

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func New(confDir string) (*Logger, error) {
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	path := filepath.Join(confDir, AppLogFileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open app log: %w", err)
	}
	return &Logger{file: f}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) Printf(format string, args ...any) {
	if l == nil || l.file == nil {
		return
	}
	line := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(l.file, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), line)
}

func (l *Logger) Writer() *os.File {
	if l == nil {
		return nil
	}
	return l.file
}

func OpenXrayLog(confDir string) (*os.File, error) {
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	path := filepath.Join(confDir, XrayLogFileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open xray log: %w", err)
	}
	return f, nil
}
