// Package domain modela las ejecuciones: el RUN oficial, que es evidencia de conformidad,
// y el EXP exploratorio, que no lo es.
package domain

import (
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

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// TestCaseRef es el TC que ejecuta un RUN. SystemKey solo viene al registrar, para validar el sistema.
type TestCaseRef struct {
	Id        string
	Code      string
	Title     string
	Retired   bool
	SystemKey string
}

// BuildRef es el BUILD del catálogo sobre el que se ejecuta.
type BuildRef struct {
	Id        string
	Code      string
	SystemKey string
	Version   string
	Checksum  string
}

// EnvironmentRef es el ambiente del catálogo en el que se ejecuta.
type EnvironmentRef struct {
	Id         string
	Code       string
	Name       string
	SystemKey  string
	Production bool
	Retired    bool
}

// RunSnapshot es el contenido de un RUN.
type RunSnapshot struct {
	Id       string
	Code     shared.Code
	TestCase TestCaseRef
	// BuildVersion, BuildChecksum y EnvironmentName se copian del BUILD y del ENV al registrar:
	// son la evidencia tal como era al ejecutar, aunque después el ambiente cambie de nombre.
	BuildVersion    string
	BuildChecksum   string
	EnvironmentName string
	// El BUILD y el ENV del catálogo; vacíos en los RUN anteriores al catálogo.
	BuildId         string
	BuildCode       string
	EnvironmentId   string
	EnvironmentCode string
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
	Id          string
	Code        shared.Code
	TestCase    TestCaseRef
	Build       BuildRef
	Environment EnvironmentRef
	Result      string
	Notes       string
	ExecutedBy  UserRef
	ExecutedAt  time.Time
	Now         time.Time
}

// NewRun valida y registra un RUN sobre un BUILD y un ENV del mismo sistema que el TC.
// Un ambiente de producción no admite RUN: QA no ejecuta en producción.
func NewRun(p NewRunParams) (Run, error) {
	if p.TestCase.Retired {
		return Run{}, ErrTestCaseRetired
	}
	var errs axis.FieldErrors
	s := RunSnapshot{
		Id: p.Id, Code: p.Code, TestCase: p.TestCase,
		BuildVersion: p.Build.Version, BuildChecksum: p.Build.Checksum, EnvironmentName: p.Environment.Name,
		BuildId: p.Build.Id, BuildCode: p.Build.Code, EnvironmentId: p.Environment.Id, EnvironmentCode: p.Environment.Code,
		Result: Result(p.Result), Notes: strings.TrimSpace(p.Notes),
		ExecutedBy: p.ExecutedBy, ExecutedAt: p.ExecutedAt.UTC(), CreatedAt: p.Now,
	}
	validCatalog(p.Build, p.Environment, p.TestCase.SystemKey, "el TC", &errs)
	if p.Environment.Production {
		errs.Add(axis.InvalidField("environment", "QA no ejecuta en producción: elige un ambiente de prueba."))
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

// RequirementRef es el REQ que explora un EXP. SystemKey solo viene al registrar, para validar el sistema.
type RequirementRef struct {
	Id        string
	Code      string
	SystemKey string
}

// ExplorationSnapshot es el contenido de un EXP.
type ExplorationSnapshot struct {
	Id              string
	Code            shared.Code
	Requirement     RequirementRef
	BuildVersion    string
	EnvironmentName string
	// El BUILD y el ENV del catálogo; vacíos en los EXP anteriores al catálogo.
	BuildId         string
	BuildCode       string
	EnvironmentId   string
	EnvironmentCode string
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
	Id            string
	Code          shared.Code
	Requirement   RequirementRef
	Build         BuildRef
	Environment   EnvironmentRef
	ManualChanges string
	Findings      string
	ExecutedBy    UserRef
	ExecutedAt    time.Time
	Now           time.Time
}

// NewExploration valida y registra un EXP sobre un BUILD y un ENV del sistema del REQ.
func NewExploration(p NewExplorationParams) (Exploration, error) {
	var errs axis.FieldErrors
	s := ExplorationSnapshot{
		Id: p.Id, Code: p.Code, Requirement: p.Requirement,
		BuildVersion: p.Build.Version, EnvironmentName: p.Environment.Name,
		BuildId: p.Build.Id, BuildCode: p.Build.Code, EnvironmentId: p.Environment.Id, EnvironmentCode: p.Environment.Code,
		ManualChanges: strings.TrimSpace(p.ManualChanges), Findings: strings.TrimSpace(p.Findings),
		ExecutedBy: p.ExecutedBy, ExecutedAt: p.ExecutedAt.UTC(), CreatedAt: p.Now, UpdatedAt: p.Now,
	}
	validCatalog(p.Build, p.Environment, p.Requirement.SystemKey, "el REQ", &errs)
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

// validCatalog comprueba que el BUILD y el ENV sean del sistema de lo que se ejecuta (owner: «el TC»
// o «el REQ») y que el ambiente siga en uso.
func validCatalog(build BuildRef, environment EnvironmentRef, systemKey, owner string, errs *axis.FieldErrors) {
	if build.SystemKey != systemKey {
		errs.Add(axis.InvalidField("build", "El BUILD es de otro sistema que "+owner+"."))
	}
	switch {
	case environment.SystemKey != systemKey:
		errs.Add(axis.InvalidField("environment", "El ambiente es de otro sistema que "+owner+"."))
	case environment.Retired:
		errs.Add(axis.InvalidField("environment", "El ambiente está retirado: no admite ejecuciones."))
	}
}

func validExecutedAt(executedAt, now time.Time, errs *axis.FieldErrors) {
	switch {
	case executedAt.IsZero():
		errs.Add(axis.Required("executedAt", "Indica cuándo se ejecutó."))
	case executedAt.After(now):
		errs.Add(axis.InvalidField("executedAt", "La fecha de ejecución no puede ser futura."))
	}
}
