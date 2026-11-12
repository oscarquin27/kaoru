// Package domain reúne las lecturas planas de trazabilidad. Este feature no tiene entidades
// ni escribe nada: cruza lo que guardan los demás para responder en una sola consulta.
package domain

import (
	"time"
)

// System es un sistema bajo prueba.
type System struct {
	Key          string
	Name         string
	DefaultRoute string
}

// StateRef es un estado con el nombre que se muestra.
type StateRef struct {
	Key  string
	Name string
}

// RunSummary es el último RUN de un TC; Code vacío significa que no tiene ninguno.
type RunSummary struct {
	Code            string
	Result          string
	BuildVersion    string
	EnvironmentName string
	ExecutedAt      time.Time
}

// DefectSummary es un defecto en una lista.
type DefectSummary struct {
	Code     string
	Title    string
	Severity string
	State    StateRef
	IsFinal  bool
}

// TestCaseSummary es un TC con su último RUN.
type TestCaseSummary struct {
	Code       string
	Title      string
	IsCritical bool
	Retired    bool
	LastRun    RunSummary
}

// CodeTitle es un par de ID legible y título.
type CodeTitle struct {
	Code  string
	Title string
}

// RiskSummary es el RISK de un REQ; Code vacío significa que no está evaluado.
type RiskSummary struct {
	Code          string
	Level         string
	Justification string
}

// UserRef es un usuario con el nombre que se muestra; Id vacío significa que no hay.
type UserRef struct {
	Id   string
	Name string
}

// RequirementHeader es el encabezado de la ficha de un REQ.
type RequirementHeader struct {
	Id                 string
	Code               string
	Title              string
	Description        string
	AcceptanceCriteria string
	System             CodeTitle
	Change             CodeTitle
	Route              string
	State              StateRef
	Owner              UserRef
	Risk               RiskSummary
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Transition es un cambio de estado ocurrido, leído de la auditoría.
type Transition struct {
	At   time.Time
	From string
	To   string
}

// Estados de una etapa en la línea de vida.
const (
	StepDone    = "done"
	StepCurrent = "current"
	StepPending = "pending"
	StepBlocked = "blocked"
)

// LifecycleStep es una etapa del ciclo en la línea de vida de un REQ.
type LifecycleStep struct {
	State  StateRef
	Status string
	// EnteredAt es cuándo entró por última vez; nil si nunca estuvo.
	EnteredAt *time.Time
	// Days es el tiempo acumulado en la etapa.
	Days float64
}

// Lifecycle arma la línea de vida: las etapas del ciclo en orden, con cuáles se completaron,
// cuál es la actual, cuáles faltan y cuánto tiempo pasó en cada una. Todo se calcula al leer.
//
// states son las etapas con posición, en orden. initial es el estado en el que nació la entidad
// y createdAt cuándo. blockedNext dice si la siguiente etapa está bloqueada por alguna regla.
// Un estado lateral (con defectos, diferido, rechazado) no es una etapa: mientras el REQ está en
// uno, ninguna etapa aparece como actual.
func Lifecycle(states []StateRef, initial string, createdAt time.Time, transitions []Transition,
	current string, blockedNext bool, now time.Time) []LifecycleStep {
	entered := map[string]time.Time{initial: createdAt}
	days := map[string]float64{}
	state, since := initial, createdAt
	for _, t := range transitions {
		days[state] += t.At.Sub(since).Hours() / 24
		state, since = t.To, t.At
		entered[state] = t.At
	}
	if now.After(since) {
		days[state] += now.Sub(since).Hours() / 24
	}

	steps := make([]LifecycleStep, len(states))
	currentIndex := -1
	for i, s := range states {
		steps[i] = LifecycleStep{State: s, Status: StepPending, Days: days[s.Key]}
		if at, ok := entered[s.Key]; ok {
			at := at
			steps[i].EnteredAt = &at
			steps[i].Status = StepDone
		}
		if s.Key == current {
			steps[i].Status = StepCurrent
			currentIndex = i
		}
	}
	if blockedNext && currentIndex >= 0 && currentIndex+1 < len(steps) && steps[currentIndex+1].Status == StepPending {
		steps[currentIndex+1].Status = StepBlocked
	}
	return steps
}

// Condition es una regla que no se cumple: lo que falta para avanzar.
type Condition struct {
	RuleKey string
	Message string
	Blocks  bool
	Related []string
}

// Readiness es una transición posible desde el estado actual, con lo que le falta.
type Readiness struct {
	To         StateRef
	Conditions []Condition
}

// Coverage es la cobertura de un REQ.
type Coverage struct {
	Required int
	Passed   int
}

// Overview es la ficha consolidada de un REQ.
type Overview struct {
	Requirement     RequirementHeader
	Lifecycle       []LifecycleStep
	Missing         []Readiness
	Coverage        Coverage
	FunctionalCases []CodeTitle
	TestCases       []TestCaseSummary
	Defects         []DefectSummary
}

// MatrixFilter acota la matriz. Un campo vacío o nil no filtra.
type MatrixFilter struct {
	ChangeCode string
	SystemKey  string
	StateKey   string
	RiskLevel  string
	OwnerId    string
	// PackageCode y ComponentCode acotan a los REQ de un PKG o que tocan un COMP.
	PackageCode   string
	ComponentCode string
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
}

// MatrixRow es una fila de la matriz: un REQ con uno de sus TC (o sin ninguno).
type MatrixRow struct {
	RequirementCode  string
	RequirementTitle string
	SystemKey        string
	State            StateRef
	ChangeCode       string
	RiskCode         string
	RiskLevel        string
	OwnerName        string
	// Packages son los códigos de los PKG que incluyen el REQ.
	Packages        []string
	FunctionalCases []string
	// TestCase.Code vacío significa que el REQ no tiene TC.
	TestCase TestCaseSummary
	Defects  []DefectSummary
}

// Indicators son los indicadores de todo lo filtrado en la matriz.
type Indicators struct {
	// CoveragePercent es la parte de los TC exigidos cuyo último RUN está aprobado.
	CoveragePercent int
	RequiredTests   int
	PassedTests     int
	// BlockedRequirements son los REQ con algún DEF de severidad alta abierto.
	BlockedRequirements int
	OpenDefects         map[string]int
}

// NewIndicators calcula el porcentaje; sin TC exigidos la cobertura es cero.
func NewIndicators(required, passed, blocked int, openDefects map[string]int) Indicators {
	percent := 0
	if required > 0 {
		percent = passed * 100 / required
	}
	return Indicators{
		CoveragePercent: percent, RequiredTests: required, PassedTests: passed,
		BlockedRequirements: blocked, OpenDefects: openDefects,
	}
}

// SearchHit es un resultado de la búsqueda global.
type SearchHit struct {
	Kind  string
	Code  string
	Title string
	// State.Key vacío significa que la entidad no tiene ciclo de vida.
	State StateRef
}
