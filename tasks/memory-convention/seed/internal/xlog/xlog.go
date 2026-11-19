// Package xlog writes one JSON object per line.
package xlog

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

var (
	mu  sync.Mutex
	out io.Writer = os.Stderr
)

// SetOutput redirects every logger; tests use it to capture lines.
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	out = w
}

// Logger tags every line with its component.
type Logger struct{ component string }

// For returns the logger of a component.
func For(component string) *Logger { return &Logger{component: component} }

func (l *Logger) Info(msg string, kv ...any)  { l.write("info", msg, kv) }
func (l *Logger) Warn(msg string, kv ...any)  { l.write("warn", msg, kv) }
func (l *Logger) Error(msg string, kv ...any) { l.write("error", msg, kv) }

func (l *Logger) write(level, msg string, kv []any) {
	line := map[string]any{"level": level, "component": l.component, "msg": msg}
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			line[k] = kv[i+1]
		}
	}
	b, _ := json.Marshal(line)
	mu.Lock()
	defer mu.Unlock()
	out.Write(append(b, '\n'))
}
