package usecase

import (
	"context"
	"fmt"

	"kaoru/internal/feature/production/application/port"
	shared "kaoru/internal/shared/domain"
)

// SignChecklistStep firma un paso de un checklist. La firma es el usuario de la sesión, la fecha
// del servidor y su comentario; después no se cambia ni se quita (reglas §A.13).
type SignChecklistStep struct {
	d Deps
}

// SignChecklistStepInput es la entrada de SignChecklistStep.
type SignChecklistStepInput struct {
	Actor    shared.Actor
	Code     string
	Position int
	Comment  string
}

// Execute exige el permiso chk.sign. Devuelve NOT_FOUND_RESOURCE si el CHK o el paso no existen
// e INVALID_ACTION si el paso ya estaba firmado.
func (uc *SignChecklistStep) Execute(ctx context.Context, in SignChecklistStepInput) (ChecklistOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionSign); err != nil {
		return ChecklistOutput{}, err
	}
	var out ChecklistOutput
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		checklist, err := findChecklist(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		// Dos firmas simultáneas del mismo paso van una tras otra: se bloquea el PKG y se vuelve a leer.
		if err := uc.d.Repository.LockPackage(ctx, checklist.Snapshot().Package.Id); err != nil {
			return err
		}
		if checklist, err = findChecklist(ctx, uc.d.Repository, in.Code); err != nil {
			return err
		}
		signature, err := checklist.Sign(in.Position, in.Actor.Id, in.Comment, uc.d.Clock.Now())
		if err != nil {
			return err
		}
		s := checklist.Snapshot()
		if err := uc.d.Repository.InsertSignature(ctx, s.Id, signature); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityChecklist, EntityId: s.Id, Code: s.Code.String(),
			Action: port.AuditDecision, Field: fmt.Sprintf("step.%d", in.Position), NewValue: "signed", Reason: signature.Comment,
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindChecklist(ctx, s.Code)
		out = toChecklistOutput(saved)
		return err
	})
	if err != nil {
		return ChecklistOutput{}, err
	}
	return out, nil
}
