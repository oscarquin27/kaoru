package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateFunctionalCase define un UC para un REQ.
type CreateFunctionalCase struct {
	d Deps
}

// CreateFunctionalCaseInput es la entrada de CreateFunctionalCase.
type CreateFunctionalCaseInput struct {
	Actor           shared.Actor
	RequirementCode string
	Title           string
	Description     string
}

// Execute exige el permiso testdesign.manage.
func (uc *CreateFunctionalCase) Execute(ctx context.Context, in CreateFunctionalCaseInput) (domain.FunctionalCaseSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return domain.FunctionalCaseSnapshot{}, err
	}
	requirement, err := uc.d.Requirements.Find(ctx, in.RequirementCode)
	if err != nil {
		return domain.FunctionalCaseSnapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.FunctionalCaseSnapshot{}, err
	}
	now := uc.d.Clock.Now()
	var out domain.FunctionalCaseSnapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixFunctionalCase, now.Year())
		if err != nil {
			return err
		}
		functionalCase, err := domain.NewFunctionalCase(id, code, requirement, in.Title, in.Description, in.Actor.Id, now)
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertFunctionalCase(ctx, functionalCase); err != nil {
			return err
		}
		out = functionalCase.Snapshot()
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityFunctionalCase, EntityId: id, Code: code,
			Action: port.AuditCreate, NewValue: out.Title,
		})
	})
	if err != nil {
		return domain.FunctionalCaseSnapshot{}, err
	}
	return out, nil
}
