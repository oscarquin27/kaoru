package usecase

import (
	"context"
	"fmt"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/production/application/port"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// PublishChecklistTemplate publica una versión nueva de la plantilla del checklist de un sistema.
// Las versiones anteriores quedan: los checklists ya abiertos conservan la suya.
type PublishChecklistTemplate struct {
	d Deps
}

// PublishChecklistTemplateInput es la entrada de PublishChecklistTemplate.
type PublishChecklistTemplateInput struct {
	Actor     shared.Actor
	SystemKey string
	Steps     []domain.StepInput
}

// Execute exige el permiso chk.manage_template. Devuelve CONFLICT si otra publicación simultánea
// tomó la misma versión.
func (uc *PublishChecklistTemplate) Execute(ctx context.Context, in PublishChecklistTemplateInput) (domain.TemplateSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManageTemplate); err != nil {
		return domain.TemplateSnapshot{}, err
	}
	system, err := uc.d.Repository.FindSystem(ctx, in.SystemKey)
	if axis.IsCode(err, axis.NotFoundResource) {
		field := axis.InvalidField("system", "El sistema no existe.")
		if in.SystemKey == "" {
			field = axis.Required("system", "Elige un sistema.")
		}
		return domain.TemplateSnapshot{}, axis.Invalid(field)
	}
	if err != nil {
		return domain.TemplateSnapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.TemplateSnapshot{}, err
	}

	var out domain.TemplateSnapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		version, err := uc.d.Repository.NextTemplateVersion(ctx, system.Key)
		if err != nil {
			return err
		}
		template, err := domain.NewTemplate(domain.NewTemplateParams{
			Id: id, System: system, Version: version, Steps: in.Steps, CreatedBy: in.Actor.Id, Now: uc.d.Clock.Now(),
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertTemplate(ctx, template); err != nil {
			if axis.IsCode(err, axis.DuplicateResource) {
				return ErrVersionTaken
			}
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityTemplate, EntityId: id, Action: port.AuditCreate,
			Field: "version", NewValue: fmt.Sprintf("%s v%d", system.Key, version),
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.LatestTemplate(ctx, system.Key)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.TemplateSnapshot{}, err
	}
	return out, nil
}
