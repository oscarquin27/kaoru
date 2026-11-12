// Package usecase contiene los casos de uso de bundle.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionBuild es el permiso para armar un PKG: crearlo, editarlo y cambiar sus BUILD y REQ.
const PermissionBuild = "pkg.build"

// Errores de los casos de uso.
var (
	ErrPackageNotFound     = axis.New(axis.NotFoundResource, "El PKG no existe.")
	ErrBuildNotFound       = axis.New(axis.NotFoundResource, "El BUILD no existe.")
	ErrRequirementNotFound = axis.New(axis.NotFoundResource, "El REQ no existe.")
)

// Campos con los que se audita el contenido de un PKG.
const (
	fieldBuilds       = "builds"
	fieldRequirements = "requirements"
)

// Motivos por los que la selección de regresión confirmada deja de valer.
const (
	reasonContentChanged = "Cambió el contenido del PKG."
	reasonBackToAssembly = "El PKG volvió a armado."
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository   port.Repository
	Workflow     port.Workflow
	Builds       port.Builds
	Requirements port.Requirements
	Regression   port.Regression
	Facts        port.Facts
	AuditTrail   port.AuditTrail
	UnitOfWork   uow.Factory
	Clock        clock.Clock
	IdGenerator  idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreatePackage            *CreatePackage
	GetPackage               *GetPackage
	ListPackages             *ListPackages
	UpdatePackage            *UpdatePackage
	TransitionPackage        *TransitionPackage
	AddPackageBuild          *AddPackageBuild
	RemovePackageBuild       *RemovePackageBuild
	AddPackageRequirement    *AddPackageRequirement
	RemovePackageRequirement *RemovePackageRequirement
	FindPackage              *FindPackage
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Workflow == nil || d.Builds == nil || d.Requirements == nil || d.Regression == nil ||
		d.Facts == nil || d.AuditTrail == nil || d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("bundle: faltan dependencias")
	}
	return &UseCases{
		CreatePackage:            &CreatePackage{d: d},
		GetPackage:               &GetPackage{d: d},
		ListPackages:             &ListPackages{d: d},
		UpdatePackage:            &UpdatePackage{d: d},
		TransitionPackage:        &TransitionPackage{d: d},
		AddPackageBuild:          &AddPackageBuild{d: d},
		RemovePackageBuild:       &RemovePackageBuild{d: d},
		AddPackageRequirement:    &AddPackageRequirement{d: d},
		RemovePackageRequirement: &RemovePackageRequirement{d: d},
		FindPackage:              &FindPackage{d: d},
	}, nil
}

// TransitionReadiness es una transición posible con lo que le falta.
type TransitionReadiness struct {
	To domain.State
	// Conditions incluye las que bloquean y las que solo avisan.
	Conditions []domain.Condition
}

// PackageOutput es un PKG tal como lo devuelven los casos de uso.
type PackageOutput struct {
	Package domain.Snapshot
	// Missing es «qué falta para avanzar», por cada transición que sale del estado actual.
	// En la lista va vacío: se calcula solo en la ficha.
	Missing []TransitionReadiness
}

// lockPackage bloquea la entidad y la lee: se usa dentro de la transacción de todo caso de uso que
// escribe, para que dos cambios simultáneos sobre la misma entidad no se pisen.
func lockPackage(ctx context.Context, repo port.Repository, raw string) (domain.Package, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Package{}, ErrPackageNotFound
	}
	if err := repo.Lock(ctx, code); err != nil {
		if axis.IsCode(err, axis.NotFoundResource) {
			return domain.Package{}, ErrPackageNotFound
		}
		return domain.Package{}, err
	}
	return findPackage(ctx, repo, raw)
}

// findPackage valida el código y lee el PKG; un código mal formado tampoco existe.
func findPackage(ctx context.Context, repo port.Repository, raw string) (domain.Package, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Package{}, ErrPackageNotFound
	}
	pkg, err := repo.FindByCode(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Package{}, ErrPackageNotFound
	}
	return pkg, err
}

