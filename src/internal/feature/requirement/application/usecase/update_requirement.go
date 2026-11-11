package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateRequirement edita los datos de un REQ y audita cada campo que cambió.
type UpdateRequirement struct {
	d Deps
}

// UpdateRequirementInput es la entrada de UpdateRequirement. Un puntero nil deja el campo
// como está; ChangeSet y OwnerSet dicen si se toca el CHG o el responsable (vacío los quita).
type UpdateRequirementInput struct {
	Actor              shared.Actor
	Code               string
	Title              *string
	Description        *string
	AcceptanceCriteria *string
	ChangeSet          bool
	ChangeCode         string
	OwnerSet           bool
	OwnerId            string
}

// Execute exige el permiso req.edit.
func (uc *UpdateRequirement) Execute(ctx context.Context, in UpdateRequirementInput) (domain.Snapshot, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionEdit); err != nil {
		return domain.Snapshot{}, err
	}
	edit := domain.Edit{Title: in.Title, Description: in.Description, AcceptanceCriteria: in.AcceptanceCriteria}
	var errs axis.FieldErrors
	var err error
	if in.ChangeSet {
		edit.Change.Set = true
		if edit.Change.Value, err = findChangeRef(ctx, uc.d.Repository, in.ChangeCode, &errs); err != nil {
			return domain.Snapshot{}, err
		}
	}
	if in.OwnerSet {
		edit.Owner.Set = true
		if edit.Owner.Value, err = findOwner(ctx, uc.d.Repository, in.OwnerId, &errs); err != nil {
			return domain.Snapshot{}, err
		}
	}
	if err := errs.Err(); err != nil {
		return domain.Snapshot{}, err
	}

	var out domain.Snapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		requirement, err := lockRequirement(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		changes, err := requirement.Apply(edit, uc.d.Clock.Now())
		out = requirement.Snapshot()
		if err != nil || len(changes) == 0 {
			return err
		}
		if err := uc.d.Repository.Update(ctx, requirement); err != nil {
			return err
		}
		return recordChanges(ctx, uc.d, in.Actor.Id, out, changes)
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}
