// Package usecase contiene los casos de uso de defect.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/defect/application/port"
	"kaoru/internal/feature/defect/domain"
	shared "kaoru/internal/shared/domain"
)

// Permisos del feature.
const (
	PermissionCreate = "def.create"
	PermissionEdit   = "def.edit"
)

// ErrDefectNotFound es el error de un DEF que no existe.
var ErrDefectNotFound = axis.New(axis.NotFoundResource, "El DEF no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Workflow    port.Workflow
	Timeline    port.Timeline
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateDefect     *CreateDefect
	GetDefect        *GetDefect
	ListDefects      *ListDefects
	UpdateDefect     *UpdateDefect
	TransitionDefect *TransitionDefect
	ListOpenDefects  *ListOpenDefects
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Workflow == nil || d.Timeline == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("defect: faltan dependencias")
	}
	return &UseCases{
		CreateDefect:     &CreateDefect{d: d},
		GetDefect:        &GetDefect{d: d},
		ListDefects:      &ListDefects{d: d},
		UpdateDefect:     &UpdateDefect{d: d},
		TransitionDefect: &TransitionDefect{d: d},
		ListOpenDefects:  &ListOpenDefects{d: d},
	}, nil
}

// DefectOutput es un DEF con el tiempo que lleva en su estado actual, calculado al leer.
type DefectOutput struct {
	Defect      domain.Snapshot
	DaysInState float64
}

func findDefect(ctx context.Context, repo port.Repository, raw string) (domain.Defect, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Defect{}, ErrDefectNotFound
	}
	defect, err := repo.FindByCode(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Defect{}, ErrDefectNotFound
	}
	return defect, err
}

// toOutputs agrega a cada DEF su tiempo en estado con una sola consulta a la auditoría.
func toOutputs(ctx context.Context, d Deps, defects []domain.Defect) ([]DefectOutput, error) {
	ids := make([]string, len(defects))
	for i, defect := range defects {
		ids[i] = defect.Snapshot().Id
	}
	entered, err := d.Timeline.LastTransitions(ctx, ids)
	if err != nil {
		return nil, err
	}
	now := d.Clock.Now()
	out := make([]DefectOutput, len(defects))
	for i, defect := range defects {
		s := defect.Snapshot()
		out[i] = DefectOutput{Defect: s}
		if at, ok := entered[s.Id]; ok {
			out[i].DaysInState = defect.DaysInState(&at, now)
		} else {
			out[i].DaysInState = defect.DaysInState(nil, now)
		}
	}
	return out, nil
}

func toOutput(ctx context.Context, d Deps, defect domain.Defect) (DefectOutput, error) {
	outs, err := toOutputs(ctx, d, []domain.Defect{defect})
	if err != nil {
		return DefectOutput{}, err
	}
	return outs[0], nil
}

// findAssignee busca a quien se asigna el DEF; vacío significa «sin asignar».
func findAssignee(ctx context.Context, repo port.Repository, id string) (*domain.UserRef, error) {
	if id == "" {
		return nil, nil
	}
	assignee, err := repo.FindUser(ctx, shared.UserId(id))
	if axis.IsCode(err, axis.NotFoundResource) {
		return nil, axis.Invalid(axis.InvalidField("assignee", "El usuario no existe."))
	}
	if err != nil {
		return nil, err
	}
	return &assignee, nil
}
