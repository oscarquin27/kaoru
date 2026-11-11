package usecase

import (
	"context"

	"kaoru/internal/feature/execution/domain"
)

// GetExploration devuelve un EXP por su ID legible. Leer no exige permiso.
type GetExploration struct {
	d Deps
}

// GetExplorationInput es la entrada de GetExploration.
type GetExplorationInput struct {
	Code string
}

// Execute busca el EXP.
func (uc *GetExploration) Execute(ctx context.Context, in GetExplorationInput) (domain.ExplorationSnapshot, error) {
	exploration, err := findExploration(ctx, uc.d.Repository, in.Code)
	return exploration.Snapshot(), err
}
