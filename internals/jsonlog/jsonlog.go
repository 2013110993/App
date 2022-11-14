// Filename: internal/jsonlog/jsonlog.go

package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// having different severity levels of logging entries
type Level int8

// Level start at zero
const (
	LevelInfo  Level = iota // value is 0
	LevelError              // values is 1
	LevelFatal              // values is 2
	LevelOff                // values is 3
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""

	}
}

// Define a custom logger
type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
}

// New() function creates a new instance of Logger
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}

}

// Helper methods
func (l *Logger) PrintInfo(message string, properties map[string]string) {
	l.print(LevelInfo, message, properties)
}

func (l *Logger) PrintError(err error, properties map[string]string) {
	l.print(LevelError, err.Error(), properties)
}

func (l *Logger) PrintFatal(err error, properties map[string]string) {
	l.print(LevelFatal, err.Error(), properties)
	os.Exit(1)
}

func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
	// ensure severity level is at least the minimum level
	if level < l.minLevel {
		return 0, nil
	}
	data := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}
	// should we include the stack trace?
	if level >= LevelError {
		data.Trace = string(debug.Stack())
	}
	//encode the log entry Json
	var entry []byte
	entry, err := json.Marshal(data)

	if err != nil {
		entry = []byte(LevelError.String() + ": unable to marshal log entry" + err.Error())
	}

	// Prepare to write the log entry
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.out.Write(append(entry, '\n'))

}

// implement the io.Writer interface
func (l *Logger) Write(message []byte) (n int, err error) {
	return l.print(LevelError, string(message), nil)

}
