package usecase

import (
	"context"

	shared "kaoru/internal/shared/domain"
)

// DeleteView quita una vista de quien la guardó.
type DeleteView struct {
	d Deps
}

// Execute devuelve NOT_FOUND_RESOURCE si la vista no existe o es de otra persona.
func (uc *DeleteView) Execute(ctx context.Context, owner shared.UserId, id string) error {
	return uc.d.Repository.Delete(ctx, owner, id)
}
