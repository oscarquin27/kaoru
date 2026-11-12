// Package usecase contiene los casos de uso de component.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/component/application/port"
	"kaoru/internal/feature/component/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionManage es el permiso para mantener el catálogo de componentes.
const PermissionManage = "comp.manage"

// ErrComponentNotFound es el error de un COMP que no existe.
var ErrComponentNotFound = axis.New(axis.NotFoundResource, "El COMP no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateComponent   *CreateComponent
	GetComponent      *GetComponent
	ListComponents    *ListComponents
	UpdateComponent   *UpdateComponent
	ResolveComponents *ResolveComponents
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("component: faltan dependencias")
	}
	return &UseCases{
		CreateComponent:   &CreateComponent{d: d},
		GetComponent:      &GetComponent{d: d},
		ListComponents:    &ListComponents{d: d},
		UpdateComponent:   &UpdateComponent{d: d},
		ResolveComponents: &ResolveComponents{d: d},
	}, nil
}

// ComponentOutput es un COMP tal como lo devuelven los casos de uso.
type ComponentOutput struct {
	Component     domain.Snapshot
	CreatedByName string
}

func toOutput(v port.ComponentView) ComponentOutput {
	return ComponentOutput{Component: v.Component.Snapshot(), CreatedByName: v.CreatedByName}
}

// lockComponent bloquea la entidad y la lee: se usa dentro de la transacción de todo caso de uso que
// escribe, para que dos cambios simultáneos sobre la misma entidad no se pisen.
func lockComponent(ctx context.Context, repo port.Repository, raw string) (port.ComponentView, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return port.ComponentView{}, ErrComponentNotFound
	}
	if err := repo.Lock(ctx, code); err != nil {
		if axis.IsCode(err, axis.NotFoundResource) {
			return port.ComponentView{}, ErrComponentNotFound
		}
		return port.ComponentView{}, err
	}
	return findComponent(ctx, repo, raw)
}

// findComponent valida el código y lee el COMP; un código mal formado tampoco existe.
func findComponent(ctx context.Context, repo port.Repository, raw string) (port.ComponentView, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return port.ComponentView{}, ErrComponentNotFound
	}
	view, err := repo.FindByCode(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return port.ComponentView{}, ErrComponentNotFound
	}
	return view, err
}
