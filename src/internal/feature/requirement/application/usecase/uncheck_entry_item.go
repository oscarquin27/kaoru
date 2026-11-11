package usecase

import (
	"context"

	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// UncheckEntryItem quita la marca de un ítem del contrato de entrada.
type UncheckEntryItem struct {
	d Deps
}

// UncheckEntryItemInput es la entrada de UncheckEntryItem.
type UncheckEntryItemInput struct {
	Actor   shared.Actor
	Code    string
	ItemKey string
}

// Execute exige el permiso entry_contract.check. Quitar una marca que no estaba no cambia nada.
func (uc *UncheckEntryItem) Execute(ctx context.Context, in UncheckEntryItemInput) error {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionEntryContract); err != nil {
		return err
	}
	return uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		requirement, item, err := findEntryItem(ctx, uc.d.Repository, in.Code, in.ItemKey)
		if err != nil || !item.Done {
			return err
		}
		s := requirement.Snapshot()
		if err := uc.d.Repository.UncheckEntryItem(ctx, s.Id, item.Key); err != nil {
			return err
		}
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, RequirementId: s.Id, Code: s.Code, Action: port.AuditUpdate,
			Field: "entryContract." + item.Key, OldValue: doneText(true), NewValue: doneText(false),
		})
	})
}

// findEntryItem lee el REQ y un ítem de su contrato de entrada.
func findEntryItem(ctx context.Context, repo port.Repository, code, itemKey string) (domain.Requirement, domain.EntryItem, error) {
	requirement, err := findRequirement(ctx, repo, code)
	if err != nil {
		return domain.Requirement{}, domain.EntryItem{}, err
	}
	items, err := repo.ListEntryContract(ctx, requirement.Snapshot().Id)
	if err != nil {
		return domain.Requirement{}, domain.EntryItem{}, err
	}
	for _, item := range items {
		if item.Key == itemKey {
			return requirement, item, nil
		}
	}
	return domain.Requirement{}, domain.EntryItem{}, ErrEntryItemNotFound
}

func doneText(done bool) string {
	if done {
		return "cumplido"
	}
	return "pendiente"
}
