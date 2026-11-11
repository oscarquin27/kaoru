// Package domain modela las ejecuciones: el RUN oficial, que es evidencia de conformidad,
// y el EXP exploratorio, que no lo es.
package domain

import (
	"regexp"
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Result es el resultado de un RUN.
type Result string

// Resultados de un RUN.
const (
	ResultPassed  Result = "passed"
	ResultFailed  Result = "failed"
	ResultBlocked Result = "blocked"
)

// ErrTestCaseRetired es el error de registrar un RUN sobre un TC retirado.
var ErrTestCaseRetired = axis.New(axis.InvalidAction, "El TC está retirado: no admite ejecuciones nuevas.")

var checksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// TestCaseRef es el TC que ejecuta un RUN.
type TestCaseRef struct {
	Id      string
	Code    string
	Title   string
	Retired bool
}

// RunSnapshot es el contenido de un RUN.
type RunSnapshot struct {
	Id       string
	Code     shared.Code
	TestCase TestCaseRef
	// BuildVersion y BuildChecksum identifican el BUILD exacto sobre el que se ejecutó.
	BuildVersion    string
	BuildChecksum   string
	EnvironmentName string
	Result          Result
	Notes           string
	ExecutedBy      UserRef
	ExecutedAt      time.Time
	CreatedAt       time.Time
}

// Run es un RUN-: la ejecución oficial de un TC sobre un BUILD sin modificar.
// No tiene métodos para cambiar su contenido: es inmutable por regla de negocio.
type Run struct {
	s RunSnapshot
}

// NewRunParams son los datos para registrar un RUN.
type NewRunParams struct {
	Id              string
	Code            shared.Code
	TestCase        TestCaseRef
	BuildVersion    string
	BuildChecksum   string
	EnvironmentName string
	Result          string
	Notes           string
	ExecutedBy      UserRef
	ExecutedAt      time.Time
	Now             time.Time
}

// NewRun valida y registra un RUN.
func NewRun(p NewRunParams) (Run, error) {
	if p.TestCase.Retired {
		return Run{}, ErrTestCaseRetired
	}
	var errs axis.FieldErrors
	s := RunSnapshot{
		Id: p.Id, Code: p.Code, TestCase: p.TestCase,
		BuildVersion:    validRequired("buildVersion", p.BuildVersion, "Indica la versión del BUILD.", &errs),
		BuildChecksum:   strings.ToLower(strings.TrimSpace(p.BuildChecksum)),
		EnvironmentName: validRequired("environmentName", p.EnvironmentName, "Indica el ambiente.", &errs),
		Result:          Result(p.Result), Notes: strings.TrimSpace(p.Notes),
		ExecutedBy: p.ExecutedBy, ExecutedAt: p.ExecutedAt.UTC(), CreatedAt: p.Now,
	}
	switch {
	case s.BuildChecksum == "":
		errs.Add(axis.Required("buildChecksum", "Pega el checksum SHA-256 del BUILD."))
	case !checksumPattern.MatchString(s.BuildChecksum):
		errs.Add(axis.InvalidField("buildChecksum", "El checksum debe ser un SHA-256: 64 caracteres hexadecimales."))
	}
	switch {
	case p.Result == "":
		errs.Add(axis.Required("result", "Indica el resultado."))
	case s.Result != ResultPassed && s.Result != ResultFailed && s.Result != ResultBlocked:
		errs.Add(axis.InvalidField("result", "El resultado debe ser aprobado, fallido o bloqueado."))
	}
	validExecutedAt(p.ExecutedAt, p.Now, &errs)
	if err := errs.Err(); err != nil {
		return Run{}, err
	}
	return Run{s: s}, nil
}

// RehydrateRun arma un RUN ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateRun(s RunSnapshot) Run { return Run{s: s} }

// Snapshot devuelve el contenido del RUN.
func (r Run) Snapshot() RunSnapshot { return r.s }

// RequirementRef es el REQ que explora un EXP.
type RequirementRef struct {
	Id   string
	Code string
}

// ExplorationSnapshot es el contenido de un EXP.
type ExplorationSnapshot struct {
	Id              string
	Code            shared.Code
	Requirement     RequirementRef
	BuildVersion    string
	EnvironmentName string
	// ManualChanges son los cambios hechos a mano sobre el BUILD o el ambiente.
	ManualChanges string
	Findings      string
	ExecutedBy    UserRef
	ExecutedAt    time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Exploration es un EXP-: una ejecución exploratoria. No cuenta como evidencia de conformidad.
type Exploration struct {
	s ExplorationSnapshot
}

// NewExplorationParams son los datos para registrar un EXP.
type NewExplorationParams struct {
	Id              string
	Code            shared.Code
	Requirement     RequirementRef
	BuildVersion    string
	EnvironmentName string
	ManualChanges   string
	Findings        string
	ExecutedBy      UserRef
	ExecutedAt      time.Time
	Now             time.Time
}

// NewExploration valida y registra un EXP.
func NewExploration(p NewExplorationParams) (Exploration, error) {
	var errs axis.FieldErrors
	s := ExplorationSnapshot{
		Id: p.Id, Code: p.Code, Requirement: p.Requirement,
		BuildVersion:    validRequired("buildVersion", p.BuildVersion, "Indica la versión del BUILD.", &errs),
		EnvironmentName: validRequired("environmentName", p.EnvironmentName, "Indica el ambiente.", &errs),
		ManualChanges:   strings.TrimSpace(p.ManualChanges), Findings: strings.TrimSpace(p.Findings),
		ExecutedBy: p.ExecutedBy, ExecutedAt: p.ExecutedAt.UTC(), CreatedAt: p.Now, UpdatedAt: p.Now,
	}
	validExecutedAt(p.ExecutedAt, p.Now, &errs)
	if err := errs.Err(); err != nil {
		return Exploration{}, err
	}
	return Exploration{s: s}, nil
}

// RehydrateExploration arma un EXP ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateExploration(s ExplorationSnapshot) Exploration { return Exploration{s: s} }

// Snapshot devuelve el contenido del EXP.
func (e Exploration) Snapshot() ExplorationSnapshot { return e.s }

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply completa los cambios manuales o los hallazgos; nil deja el campo como está.
func (e *Exploration) Apply(manualChanges, findings *string, now time.Time) []FieldChange {
	next := e.s
	if manualChanges != nil {
		next.ManualChanges = strings.TrimSpace(*manualChanges)
	}
	if findings != nil {
		next.Findings = strings.TrimSpace(*findings)
	}
	var changes []FieldChange
	if next.ManualChanges != e.s.ManualChanges {
		changes = append(changes, FieldChange{Field: "manualChanges", Old: e.s.ManualChanges, New: next.ManualChanges})
	}
	if next.Findings != e.s.Findings {
		changes = append(changes, FieldChange{Field: "findings", Old: e.s.Findings, New: next.Findings})
	}
	if len(changes) > 0 {
		next.UpdatedAt = now
		e.s = next
	}
	return changes
}

// CoverageTestCase es un TC de un REQ, para contar su cobertura.
type CoverageTestCase struct {
	Id      string
	Code    string
	Retired bool
}

// Coverage cuenta los TC que un REQ exige (los que no están retirados) y devuelve los
// códigos de los que no tienen su último RUN aprobado.
func Coverage(testCases []CoverageTestCase, latest map[string]Result) (required int, pending []string) {
	for _, tc := range testCases {
		if tc.Retired {
			continue
		}
		required++
		if latest[tc.Id] != ResultPassed {
			pending = append(pending, tc.Code)
		}
	}
	return required, pending
}

func validRequired(field, raw, message string, errs *axis.FieldErrors) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		errs.Add(axis.Required(field, message))
	}
	return value
}

func validExecutedAt(executedAt, now time.Time, errs *axis.FieldErrors) {
	switch {
	case executedAt.IsZero():
		errs.Add(axis.Required("executedAt", "Indica cuándo se ejecutó."))
	case executedAt.After(now):
		errs.Add(axis.InvalidField("executedAt", "La fecha de ejecución no puede ser futura."))
	}
}
