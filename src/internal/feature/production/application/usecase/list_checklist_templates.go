package usecase

import (
	"context"

	"kaoru/internal/feature/production/domain"
)

// ListChecklistTemplates devuelve las versiones de la plantilla, la más reciente de cada sistema primero.
type ListChecklistTemplates struct {
	d Deps
}

// ListChecklistTemplatesInput es la entrada de ListChecklistTemplates. SystemKey vacío no filtra.
type ListChecklistTemplatesInput struct {
	SystemKey string
}

// Execute lista las versiones. Leer no exige permiso.
func (uc *ListChecklistTemplates) Execute(ctx context.Context, in ListChecklistTemplatesInput) ([]domain.TemplateSnapshot, error) {
	templates, err := uc.d.Repository.ListTemplates(ctx, in.SystemKey)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TemplateSnapshot, len(templates))
	for i, t := range templates {
		out[i] = t.Snapshot()
	}
	return out, nil
}
