package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Logger struct {
	*slog.Logger
	runID   string
	service string
}

func New(level, service, runID string) *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.String("timestamp", attr.Value.Time().UTC().Format(time.RFC3339))
			}
			if attr.Key == slog.MessageKey {
				return slog.String("message", attr.Value.String())
			}
			if attr.Key == slog.LevelKey {
				return slog.String("level", attr.Value.String())
			}
			return attr
		},
	})
	base := slog.New(handler).With(
		slog.String("run_id", runID),
		slog.String("service", service),
	)
	return &Logger{Logger: base, runID: runID, service: service}
}

func (l *Logger) Component(component string) *slog.Logger {
	return l.With(slog.String("component", component))
}

func (l *Logger) WithRequest(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

type requestIDKey struct{}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
