package logging

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

const logBufferCapacity = 200

type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type LogBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	pos     int
	full    bool
}

func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, logBufferCapacity),
	}
}

func (b *LogBuffer) Append(entry LogEntry) {
	b.mu.Lock()
	b.entries[b.pos] = entry
	b.pos = (b.pos + 1) % logBufferCapacity
	if b.pos == 0 {
		b.full = true
	}
	b.mu.Unlock()
}

func (b *LogBuffer) GetRecentLogs(level string, limit int) []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var count int
	if b.full {
		count = logBufferCapacity
	} else {
		count = b.pos
	}

	if limit <= 0 {
		limit = 50
	}

	result := make([]LogEntry, 0, limit)
	for i := 0; i < count && len(result) < limit; i++ {
		idx := (b.pos - 1 - i + logBufferCapacity) % logBufferCapacity
		entry := b.entries[idx]
		if level != "" && entry.Level != level {
			continue
		}
		result = append(result, entry)
	}

	return result
}

type bufferHandler struct {
	inner  slog.Handler
	buffer *LogBuffer
}

func (h *bufferHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *bufferHandler) Handle(ctx context.Context, rec slog.Record) error {
	h.buffer.Append(LogEntry{
		Time:    rec.Time,
		Level:   rec.Level.String(),
		Message: rec.Message,
	})
	return h.inner.Handle(ctx, rec)
}

func (h *bufferHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &bufferHandler{
		inner:  h.inner.WithAttrs(attrs),
		buffer: h.buffer,
	}
}

func (h *bufferHandler) WithGroup(name string) slog.Handler {
	return &bufferHandler{
		inner:  h.inner.WithGroup(name),
		buffer: h.buffer,
	}
}

func New() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func NewWithBuffer(buf *LogBuffer) *slog.Logger {
	inner := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(&bufferHandler{inner: inner, buffer: buf})
}

/** 将已有 logger 包装为带 buffer 的 logger，保留原始 handler 的输出能力 */
func WrapWithBuffer(logger *slog.Logger, buf *LogBuffer) *slog.Logger {
	return slog.New(&bufferHandler{inner: logger.Handler(), buffer: buf})
}
