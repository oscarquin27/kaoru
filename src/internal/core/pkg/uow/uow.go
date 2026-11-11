// Package uow define la unidad de trabajo que usan los casos de uso.
package uow

import "context"

// Factory abre una transacción o reutiliza la del context.
// Dentro de fn solo se usa el ctx que recibe fn.
type Factory interface {
	Within(ctx context.Context, fn func(ctx context.Context) error) error
}