// gatherFacts junta los hechos con los que se evalúan las reglas de bloqueo del PKG.
func gatherFacts(ctx context.Context, d Deps, pkg domain.Package) (domain.Facts, error) {
	s := pkg.Snapshot()
	facts := domain.Facts{Builds: len(s.Builds), Requirements: len(s.Requirements)}
	var err error
	if facts.Regression, err = d.Facts.Regression(ctx, s.Code); err != nil {
		return domain.Facts{}, err
	}
	codes := make([]string, len(s.Requirements))
	for i, r := range s.Requirements {
		codes[i] = r.Code
	}
	if facts.OpenDefects, err = d.Facts.OpenDefects(ctx, codes); err != nil {
		return domain.Facts{}, err
	}
	if facts.DeployedToProduction, err = d.Facts.DeployedToProduction(ctx, s.Code); err != nil {
		return domain.Facts{}, err
	}
	return facts, nil
}

// readPackage lee el PKG ya guardado y calcula qué le falta para cada transición que sale de su estado.
func readPackage(ctx context.Context, d Deps, raw string) (PackageOutput, error) {
	pkg, err := findPackage(ctx, d.Repository, raw)
	if err != nil {
		return PackageOutput{}, err
	}
	s := pkg.Snapshot()
	transitions, err := d.Workflow.Outgoing(ctx, s.State.Key)
	if err != nil {
		return PackageOutput{}, err
	}
	facts, err := gatherFacts(ctx, d, pkg)
	if err != nil {
		return PackageOutput{}, err
	}
	out := PackageOutput{Package: s, Missing: make([]TransitionReadiness, len(transitions))}
	for i, t := range transitions {
		out.Missing[i] = TransitionReadiness{To: t.To, Conditions: domain.Unmet(t.Rules, facts)}
	}
	return out, nil
}

// changeContent es el esqueleto de los cuatro casos de uso que cambian el contenido de un PKG:
// comprueba el permiso, bloquea el PKG, aplica el cambio y, si de verdad cambió algo, guarda la
// fecha de cambio, audita y avisa a la regresión de que su confirmación ya no corresponde.
func changeContent(
	ctx context.Context, d Deps, actor shared.Actor, code string,
	apply func(ctx context.Context, pkg *domain.Package) (record port.AuditRecord, changed bool, err error),
) (PackageOutput, error) {
	if err := d.Workflow.Require(ctx, actor.Roles, PermissionBuild); err != nil {
		return PackageOutput{}, err
	}
	var out PackageOutput
	err := d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		pkg, err := lockPackage(ctx, d.Repository, code)
		if err != nil {
			return err
		}
		record, changed, err := apply(ctx, &pkg)
		if err != nil {
			return err
		}
		if changed {
			s := pkg.Snapshot()
			if err := d.Repository.Update(ctx, pkg); err != nil {
				return err
			}
			record.Actor, record.PackageId, record.Code, record.Action = actor.Id, s.Id, s.Code, port.AuditUpdate
			if err := d.AuditTrail.Record(ctx, record); err != nil {
				return err
			}
			if err := d.Regression.Reopen(ctx, actor.Id, s.Code, reasonContentChanged); err != nil {
				return err
			}
		}
		out, err = readPackage(ctx, d, code)
		return err
	})
	if err != nil {
		return PackageOutput{}, err
	}
	return out, nil
}

// findBuild lee el BUILD del catálogo; el que no existe es ErrBuildNotFound.
func findBuild(ctx context.Context, d Deps, code string) (domain.Build, error) {
	build, err := d.Builds.Find(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Build{}, ErrBuildNotFound
	}
	return build, err
}

// findRequirement lee el REQ; el que no existe es ErrRequirementNotFound.
func findRequirement(ctx context.Context, d Deps, code string) (domain.Requirement, error) {
	requirement, err := d.Requirements.Find(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Requirement{}, ErrRequirementNotFound
	}
	return requirement, err
}
