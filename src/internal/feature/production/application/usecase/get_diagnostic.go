package usecase

import (
	"context"

	"kaoru/internal/feature/production/domain"
)

// GetDiagnostic devuelve un DIAG por su ID legible. Leer no exige permiso.
type GetDiagnostic struct {
	d Deps
}

// GetDiagnosticInput es la entrada de GetDiagnostic.
type GetDiagnosticInput struct {
	Code string
}

// Execute devuelve ErrDiagnosticNotFound si el DIAG no existe.
func (uc *GetDiagnostic) Execute(ctx context.Context, in GetDiagnosticInput) (domain.DiagnosticSnapshot, error) {
	diagnostic, err := findDiagnostic(ctx, uc.d.Repository, in.Code)
	return diagnostic.Snapshot(), err
}
