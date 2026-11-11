package usecase

import (
	"context"
	"strings"

	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// CheckEntryItem marca un ítem del contrato de entrada como cumplido.
type CheckEntryItem struct {
	d Deps
}

// CheckEntryItemInput es la entrada de CheckEntryItem. Quién y cuándo salen de la sesión y del servidor.
type CheckEntryItemInput struct {
	Actor   shared.Actor
	Code    string
	ItemKey string
	Note    string
}

// Execute exige el permiso entry_contract.check.
func (uc *CheckEntryItem) Execute(ctx context.Context, in CheckEntryItemInput) (domain.EntryItem, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionEntryContract); err != nil {
		return domain.EntryItem{}, err
	}
	note := strings.TrimSpace(in.Note)
	var out domain.EntryItem
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		requirement, item, err := findEntryItem(ctx, uc.d.Repository, in.Code, in.ItemKey)
		if err != nil {
			return err
		}
		s := requirement.Snapshot()
		if err := uc.d.Repository.CheckEntryItem(ctx, s.Id, item.Key, note, in.Actor.Id, uc.d.Clock.Now()); err != nil {
			return err
		}
		if !item.Done || item.Note != note {
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, RequirementId: s.Id, Code: s.Code, Action: port.AuditUpdate,
				Field: "entryContract." + item.Key, OldValue: doneText(item.Done), NewValue: doneText(true), Reason: note,
			}); err != nil {
				return err
			}
		}
		_, out, err = findEntryItem(ctx, uc.d.Repository, in.Code, in.ItemKey)
		return err
	})
	if err != nil {
		return domain.EntryItem{}, err
	}
	return out, nil
}
