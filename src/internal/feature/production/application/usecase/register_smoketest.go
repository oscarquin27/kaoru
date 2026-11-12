package usecase

import (
	"context"

	"kaoru/internal/feature/production/application/port"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// RegisterSmokeTest registra el resultado de la prueba de humo posterior a un despliegue de un PKG
// en producción. Después no se edita ni se borra: si se repite, se registra otra.
type RegisterSmokeTest struct {
	d Deps
}

// RegisterSmokeTestInput es la entrada de RegisterSmokeTest.
type RegisterSmokeTestInput struct {
	Actor          shared.Actor
	DeploymentCode string
	Result         string
	Notes          string
}

// Execute exige el permiso smoke.create.
func (uc *RegisterSmokeTest) Execute(ctx context.Context, in RegisterSmokeTestInput) (domain.SmokeTestSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionSmoke); err != nil {
		return domain.SmokeTestSnapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.SmokeTestSnapshot{}, err
	}
	now := uc.d.Clock.Now()

	var out domain.SmokeTestSnapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		deployment, err := findDeployment(ctx, uc.d.Repository, in.DeploymentCode)
		if err != nil {
			return err
		}
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixSmokeTest, now.Year())
		if err != nil {
			return err
		}
		smokeTest, err := domain.NewSmokeTest(domain.NewSmokeTestParams{
			Id: id, Code: code, Deployment: deployment, Result: in.Result, Notes: in.Notes, ExecutedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertSmokeTest(ctx, smokeTest); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntitySmokeTest, EntityId: id, Code: code.String(),
			Action: port.AuditCreate, Field: "result", NewValue: string(smokeTest.Snapshot().Result),
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindSmokeTest(ctx, code)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.SmokeTestSnapshot{}, err
	}
	return out, nil
}
