package usecase

import (
	"context"

	"kaoru/internal/feature/production/domain"
)

// ListDiagnostics devuelve una página de DIAG, del más reciente al más antiguo.
type ListDiagnostics struct {
	d Deps
}

// ListDiagnosticsOutput es una página de DIAG.
type ListDiagnosticsOutput struct {
	Diagnostics []domain.DiagnosticSnapshot
	Total       int
}

// Execute lista los DIAG. Leer no exige permiso.
func (uc *ListDiagnostics) Execute(ctx context.Context, in ListInput) (ListDiagnosticsOutput, error) {
	diagnostics, total, err := uc.d.Repository.ListDiagnostics(ctx, in.filter())
	if err != nil {
		return ListDiagnosticsOutput{}, err
	}
	out := ListDiagnosticsOutput{Diagnostics: make([]domain.DiagnosticSnapshot, len(diagnostics)), Total: total}
	for i, d := range diagnostics {
		out.Diagnostics[i] = d.Snapshot()
	}
	return out, nil
}
