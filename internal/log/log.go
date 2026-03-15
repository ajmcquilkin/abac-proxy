package log

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
)

type Fields map[string]any

type (
	Logger        = zap.SugaredLogger
	ContextLogger struct {
		*Logger
	}
)

var (
	defaultMu sync.RWMutex
	def       = zap.S()
)

type contextKey int

const (
	loggerKey contextKey = iota
)

func NewService(name string) (*Logger, error) {
	base, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	if name != "" {
		base = base.Named(name)
	}
	return base.Sugar(), nil
}

func MustInitService(name string) *Logger {
	l, err := NewService(name)
	if err != nil {
		panic(err)
	}
	SetDefault(l)
	return l
}

func Sync(l *Logger) error {
	if l == nil {
		return nil
	}
	err := l.Sync()
	if err == os.ErrInvalid {
		return nil
	}
	return err
}

func SetDefault(l *Logger) {
	if l == nil {
		return
	}
	defaultMu.Lock()
	defer defaultMu.Unlock()
	def = l
}

func defaultLogger() *Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return def
}

func WithContext(ctx context.Context, l *Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey, l)
}

func From(ctx context.Context) *ContextLogger {
	if ctx != nil {
		if l, ok := ctx.Value(loggerKey).(*Logger); ok && l != nil {
			return &ContextLogger{Logger: l}
		}
	}
	return &ContextLogger{Logger: defaultLogger()}
}

func (l *ContextLogger) With(fields Fields) *ContextLogger {
	if l == nil || l.Logger == nil {
		return &ContextLogger{Logger: defaultLogger()}
	}
	return &ContextLogger{Logger: l.Logger.With(fields.kv()...)}
}

func (f Fields) kv() []any {
	if len(f) == 0 {
		return nil
	}
	kv := make([]any, 0, len(f)*2)
	for k, v := range f {
		kv = append(kv, k, v)
	}
	return kv
}
