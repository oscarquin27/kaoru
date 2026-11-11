package usecase

import (
	"context"

	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateExploration completa los cambios manuales o los hallazgos de un EXP.
type UpdateExploration struct {
	d Deps
}

// UpdateExplorationInput es la entrada de UpdateExploration; nil deja el campo como está.
type UpdateExplorationInput struct {
	Actor         shared.Actor
	Code          string
	ManualChanges *string
	Findings      *string
}

// Execute exige el permiso exp.create.
func (uc *UpdateExploration) Execute(ctx context.Context, in UpdateExplorationInput) (domain.ExplorationSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionCreateExploration); err != nil {
		return domain.ExplorationSnapshot{}, err
	}
	var out domain.ExplorationSnapshot
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		exploration, err := findExploration(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		changes := exploration.Apply(in.ManualChanges, in.Findings, uc.d.Clock.Now())
		out = exploration.Snapshot()
		if len(changes) == 0 {
			return nil
		}
		if err := uc.d.Repository.UpdateExploration(ctx, exploration); err != nil {
			return err
		}
		for _, c := range changes {
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, EntityType: port.EntityExploration, EntityId: out.Id, Code: out.Code,
				Action: port.AuditUpdate, Field: c.Field, OldValue: c.Old, NewValue: c.New,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.ExplorationSnapshot{}, err
	}
	return out, nil
}
