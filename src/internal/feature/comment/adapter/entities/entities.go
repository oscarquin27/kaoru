// Package entities implementa el puerto Entities con los casos de uso de los features
// cuyas entidades admiten comentarios.
package entities

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	bundleuc "kaoru/internal/feature/bundle/application/usecase"
	changeuc "kaoru/internal/feature/change/application/usecase"
	"kaoru/internal/feature/comment/application/port"
	"kaoru/internal/feature/comment/domain"
	defectuc "kaoru/internal/feature/defect/application/usecase"
	requirementuc "kaoru/internal/feature/requirement/application/usecase"
)

// Sources son los casos de uso con los que se busca cada tipo de entidad.
type Sources struct {
	Requirement *requirementuc.UseCases
	Change      *changeuc.UseCases
	Defect      *defectuc.UseCases
	Bundle      *bundleuc.UseCases
}

// Entities resuelve la entidad que se comenta.
type Entities struct {
	s Sources
}

var _ port.Entities = (*Entities)(nil)

// New crea el adaptador.
func New(s Sources) *Entities { return &Entities{s: s} }

// Resolve conserva el código de error del feature dueño de la entidad.
func (e *Entities) Resolve(ctx context.Context, entityType, code string) (domain.EntityRef, error) {
	var id string
	var err error
	switch entityType {
	case "requirement":
		out, e2 := e.s.Requirement.GetRequirement.Execute(ctx, requirementuc.GetRequirementInput{Code: code})
		id, err = out.Id, e2
	case "change":
		out, e2 := e.s.Change.GetChange.Execute(ctx, changeuc.GetChangeInput{Code: code})
		id, err = out.Change.Id, e2
	case "defect":
		out, e2 := e.s.Defect.GetDefect.Execute(ctx, defectuc.GetDefectInput{Code: code})
		id, err = out.Defect.Id, e2
	case "package":
		out, e2 := e.s.Bundle.FindPackage.Execute(ctx, bundleuc.FindPackageInput{Code: code})
		id, err = out.Id, e2
	default:
		return domain.EntityRef{}, axis.New(axis.NotFoundResource, "Ese tipo de entidad no admite comentarios.")
	}
	if err != nil {
		return domain.EntityRef{}, err
	}
	return domain.EntityRef{Type: entityType, Id: id, Code: code}, nil
}
