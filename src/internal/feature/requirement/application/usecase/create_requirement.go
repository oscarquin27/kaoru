package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateRequirement registra un REQ en el estado inicial de su flujo.
type CreateRequirement struct {
	d Deps
}

// CreateRequirementInput es la entrada de CreateRequirement. ChangeCode y OwnerId son opcionales.
type CreateRequirementInput struct {
	Actor              shared.Actor
	Title              string
	Description        string
	AcceptanceCriteria string
	SystemKey          string
	ChangeCode         string
	OwnerId            string
}

// Execute exige el permiso req.create.
func (uc *CreateRequirement) Execute(ctx context.Context, in CreateRequirementInput) (domain.Snapshot, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionCreate); err != nil {
		return domain.Snapshot{}, err
	}
	var errs axis.FieldErrors
	system, err := uc.d.Repository.FindSystem(ctx, in.SystemKey)
	switch {
	case err == nil:
	case !axis.IsCode(err, axis.NotFoundResource):
		return domain.Snapshot{}, err
	case in.SystemKey == "":
		errs.Add(axis.Required("system", "Elige un sistema."))
	default:
		errs.Add(axis.InvalidField("system", "El sistema no existe."))
	}
	change, err := findChangeRef(ctx, uc.d.Repository, in.ChangeCode, &errs)
	if err != nil {
		return domain.Snapshot{}, err
	}
	owner, err := findOwner(ctx, uc.d.Repository, in.OwnerId, &errs)
	if err != nil {
		return domain.Snapshot{}, err
	}
	if err := errs.Err(); err != nil {
		return domain.Snapshot{}, err
	}
	creator, err := uc.d.Repository.FindUser(ctx, in.Actor.Id)
	if err != nil {
		return domain.Snapshot{}, err
	}
	state, err := uc.d.Workflow.InitialState(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.Snapshot{}, err
	}
	now := uc.d.Clock.Now()

	var out domain.Snapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, now.Year())
		if err != nil {
			return err
		}
		requirement, err := domain.New(domain.NewParams{
			Id: id, Code: code, Title: in.Title, Description: in.Description, AcceptanceCriteria: in.AcceptanceCriteria,
			System: system, Change: change, State: state, Owner: owner, CreatedBy: creator, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.Insert(ctx, requirement); err != nil {
			return err
		}
		out = requirement.Snapshot()
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, RequirementId: id, Code: code, Action: port.AuditCreate, NewValue: out.Title,
		})
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}

// findChangeRef busca el CHG por su código; vacío significa «sin CHG».
func findChangeRef(ctx context.Context, repo port.Repository, code string, errs *axis.FieldErrors) (*domain.ChangeRef, error) {
	if code == "" {
		return nil, nil
	}
	change, err := repo.FindChange(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		errs.Add(axis.InvalidField("change", "El CHG no existe."))
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &change, nil
}

// findOwner busca al responsable; vacío significa «sin responsable».
func findOwner(ctx context.Context, repo port.Repository, id string, errs *axis.FieldErrors) (*domain.UserRef, error) {
	if id == "" {
		return nil, nil
	}
	owner, err := repo.FindUser(ctx, shared.UserId(id))
	if axis.IsCode(err, axis.NotFoundResource) {
		errs.Add(axis.InvalidField("owner", "El responsable no existe."))
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &owner, nil
}
