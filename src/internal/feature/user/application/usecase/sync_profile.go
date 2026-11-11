package usecase

import (
	"context"

	"kaoru/internal/feature/user/domain"
)

// SyncProfile guarda el perfil del token para que el resto del sistema pueda
// referenciar al usuario (responsables, auditoría).
type SyncProfile struct {
	d Deps
}

// Execute crea o actualiza el perfil.
func (uc *SyncProfile) Execute(ctx context.Context, in ProfileInput) error {
	profile, err := domain.NewProfile(in.Id, in.Username, in.DisplayName, in.Email)
	if err != nil {
		return err
	}
	return uc.d.Repository.Upsert(ctx, profile, uc.d.Clock.Now())
}
