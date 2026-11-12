package domain

import (
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// ErrAlreadyReviewed es el error de revisar dos veces el mismo DIAG.
var ErrAlreadyReviewed = axis.New(axis.InvalidAction, "El DIAG ya está revisado. Si el resultado cambió, se entrega otro DIAG.")

// DeploymentRef es el despliegue al que se refiere un DIAG o un SMOKE.
// Package es nil si el despliegue fue de un BUILD suelto.
type DeploymentRef struct {
	Id          string
	Code        string
	Environment EnvironmentRef
	Package     *PackageRef
	ExecutedAt  time.Time
}

// validDeployment exige que el despliegue sea de un PKG en un ambiente de producción.
func validDeployment(d DeploymentRef, errs *axis.FieldErrors) {
	switch {
	case d.Package == nil:
		errs.Add(axis.InvalidField("deployment", "El despliegue no es de un PKG."))
	case d.Environment.Kind != KindProduction:
		errs.Add(axis.InvalidField("deployment", "El despliegue no es en un ambiente de producción."))
	}
}

// ReviewResult es lo que QA marca al revisar un DIAG.
type ReviewResult string

// Resultados de la revisión de un DIAG.
const (
	ReviewConforming    ReviewResult = "conforming"
	ReviewNonConforming ReviewResult = "non_conforming"
)

// Review es la revisión de QA de un DIAG.
type Review struct {
	Result     ReviewResult
	Notes      string
	ReviewedBy UserRef
	ReviewedAt time.Time
}

// DiagnosticSnapshot es el contenido de un DIAG; Review es nil mientras QA no lo revise.
type DiagnosticSnapshot struct {
	Id          string
	Code        shared.Code
	Package     PackageRef
	Deployment  DeploymentRef
	Notes       string
	SubmittedBy UserRef
	CreatedAt   time.Time
	Review      *Review
}

// Diagnostic es un DIAG-: lo que dev u ops entregan del diagnóstico posterior a un despliegue.
type Diagnostic struct {
	s DiagnosticSnapshot
}

// NewDiagnosticParams son los datos para entregar un DIAG.
type NewDiagnosticParams struct {
	Id          string
	Code        shared.Code
	Deployment  DeploymentRef
	Notes       string
	SubmittedBy shared.UserId
	Now         time.Time
}

// NewDiagnostic valida y entrega un DIAG, sin revisar.
func NewDiagnostic(p NewDiagnosticParams) (Diagnostic, error) {
	var errs axis.FieldErrors
	validDeployment(p.Deployment, &errs)
	if err := errs.Err(); err != nil {
		return Diagnostic{}, err
	}
	return Diagnostic{s: DiagnosticSnapshot{
		Id: p.Id, Code: p.Code, Package: *p.Deployment.Package, Deployment: p.Deployment,
		Notes: strings.TrimSpace(p.Notes), SubmittedBy: UserRef{Id: p.SubmittedBy}, CreatedAt: p.Now,
	}}, nil
}

// RehydrateDiagnostic arma un DIAG ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateDiagnostic(s DiagnosticSnapshot) Diagnostic { return Diagnostic{s: s} }

// Snapshot devuelve el contenido del DIAG.
func (d Diagnostic) Snapshot() DiagnosticSnapshot { return d.s }

// Review marca el resultado del DIAG, una sola vez. Un «no conforme» explica qué no lo está.
func (d *Diagnostic) Review(result, notes string, by shared.UserId, now time.Time) (Review, error) {
	if d.s.Review != nil {
		return Review{}, ErrAlreadyReviewed
	}
	var errs axis.FieldErrors
	review := Review{Result: ReviewResult(result), Notes: strings.TrimSpace(notes), ReviewedBy: UserRef{Id: by}, ReviewedAt: now}
	switch review.Result {
	case ReviewConforming:
	case ReviewNonConforming:
		if review.Notes == "" {
			errs.Add(axis.Required("notes", "Explica qué no está conforme."))
		}
	case "":
		errs.Add(axis.Required("result", "Indica el resultado."))
	default:
		errs.Add(axis.InvalidField("result", "El resultado debe ser conforme o no conforme."))
	}
	if err := errs.Err(); err != nil {
		return Review{}, err
	}
	d.s.Review = &review
	return review, nil
}

// SmokeResult es el resultado de una prueba de humo.
type SmokeResult string

// Resultados de una prueba de humo.
const (
	SmokePassed SmokeResult = "passed"
	SmokeFailed SmokeResult = "failed"
)

// SmokeTestSnapshot es el contenido de un SMOKE.
type SmokeTestSnapshot struct {
	Id         string
	Code       shared.Code
	Package    PackageRef
	Deployment DeploymentRef
	Result     SmokeResult
	Notes      string
	ExecutedBy UserRef
	CreatedAt  time.Time
}

// SmokeTest es un SMOKE-: el resultado de la prueba de humo posterior a un despliegue. No se corrige: se registra otra.
type SmokeTest struct {
	s SmokeTestSnapshot
}

// NewSmokeTestParams son los datos para registrar un SMOKE.
type NewSmokeTestParams struct {
	Id         string
	Code       shared.Code
	Deployment DeploymentRef
	Result     string
	Notes      string
	ExecutedBy shared.UserId
	Now        time.Time
}

// NewSmokeTest valida y registra un SMOKE. Una prueba fallida dice qué falló.
func NewSmokeTest(p NewSmokeTestParams) (SmokeTest, error) {
	var errs axis.FieldErrors
	validDeployment(p.Deployment, &errs)
	result, notes := SmokeResult(p.Result), strings.TrimSpace(p.Notes)
	switch result {
	case SmokePassed:
	case SmokeFailed:
		if notes == "" {
			errs.Add(axis.Required("notes", "Explica qué falló."))
		}
	case "":
		errs.Add(axis.Required("result", "Indica el resultado."))
	default:
		errs.Add(axis.InvalidField("result", "El resultado debe ser aprobada o fallida."))
	}
	if err := errs.Err(); err != nil {
		return SmokeTest{}, err
	}
	return SmokeTest{s: SmokeTestSnapshot{
		Id: p.Id, Code: p.Code, Package: *p.Deployment.Package, Deployment: p.Deployment,
		Result: result, Notes: notes, ExecutedBy: UserRef{Id: p.ExecutedBy}, CreatedAt: p.Now,
	}}, nil
}

// RehydrateSmokeTest arma un SMOKE ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateSmokeTest(s SmokeTestSnapshot) SmokeTest { return SmokeTest{s: s} }

// Snapshot devuelve el contenido del SMOKE.
func (t SmokeTest) Snapshot() SmokeTestSnapshot { return t.s }

// PackageFacts es lo que un PKG necesita saber de producción para sus reglas de bloqueo.
// Cuenta lo último de cada cosa: un checklist, un DIAG o un SMOKE más reciente reemplaza al anterior.
type PackageFacts struct {
	// ChecklistExists es false si el PKG no tiene ningún checklist abierto.
	ChecklistExists bool
	// ChecklistPending son los pasos sin firmar del último checklist.
	ChecklistPending int
	// Diagnostic es el estado del último DIAG: vacío si no hay ninguno, «pending» si falta revisarlo,
	// o el resultado de su revisión.
	Diagnostic string
	// Smoke es el resultado del último SMOKE; vacío si no hay ninguno.
	Smoke string
}

// DiagnosticPending es el estado de un DIAG entregado que QA todavía no revisa.
const DiagnosticPending = "pending"
