package bootstrap

import (
	"context"
	"io"
	"log/slog"

	"kaoru/internal/core/pkg/logging"
)

// newLogger crea el logger concreto: JSON en contenedores, texto legible en desarrollo.
func newLogger(w io.Writer, format string) *slog.Logger {
	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(w, nil)
	} else {
		handler = slog.NewJSONHandler(w, nil)
	}
	return slog.New(contextHandler{Handler: handler})
}

// contextHandler agrega a cada línea los campos de la petición que viajan en el context.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := logging.FieldsFrom(ctx)
	if fields.TraceID != "" {
		record.AddAttrs(slog.String("trace_id", fields.TraceID))
	}
	if fields.UserSub != "" {
		record.AddAttrs(slog.String("user_sub", fields.UserSub))
	}
	return h.Handler.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name)}
}
