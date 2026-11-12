package usecase

import (
	"context"
	"slices"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// SetRequirementComponents declara los componentes que toca un REQ: reemplaza la lista.
type SetRequirementComponents struct {
	d Deps
}

// SetRequirementComponentsInput es la entrada de SetRequirementComponents.
// Components son códigos de COMP; una lista vacía quita todos y nil no vale.
type SetRequirementComponentsInput struct {
	Actor      shared.Actor
	Code       string
	Components []string
}

// Execute exige el permiso req.edit. Los componentes deben ser del sistema del REQ y no estar retirados.
func (uc *SetRequirementComponents) Execute(ctx context.Context, in SetRequirementComponentsInput) ([]domain.ComponentRef, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionEdit); err != nil {
		return nil, err
	}
	if in.Components == nil {
		return nil, axis.Invalid(axis.Required("components", "Indica los componentes."))
	}
	var out []domain.ComponentRef
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		requirement, err := lockRequirement(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		s := requirement.Snapshot()
		refs, err := uc.d.Components.Resolve(ctx, s.System.Key, in.Components)
		if err != nil {
			return err
		}
		current, err := uc.d.Repository.ListComponents(ctx, s.Id)
		if err != nil {
			return err
		}
		before, after := componentCodes(current), componentCodes(refs)
		if before == after {
			out = current
			return nil
		}
		ids := make([]string, len(refs))
		for i, r := range refs {
			ids[i] = r.Id
		}
		if err := uc.d.Repository.ReplaceComponents(ctx, s.Id, ids); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, RequirementId: s.Id, Code: s.Code, Action: port.AuditUpdate,
			Field: "components", OldValue: before, NewValue: after,
		}); err != nil {
			return err
		}
		out, err = uc.d.Repository.ListComponents(ctx, s.Id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// componentCodes es la lista de códigos, ordenada, tal como queda en la auditoría.
func componentCodes(refs []domain.ComponentRef) string {
	codes := make([]string, len(refs))
	for i, r := range refs {
		codes[i] = r.Code
	}
	slices.Sort(codes)
	return strings.Join(codes, ", ")
}
