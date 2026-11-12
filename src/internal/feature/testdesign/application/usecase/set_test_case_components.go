package usecase

import (
	"context"
	"slices"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// SetTestCaseComponents declara los componentes que ejercita un TC: reemplaza la lista.
type SetTestCaseComponents struct {
	d Deps
}

// SetTestCaseComponentsInput es la entrada de SetTestCaseComponents.
// Components son códigos de COMP; una lista vacía quita todos y nil no vale.
type SetTestCaseComponentsInput struct {
	Actor      shared.Actor
	Code       string
	Components []string
}

// Execute exige el permiso testdesign.manage. Los componentes deben ser del sistema del TC y no estar retirados.
func (uc *SetTestCaseComponents) Execute(ctx context.Context, in SetTestCaseComponentsInput) ([]domain.ComponentRef, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return nil, err
	}
	if in.Components == nil {
		return nil, axis.Invalid(axis.Required("components", "Indica los componentes."))
	}
	code, err := shared.ParseCode(in.Code)
	if err != nil {
		return nil, ErrTestCaseNotFound
	}
	var out []domain.ComponentRef
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		// Se bloquea el TC para que dos declaraciones simultáneas no se mezclen.
		if err := uc.d.Repository.LockTestCase(ctx, code); err != nil {
			if axis.IsCode(err, axis.NotFoundResource) {
				return ErrTestCaseNotFound
			}
			return err
		}
		testCase, err := findTestCase(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		s := testCase.Snapshot()
		refs, err := uc.d.Components.Resolve(ctx, s.SystemKey, in.Components)
		if err != nil {
			return err
		}
		current, err := uc.d.Repository.ListTestCaseComponents(ctx, s.Id)
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
		if err := uc.d.Repository.ReplaceTestCaseComponents(ctx, s.Id, ids); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityTestCase, EntityId: s.Id, Code: s.Code, Action: port.AuditUpdate,
			Field: "components", OldValue: before, NewValue: after,
		}); err != nil {
			return err
		}
		out, err = uc.d.Repository.ListTestCaseComponents(ctx, s.Id)
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
