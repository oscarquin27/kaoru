package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateFunctionalCase edita un UC y audita cada campo que cambió.
type UpdateFunctionalCase struct {
	d Deps
}

// UpdateFunctionalCaseInput es la entrada de UpdateFunctionalCase; nil deja el campo como está.
type UpdateFunctionalCaseInput struct {
	Actor       shared.Actor
	Code        string
	Title       *string
	Description *string
}

// Execute exige el permiso testdesign.manage.
func (uc *UpdateFunctionalCase) Execute(ctx context.Context, in UpdateFunctionalCaseInput) (domain.FunctionalCaseSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return domain.FunctionalCaseSnapshot{}, err
	}
	var out domain.FunctionalCaseSnapshot
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		functionalCase, err := findFunctionalCase(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		changes, err := functionalCase.Apply(in.Title, in.Description, uc.d.Clock.Now())
		out = functionalCase.Snapshot()
		if err != nil || len(changes) == 0 {
			return err
		}
		if err := uc.d.Repository.UpdateFunctionalCase(ctx, functionalCase); err != nil {
			return err
		}
		return recordChanges(ctx, uc.d, in.Actor.Id, port.EntityFunctionalCase, out.Id, out.Code, changes)
	})
	if err != nil {
		return domain.FunctionalCaseSnapshot{}, err
	}
	return out, nil
}
