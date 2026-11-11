package usecase

import (
	"context"

	"kaoru/internal/feature/defect/application/port"
)

// maxOpenDefects acota cuántos defectos abiertos de un REQ se leen de una vez.
const maxOpenDefects = 10000

// ListOpenDefects devuelve los DEF de un REQ que no están en un estado final.
// Lo usa requirement para sus reglas de bloqueo.
type ListOpenDefects struct {
	d Deps
}

// ListOpenDefectsInput es la entrada de ListOpenDefects.
type ListOpenDefectsInput struct {
	RequirementCode string
}

// OpenDefect es un defecto abierto, con lo mínimo para decidir si bloquea.
type OpenDefect struct {
	Code     string
	Severity string
}

// Execute lista los defectos abiertos. Leer no exige permiso.
func (uc *ListOpenDefects) Execute(ctx context.Context, in ListOpenDefectsInput) ([]OpenDefect, error) {
	defects, _, err := uc.d.Repository.List(ctx, port.Filter{
		RequirementCode: in.RequirementCode, OnlyOpen: true, Page: 1, Size: maxOpenDefects,
	})
	if err != nil {
		return nil, err
	}
	out := make([]OpenDefect, len(defects))
	for i, defect := range defects {
		s := defect.Snapshot()
		out[i] = OpenDefect{Code: s.Code.String(), Severity: string(s.Severity)}
	}
	return out, nil
}
