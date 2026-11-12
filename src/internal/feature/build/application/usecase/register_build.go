package usecase

import (
	"context"
	"slices"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/build/application/port"
	"kaoru/internal/feature/build/domain"
	shared "kaoru/internal/shared/domain"
)

// RegisterBuild registra un BUILD con su manifiesto. Después no se edita ni se borra.
type RegisterBuild struct {
	d Deps
}

// RegisterBuildInput es la entrada de RegisterBuild.
type RegisterBuildInput struct {
	Actor     shared.Actor
	SystemKey string
	Version   string
	Checksum  string
	// RequirementCodes son los REQ que incluye el BUILD; los repetidos cuentan una vez.
	RequirementCodes []string
	Migrations       string
	Dependencies     string
	Notes            string
}

// Execute exige el permiso build.register.
func (uc *RegisterBuild) Execute(ctx context.Context, in RegisterBuildInput) (domain.Snapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionRegister); err != nil {
		return domain.Snapshot{}, err
	}
	system, err := uc.d.Repository.FindSystem(ctx, in.SystemKey)
	if axis.IsCode(err, axis.NotFoundResource) {
		field := axis.InvalidField("system", "El sistema no existe.")
		if in.SystemKey == "" {
			field = axis.Required("system", "Elige un sistema.")
		}
		return domain.Snapshot{}, axis.Invalid(field)
	}
	if err != nil {
		return domain.Snapshot{}, err
	}
	requirements, err := uc.findRequirements(ctx, in.RequirementCodes)
	if err != nil {
		return domain.Snapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.Snapshot{}, err
	}
	now := uc.d.Clock.Now()

	var out domain.Snapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, now.Year())
		if err != nil {
			return err
		}
		build, err := domain.New(domain.NewParams{
			Id: id, Code: code, System: system, Version: in.Version, Checksum: in.Checksum, Requirements: requirements,
			Migrations: in.Migrations, Dependencies: in.Dependencies, Notes: in.Notes, BuiltBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		checksum := build.Snapshot().Checksum
		// El mismo artefacto no se registra dos veces.
		existing, err := uc.d.Repository.FindCodeByChecksum(ctx, checksum)
		if err == nil {
			return axis.Invalid(axis.InvalidField("checksum", "Ese checksum ya está registrado como "+existing.String()+"."))
		}
		if !axis.IsCode(err, axis.NotFoundResource) {
			return err
		}
		if err := uc.d.Repository.Insert(ctx, build); err != nil {
			// Otro registro con el mismo checksum entró entre la comprobación y el alta.
			if axis.IsCode(err, axis.DuplicateResource) {
				return axis.Invalid(axis.InvalidField("checksum", "Ese checksum ya está registrado."))
			}
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, BuildId: id, Code: code, Field: "checksum", NewValue: checksum,
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindByCode(ctx, code)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}

// findRequirements busca cada REQ del manifiesto; uno que no existe es un error del campo.
func (uc *RegisterBuild) findRequirements(ctx context.Context, codes []string) ([]domain.RequirementRef, error) {
	var requirements []domain.RequirementRef
	var seen []string
	for _, raw := range codes {
		code := strings.TrimSpace(raw)
		if slices.Contains(seen, code) {
			continue
		}
		seen = append(seen, code)
		requirement, err := uc.d.Requirements.Find(ctx, code)
		if axis.IsCode(err, axis.NotFoundResource) {
			return nil, axis.Invalid(axis.InvalidField("requirements", "El REQ «"+code+"» no existe."))
		}
		if err != nil {
			return nil, err
		}
		requirements = append(requirements, requirement)
	}
	return requirements, nil
}
