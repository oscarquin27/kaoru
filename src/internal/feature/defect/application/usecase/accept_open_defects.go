package usecase

import (
	"context"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/defect/application/port"
	shared "kaoru/internal/shared/domain"
)

// AcceptOpenDefects deja aceptados los DEF abiertos de un REQ: lo usa requirement al aprobarlo
// con defectos. Los DEF siguen abiertos como traza, pero ya no bloquean al REQ ni al PKG.
type AcceptOpenDefects struct {
	d Deps
}

// AcceptOpenDefectsInput es la entrada de AcceptOpenDefects. Reason es el motivo de la aprobación.
type AcceptOpenDefectsInput struct {
	Actor           shared.Actor
	RequirementCode string
	Reason          string
}

// Execute no exige permiso propio: quien llama ya comprobó el de la transición del REQ. Corre en
// la transacción de quien llama, si la hay. Devuelve los códigos de los DEF que aceptó.
func (uc *AcceptOpenDefects) Execute(ctx context.Context, in AcceptOpenDefectsInput) ([]string, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, axis.Invalid(axis.Required("reason", "Escribe el motivo."))
	}
	var accepted []string
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		open, _, err := uc.d.Repository.List(ctx, port.Filter{RequirementCode: in.RequirementCode, OnlyOpen: true, Page: 1, Size: maxOpenDefects})
		if err != nil {
			return err
		}
		pending := open[:0]
		for _, defect := range open {
			if defect.Snapshot().Acceptance == nil {
				pending = append(pending, defect)
			}
		}
		if len(pending) == 0 {
			return nil
		}
		actor, err := uc.d.Repository.FindUser(ctx, in.Actor.Id)
		if err != nil {
			return err
		}
		now := uc.d.Clock.Now()
		for _, listed := range pending {
			defect, err := lockDefect(ctx, uc.d.Repository, listed.Snapshot().Code.String())
			if err != nil {
				return err
			}
			if !defect.Accept(actor, reason, now) {
				continue
			}
			if err := uc.d.Repository.Update(ctx, defect); err != nil {
				return err
			}
			s := defect.Snapshot()
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, DefectId: s.Id, Code: s.Code, Action: port.AuditUpdate,
				Field: "accepted", NewValue: in.RequirementCode, Reason: reason,
			}); err != nil {
				return err
			}
			accepted = append(accepted, s.Code.String())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return accepted, nil
}
