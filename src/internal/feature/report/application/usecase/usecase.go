// Package usecase contiene los casos de uso de report.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/report/application/port"
	"kaoru/internal/feature/report/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionIssue es el permiso de emitir un RPT, que tiene QA.
const PermissionIssue = "rpt.issue"

// ErrReportNotFound es el error de un RPT que no existe.
var ErrReportNotFound = axis.New(axis.NotFoundResource, "El informe no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Evidence    port.Evidence
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	IssueReport *IssueReport
	GetReport   *GetReport
	ListReports *ListReports
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Evidence == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("report: faltan dependencias")
	}
	return &UseCases{
		IssueReport: &IssueReport{d: d},
		GetReport:   &GetReport{d: d},
		ListReports: &ListReports{d: d},
	}, nil
}

// findReport valida el código y lee el RPT; un código mal formado tampoco existe.
func findReport(ctx context.Context, repo port.Repository, raw string) (domain.Report, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Report{}, ErrReportNotFound
	}
	report, err := repo.FindByCode(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Report{}, ErrReportNotFound
	}
	return report, err
}
