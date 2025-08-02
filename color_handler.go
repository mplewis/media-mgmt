package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[37m"
	colorBold   = "\033[1m"
)

type ColorHandler struct {
	writer io.Writer
	opts   *slog.HandlerOptions
}

func NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ColorHandler{
		writer: w,
		opts:   opts,
	}
}

func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}

	levelColor := h.getLevelColor(r.Level)
	levelText := h.getLevelText(r.Level)

	timestamp := r.Time.Format("15:04:05")

	var attrs []string
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
		return true
	})

	var attrsText string
	if len(attrs) > 0 {
		attrsText = " " + colorGray + strings.Join(attrs, " ") + colorReset
	}

	message := fmt.Sprintf("%s[%s]%s %s%s %s%s%s\n",
		colorGray, timestamp, colorReset,
		levelColor, levelText, colorReset,
		r.Message, attrsText)

	_, err := h.writer.Write([]byte(message))
	return err
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *ColorHandler) getLevelColor(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return colorBlue
	case slog.LevelInfo:
		return colorReset
	case slog.LevelWarn:
		return colorYellow
	case slog.LevelError:
		return colorRed + colorBold
	default:
		return colorReset
	}
}

func (h *ColorHandler) getLevelText(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERRO"
	default:
		return level.String()
	}
}
