package usecase

import (
	"context"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateRun registra un RUN: la ejecución oficial de un TC sobre un BUILD y un ENV del catálogo.
// Quién lo registra y cuándo salen de la sesión y del servidor.
type CreateRun struct {
	d Deps
}

// CreateRunInput es la entrada de CreateRun.
type CreateRunInput struct {
	Actor           shared.Actor
	TestCaseCode    string
	BuildCode       string
	EnvironmentCode string
	Result          string
	Notes           string
	ExecutedAt      time.Time
}

// Execute exige el permiso run.create.
func (uc *CreateRun) Execute(ctx context.Context, in CreateRunInput) (domain.RunSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionCreateRun); err != nil {
		return domain.RunSnapshot{}, err
	}
	var missing axis.FieldErrors
	testCase, err := uc.d.Repository.FindTestCase(ctx, in.TestCaseCode)
	switch {
	case in.TestCaseCode == "":
		missing.Add(axis.Required("testCase", "Indica el TC que se ejecutó."))
	case axis.IsCode(err, axis.NotFoundResource):
		missing.Add(axis.InvalidField("testCase", "El TC no existe."))
	case err != nil:
		return domain.RunSnapshot{}, err
	}
	build, environment, err := findCatalog(ctx, uc.d, in.BuildCode, in.EnvironmentCode, &missing)
	if err != nil {
		return domain.RunSnapshot{}, err
	}
	if err := missing.Err(); err != nil {
		return domain.RunSnapshot{}, err
	}
	executor, err := uc.d.Repository.FindUser(ctx, in.Actor.Id)
	if err != nil {
		return domain.RunSnapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.RunSnapshot{}, err
	}
	now := uc.d.Clock.Now()
	var out domain.RunSnapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixRun, now.Year())
		if err != nil {
			return err
		}
		run, err := domain.NewRun(domain.NewRunParams{
			Id: id, Code: code, TestCase: testCase, Build: build, Environment: environment,
			Result: in.Result, Notes: in.Notes,
			ExecutedBy: executor, ExecutedAt: in.ExecutedAt, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertRun(ctx, run); err != nil {
			return err
		}
		out = run.Snapshot()
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityRun, EntityId: id, Code: code,
			Action: port.AuditCreate, Field: "result", NewValue: string(out.Result),
		})
	})
	if err != nil {
		return domain.RunSnapshot{}, err
	}
	return out, nil
}
