package usecase

import (
	"context"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateExploration registra un EXP: una ejecución exploratoria de un REQ sobre un BUILD y un ENV del catálogo.
type CreateExploration struct {
	d Deps
}

// CreateExplorationInput es la entrada de CreateExploration.
type CreateExplorationInput struct {
	Actor           shared.Actor
	RequirementCode string
	BuildCode       string
	EnvironmentCode string
	ManualChanges   string
	Findings        string
	ExecutedAt      time.Time
}

// Execute exige el permiso exp.create.
func (uc *CreateExploration) Execute(ctx context.Context, in CreateExplorationInput) (domain.ExplorationSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionCreateExploration); err != nil {
		return domain.ExplorationSnapshot{}, err
	}
	var missing axis.FieldErrors
	requirement, err := uc.d.Repository.FindRequirement(ctx, in.RequirementCode)
	switch {
	case in.RequirementCode == "":
		missing.Add(axis.Required("requirement", "Indica el REQ que se exploró."))
	case axis.IsCode(err, axis.NotFoundResource):
		missing.Add(axis.InvalidField("requirement", "El REQ no existe."))
	case err != nil:
		return domain.ExplorationSnapshot{}, err
	}
	build, environment, err := findCatalog(ctx, uc.d, in.BuildCode, in.EnvironmentCode, &missing)
	if err != nil {
		return domain.ExplorationSnapshot{}, err
	}
	if err := missing.Err(); err != nil {
		return domain.ExplorationSnapshot{}, err
	}
	executor, err := uc.d.Repository.FindUser(ctx, in.Actor.Id)
	if err != nil {
		return domain.ExplorationSnapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.ExplorationSnapshot{}, err
	}
	now := uc.d.Clock.Now()
	var out domain.ExplorationSnapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixExploration, now.Year())
		if err != nil {
			return err
		}
		exploration, err := domain.NewExploration(domain.NewExplorationParams{
			Id: id, Code: code, Requirement: requirement, Build: build, Environment: environment,
			ManualChanges: in.ManualChanges, Findings: in.Findings,
			ExecutedBy: executor, ExecutedAt: in.ExecutedAt, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertExploration(ctx, exploration); err != nil {
			return err
		}
		out = exploration.Snapshot()
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityExploration, EntityId: id, Code: code, Action: port.AuditCreate,
		})
	})
	if err != nil {
		return domain.ExplorationSnapshot{}, err
	}
	return out, nil
}
