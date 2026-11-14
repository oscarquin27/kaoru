// Package usecase contiene los casos de uso de requirement.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// Permisos del feature.
const (
	PermissionCreate        = "req.create"
	PermissionEdit          = "req.edit"
	PermissionManageLinks   = "link.manage"
	PermissionEntryContract = "entry_contract.check"
)

// ErrRequirementNotFound es el error de un REQ que no existe.
var ErrRequirementNotFound = axis.New(axis.NotFoundResource, "El REQ no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Workflow    port.Workflow
	Facts       port.Facts
	Components  port.Components
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateRequirement         *CreateRequirement
	GetRequirement            *GetRequirement
	ListRequirements          *ListRequirements
	UpdateRequirement         *UpdateRequirement
	TransitionRequirement     *TransitionRequirement
	GetReadiness              *GetReadiness
	ListLinks                 *ListLinks
	AddLink                   *AddLink
	UpdateLink                *UpdateLink
	RemoveLink                *RemoveLink
	GetEntryContract          *GetEntryContract
	CheckEntryItem            *CheckEntryItem
	UncheckEntryItem          *UncheckEntryItem
	ListRequirementComponents *ListRequirementComponents
	SetRequirementComponents  *SetRequirementComponents
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Workflow == nil || d.Facts == nil || d.Components == nil ||
		d.AuditTrail == nil || d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("requirement: faltan dependencias")
	}
	return &UseCases{
		CreateRequirement:         &CreateRequirement{d: d},
		GetRequirement:            &GetRequirement{d: d},
		ListRequirements:          &ListRequirements{d: d},
		UpdateRequirement:         &UpdateRequirement{d: d},
		TransitionRequirement:     &TransitionRequirement{d: d},
		GetReadiness:              &GetReadiness{d: d},
		ListLinks:                 &ListLinks{d: d},
		AddLink:                   &AddLink{d: d},
		UpdateLink:                &UpdateLink{d: d},
		RemoveLink:                &RemoveLink{d: d},
		GetEntryContract:          &GetEntryContract{d: d},
		CheckEntryItem:            &CheckEntryItem{d: d},
		UncheckEntryItem:          &UncheckEntryItem{d: d},
		ListRequirementComponents: &ListRequirementComponents{d: d},
		SetRequirementComponents:  &SetRequirementComponents{d: d},
	}, nil
}

// lockRequirement bloquea la entidad y la lee: se usa dentro de la transacción de todo caso de uso que
// escribe, para que dos cambios simultáneos sobre la misma entidad no se pisen.
func lockRequirement(ctx context.Context, repo port.Repository, raw string) (domain.Requirement, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Requirement{}, ErrRequirementNotFound
	}
	if err := repo.Lock(ctx, code); err != nil {
		if axis.IsCode(err, axis.NotFoundResource) {
			return domain.Requirement{}, ErrRequirementNotFound
		}
		return domain.Requirement{}, err
	}
	return findRequirement(ctx, repo, raw)
}

// findRequirement valida el código y lee el REQ; un código mal formado tampoco existe.
func findRequirement(ctx context.Context, repo port.Repository, raw string) (domain.Requirement, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Requirement{}, ErrRequirementNotFound
	}
	requirement, err := repo.FindByCode(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Requirement{}, ErrRequirementNotFound
	}
	return requirement, err
}

// gatherFacts junta los hechos con los que se evalúan las reglas de bloqueo del REQ.
func gatherFacts(ctx context.Context, d Deps, requirement domain.Requirement) (domain.Facts, error) {
	s := requirement.Snapshot()
	facts := domain.Facts{Route: requirement.Route()}
	var err error
	if facts.EntryItems, err = d.Repository.ListEntryContract(ctx, s.Id); err != nil {
		return domain.Facts{}, err
	}
	links, err := d.Repository.ListLinks(ctx, s.Id)
	if err != nil {
		return domain.Facts{}, err
	}
	for _, link := range links {
		if link.NeedsSnapshot() {
			facts.LinksWithoutSnapshot++
		}
	}
	if facts.Risk, err = d.Facts.Risk(ctx, s.Code); err != nil {
		return domain.Facts{}, err
	}
	if facts.Coverage, err = d.Facts.Coverage(ctx, s.Code); err != nil {
		return domain.Facts{}, err
	}
	if facts.OpenDefects, err = d.Facts.OpenDefects(ctx, s.Code); err != nil {
		return domain.Facts{}, err
	}
	if s.Change != nil {
		certification, err := d.Facts.Certification(ctx, s.Change.Code)
		if err != nil {
			return domain.Facts{}, err
		}
		facts.Certification = &certification
	}
	return facts, nil
}

// recordChanges audita cada campo que cambió.
func recordChanges(ctx context.Context, d Deps, actor shared.UserId, s domain.Snapshot, changes []domain.FieldChange) error {
	for _, c := range changes {
		if err := d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: actor, RequirementId: s.Id, Code: s.Code, Action: port.AuditUpdate,
			Field: c.Field, OldValue: c.Old, NewValue: c.New,
		}); err != nil {
			return err
		}
	}
	return nil
}
