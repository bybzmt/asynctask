package server

import (
	"context"
	"log/slog"
	"log/syslog"
)

type syslogWriter struct {
	log   *syslog.Writer
	level slog.Level
}

func newSyslogWriter(w *syslog.Writer) *syslogWriter {
	return &syslogWriter{
		log: w,
	}
}

func (h *syslogWriter) Write(p []byte) (n int, err error) {
	switch h.level {
	case slog.LevelDebug:
		err = h.log.Debug(string(p))
	case slog.LevelInfo:
		err = h.log.Info(string(p))
	case slog.LevelWarn:
		err = h.log.Warning(string(p))
	case slog.LevelError:
		err = h.log.Err(string(p))
	default:
		if h.level.Level() < slog.LevelDebug.Level() {
			err = h.log.Debug(string(p))
		} else if h.level.Level() > slog.LevelError.Level() {
			err = h.log.Err(string(p))
		} else {
			err = h.log.Info(string(p))
		}
	}

	return len(p), err
}

func newSyslogHandler(w *syslogWriter, h slog.Handler) slog.Handler {
	return &syslogHandler{
		writer:  w,
		handler: h,
	}
}

type syslogHandler struct {
	writer  *syslogWriter
	handler slog.Handler
}

func (h *syslogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *syslogHandler) Handle(ctx context.Context, record slog.Record) error {
	h.writer.level = record.Level
	return h.handler.Handle(ctx, record)
}

func (h *syslogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &syslogHandler{
		writer:  h.writer,
		handler: h.handler.WithAttrs(attrs),
	}
}

func (h *syslogHandler) WithGroup(name string) slog.Handler {
	return &syslogHandler{
		writer:  h.writer,
		handler: h.handler.WithGroup(name),
	}
}

type SyslogOptions struct {
	Network  string
	Addr     string
	Priority syslog.Priority
	Tag      string
	Fmt      string
	Level    slog.Leveler
	WithTime bool
}

func NewSyslog(o *SyslogOptions) (*slog.Logger, error) {
	sw, err := syslog.Dial(o.Network, o.Addr, o.Priority, o.Tag)
	if err != nil {
		return nil, err
	}

	w2 := newSyslogWriter(sw)

	var h slog.Handler

	ho := &slog.HandlerOptions{
		Level: o.Level,
	}

	if !o.WithTime {
		ho.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && (a.Key == "time" || a.Key == "level") {
				return slog.Attr{}
			}
			return a
		}
	}

	if o.Fmt == "json" {
		h = slog.NewJSONHandler(w2, ho)
	} else {
		h = slog.NewTextHandler(w2, ho)
	}

	h2 := newSyslogHandler(w2, h)

	return slog.New(h2), nil
}
