package usecase

import (
	"context"

	"kaoru/internal/feature/production/application/port"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// ReviewDiagnostic es la revisión de QA de un DIAG: marca su resultado, una sola vez (regla de negocio 7).
type ReviewDiagnostic struct {
	d Deps
}

// ReviewDiagnosticInput es la entrada de ReviewDiagnostic.
type ReviewDiagnosticInput struct {
	Actor  shared.Actor
	Code   string
	Result string
	Notes  string
}

// Execute exige el permiso diag.review. Devuelve INVALID_ACTION si el DIAG ya estaba revisado.
func (uc *ReviewDiagnostic) Execute(ctx context.Context, in ReviewDiagnosticInput) (domain.DiagnosticSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionReviewDiagnostic); err != nil {
		return domain.DiagnosticSnapshot{}, err
	}
	var out domain.DiagnosticSnapshot
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		diagnostic, err := findDiagnostic(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		// Dos revisiones simultáneas van una tras otra: se bloquea el PKG y se vuelve a leer.
		if err := uc.d.Repository.LockPackage(ctx, diagnostic.Snapshot().Package.Id); err != nil {
			return err
		}
		if diagnostic, err = findDiagnostic(ctx, uc.d.Repository, in.Code); err != nil {
			return err
		}
		review, err := diagnostic.Review(in.Result, in.Notes, in.Actor.Id, uc.d.Clock.Now())
		if err != nil {
			return err
		}
		s := diagnostic.Snapshot()
		if err := uc.d.Repository.InsertReview(ctx, s.Id, review); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityDiagnostic, EntityId: s.Id, Code: s.Code.String(),
			Action: port.AuditDecision, Field: "review", NewValue: string(review.Result), Reason: review.Notes,
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindDiagnostic(ctx, s.Code)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.DiagnosticSnapshot{}, err
	}
	return out, nil
}
