package usecase

import (
	"context"

	"kaoru/internal/feature/production/application/port"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// SubmitDiagnostic entrega el diagnóstico posterior a un despliegue de un PKG en producción.
// Su salida se adjunta después como documento del DIAG; QA lo revisa aparte.
type SubmitDiagnostic struct {
	d Deps
}

// SubmitDiagnosticInput es la entrada de SubmitDiagnostic.
type SubmitDiagnosticInput struct {
	Actor          shared.Actor
	DeploymentCode string
	Notes          string
}

// Execute exige el permiso diag.submit.
func (uc *SubmitDiagnostic) Execute(ctx context.Context, in SubmitDiagnosticInput) (domain.DiagnosticSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionSubmitDiagnostic); err != nil {
		return domain.DiagnosticSnapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.DiagnosticSnapshot{}, err
	}
	now := uc.d.Clock.Now()

	var out domain.DiagnosticSnapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		deployment, err := findDeployment(ctx, uc.d.Repository, in.DeploymentCode)
		if err != nil {
			return err
		}
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixDiagnostic, now.Year())
		if err != nil {
			return err
		}
		diagnostic, err := domain.NewDiagnostic(domain.NewDiagnosticParams{
			Id: id, Code: code, Deployment: deployment, Notes: in.Notes, SubmittedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertDiagnostic(ctx, diagnostic); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityDiagnostic, EntityId: id, Code: code.String(),
			Action: port.AuditCreate, Field: "deployment", NewValue: deployment.Code,
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindDiagnostic(ctx, code)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.DiagnosticSnapshot{}, err
	}
	return out, nil
}
