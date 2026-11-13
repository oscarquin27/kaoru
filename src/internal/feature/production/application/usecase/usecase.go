// Package usecase contiene los casos de uso de production.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/production/application/port"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// Permisos del feature. El checklist es de change management y de ops; QA no firma pasos en
// producción (regla de negocio 6). El DIAG lo entregan dev u ops y lo revisa QA; el SMOKE lo registra QA (regla 7).
const (
	PermissionManageTemplate   = "chk.manage_template"
	PermissionStart            = "chk.start"
	PermissionSign             = "chk.sign"
	PermissionSubmitDiagnostic = "diag.submit"
	PermissionReviewDiagnostic = "diag.review"
	PermissionSmoke            = "smoke.create"
)

// Errores de los casos de uso.
var (
	ErrChecklistNotFound  = axis.New(axis.NotFoundResource, "El checklist no existe.")
	ErrDiagnosticNotFound = axis.New(axis.NotFoundResource, "El DIAG no existe.")
	ErrSmokeTestNotFound  = axis.New(axis.NotFoundResource, "El SMOKE no existe.")
	ErrNoTemplate         = axis.New(axis.InvalidAction, "El sistema del PKG no tiene plantilla de checklist. Publica una antes de abrir el checklist.")
	ErrVersionTaken       = axis.New(axis.Conflict, "Otra persona publicó una versión al mismo tiempo. Revisa la plantilla y vuelve a intentarlo.")
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	PublishChecklistTemplate *PublishChecklistTemplate
	ListChecklistTemplates   *ListChecklistTemplates
	StartChecklist           *StartChecklist
	GetChecklist             *GetChecklist
	ListChecklists           *ListChecklists
	SignChecklistStep        *SignChecklistStep
	SubmitDiagnostic         *SubmitDiagnostic
	GetDiagnostic            *GetDiagnostic
	ListDiagnostics          *ListDiagnostics
	ReviewDiagnostic         *ReviewDiagnostic
	RegisterSmokeTest        *RegisterSmokeTest
	GetSmokeTest             *GetSmokeTest
	ListSmokeTests           *ListSmokeTests
	GetPackageFacts          *GetPackageFacts
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("production: faltan dependencias")
	}
	return &UseCases{
		PublishChecklistTemplate: &PublishChecklistTemplate{d: d},
		ListChecklistTemplates:   &ListChecklistTemplates{d: d},
		StartChecklist:           &StartChecklist{d: d},
		GetChecklist:             &GetChecklist{d: d},
		ListChecklists:           &ListChecklists{d: d},
		SignChecklistStep:        &SignChecklistStep{d: d},
		SubmitDiagnostic:         &SubmitDiagnostic{d: d},
		GetDiagnostic:            &GetDiagnostic{d: d},
		ListDiagnostics:          &ListDiagnostics{d: d},
		ReviewDiagnostic:         &ReviewDiagnostic{d: d},
		RegisterSmokeTest:        &RegisterSmokeTest{d: d},
		GetSmokeTest:             &GetSmokeTest{d: d},
		ListSmokeTests:           &ListSmokeTests{d: d},
		GetPackageFacts:          &GetPackageFacts{d: d},
	}, nil
}

// ChecklistOutput es un CHK tal como lo devuelven los casos de uso, con su avance.
type ChecklistOutput struct {
	Checklist   domain.ChecklistSnapshot
	SignedSteps int
	Complete    bool
}

func toChecklistOutput(c domain.Checklist) ChecklistOutput {
	return ChecklistOutput{Checklist: c.Snapshot(), SignedSteps: c.SignedSteps(), Complete: c.Complete()}
}

// findChecklist valida el código y lee el CHK; un código mal formado tampoco existe.
func findChecklist(ctx context.Context, repo port.Repository, raw string) (domain.Checklist, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Checklist{}, ErrChecklistNotFound
	}
	checklist, err := repo.FindChecklist(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Checklist{}, ErrChecklistNotFound
	}
	return checklist, err
}

// findDiagnostic valida el código y lee el DIAG; un código mal formado tampoco existe.
func findDiagnostic(ctx context.Context, repo port.Repository, raw string) (domain.Diagnostic, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Diagnostic{}, ErrDiagnosticNotFound
	}
	diagnostic, err := repo.FindDiagnostic(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Diagnostic{}, ErrDiagnosticNotFound
	}
	return diagnostic, err
}

// findDeployment lee el despliegue de un DIAG o de un SMOKE; el que falta o no existe es un error del campo.
func findDeployment(ctx context.Context, repo port.Repository, code string) (domain.DeploymentRef, error) {
	deployment, err := repo.FindDeployment(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		field := axis.InvalidField("deployment", "El despliegue no existe.")
		if code == "" {
			field = axis.Required("deployment", "Indica el despliegue.")
		}
		return domain.DeploymentRef{}, axis.Invalid(field)
	}
	return deployment, err
}
