// Package logging define el Logger que se inyecta y los campos que viajan en el context.
// El logger concreto se crea en bootstrap.
package logging

import "context"

// Logger es lo que usan los módulos. *slog.Logger lo cumple.
type Logger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type fieldsKey struct{}

// Fields son los campos de la petición que acompañan a cada línea de log.
type Fields struct {
	TraceID string
	UserSub string
}

// WithFields guarda los campos en el context.
func WithFields(ctx context.Context, f Fields) context.Context {
	return context.WithValue(ctx, fieldsKey{}, f)
}

// FieldsFrom lee los campos del context.
func FieldsFrom(ctx context.Context) Fields {
	f, _ := ctx.Value(fieldsKey{}).(Fields)
	return f
}
