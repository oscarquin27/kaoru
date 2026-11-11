package usecase

import (
	"context"

	"kaoru/internal/feature/audit/domain"
	shared "kaoru/internal/shared/domain"
)

// Record agrega un registro a la cadena. Lo llaman los demás features dentro de su
// transacción: si la auditoría falla, la operación no se guarda.
type Record struct {
	d Deps
}

// RecordInput es un hecho a auditar. La fecha la pone el servidor.
type RecordInput struct {
	ActorId    shared.UserId
	EntityType string
	EntityId   string
	EntityCode string
	Action     domain.Action
	Field      string
	OldValue   string
	NewValue   string
	Reason     string
}

// Execute encadena y guarda el registro.
func (uc *Record) Execute(ctx context.Context, in RecordInput) error {
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return err
	}
	entry, err := domain.NewEntry(id, uc.d.Clock.Now(), in.ActorId, in.EntityType, in.EntityId, in.EntityCode,
		in.Action, in.Field, in.OldValue, in.NewValue, in.Reason)
	if err != nil {
		return err
	}
	return uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		if err := uc.d.Repository.LockChain(ctx); err != nil {
			return err
		}
		prev, err := uc.d.Repository.LastHash(ctx)
		if err != nil {
			return err
		}
		return uc.d.Repository.Insert(ctx, entry.Chained(prev))
	})
}
