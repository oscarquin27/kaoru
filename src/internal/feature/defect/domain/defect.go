// Package domain modela el DEF: un defecto ligado a un REQ y al RUN que lo reprodujo.
package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Severity es la severidad de un defecto.
type Severity string

// Severidades.
const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

const (
	maxTitleLength = 200
	// runPassed es el resultado de un RUN que no reproduce nada.
	runPassed = "passed"
	// RuleSeverityIsNot es la regla que impide una transición para cierta severidad.
	RuleSeverityIsNot = "defect_severity_is_not"
)

// State es el estado del DEF en su flujo configurable.
type State struct {
	Id      string
	Key     string
	Name    string
	IsFinal bool
}

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// RequirementRef es el REQ al que pertenece el DEF.
type RequirementRef struct {
	Id   string
	Code string
}

// RunRef es el RUN que reprodujo el defecto.
type RunRef struct {
	Id     string
	Code   string
	Result string
	// TestCaseId es el TC que ejecutó el RUN.
	TestCaseId string
}

// Snapshot es el contenido de un DEF.
type Snapshot struct {
	Id          string
	Code        shared.Code
	Title       string
	Description string
	Requirement RequirementRef
	Run         RunRef
	Severity    Severity
	State       State
	// Assignee es quien lo corrige; nil si nadie lo tiene.
	Assignee  *UserRef
	CreatedBy UserRef
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Defect es un DEF.
type Defect struct {
	s Snapshot
}

// NewParams son los datos para abrir un DEF. RequirementTestCases son los TC que validan el REQ.
type NewParams struct {
	RequirementTestCases []string
	Id                   string
	Code                 shared.Code
	Title                string
	Description          string
	Requirement          RequirementRef
	Run                  RunRef
	Severity             string
	State                State
	Assignee             *UserRef
	CreatedBy            UserRef
	Now                  time.Time
}

// New valida y abre un DEF. Exige un RUN que no esté aprobado: un hallazgo de una
// exploración se reproduce en un RUN antes de abrirse como defecto.
func New(p NewParams) (Defect, error) {
	var errs axis.FieldErrors
	s := Snapshot{
		Id: p.Id, Code: p.Code, Title: validTitle(p.Title, &errs),
		Description: validDescription(p.Description, &errs),
		Requirement: p.Requirement, Run: p.Run, Severity: validSeverity(p.Severity, &errs),
		State: p.State, Assignee: p.Assignee, CreatedBy: p.CreatedBy, CreatedAt: p.Now, UpdatedAt: p.Now,
	}
	switch {
	case p.Run.Result == runPassed:
		errs.Add(axis.InvalidField("run", "Ese RUN está aprobado. Un DEF necesita el RUN que lo reprodujo."))
	case !slices.Contains(p.RequirementTestCases, p.Run.TestCaseId):
		// Sin esto la matriz mostraría un defecto en un REQ con la evidencia de otro.
		errs.Add(axis.InvalidField("run", "Ese RUN es de un TC que no valida este REQ."))
	}
	if err := errs.Err(); err != nil {
		return Defect{}, err
	}
	return Defect{s: s}, nil
}

// Rehydrate arma un DEF ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Defect { return Defect{s: s} }

// Snapshot devuelve el contenido del DEF.
func (d Defect) Snapshot() Snapshot { return d.s }

// AssigneeEdit cambia a quién está asignado: Set dice si se toca; Value nil lo quita.
type AssigneeEdit struct {
	Set   bool
	Value *UserRef
}

// Edit son los campos a cambiar; un puntero nil deja el campo como está.
type Edit struct {
	Title       *string
	Description *string
	Severity    *string
	Assignee    AssigneeEdit
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (d *Defect) Apply(e Edit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := d.s
	if e.Title != nil {
		next.Title = validTitle(*e.Title, &errs)
	}
	if e.Description != nil {
		next.Description = validDescription(*e.Description, &errs)
	}
	if e.Severity != nil {
		next.Severity = validSeverity(*e.Severity, &errs)
	}
	if e.Assignee.Set {
		next.Assignee = e.Assignee.Value
	}
	if err := errs.Err(); err != nil {
		return nil, err
	}
	var changes []FieldChange
	add := func(field, old, new string) {
		if old != new {
			changes = append(changes, FieldChange{Field: field, Old: old, New: new})
		}
	}
	add("title", d.s.Title, next.Title)
	add("description", d.s.Description, next.Description)
	add("severity", string(d.s.Severity), string(next.Severity))
	add("assignee", userId(d.s.Assignee), userId(next.Assignee))
	if len(changes) > 0 {
		next.UpdatedAt = now
		d.s = next
	}
	return changes, nil
}

// MoveTo pasa el DEF a otro estado. Que la transición valga lo deciden el flujo y sus reglas.
func (d *Defect) MoveTo(state State, now time.Time) FieldChange {
	change := FieldChange{Field: "state", Old: d.s.State.Key, New: state.Key}
	d.s.State = state
	d.s.UpdatedAt = now
	return change
}

// DaysInState es el tiempo que lleva el DEF en su estado actual. enteredAt es la fecha de
// su última transición; si nunca cambió de estado, cuenta desde que se abrió. No hay SLA: solo se mide.
func (d Defect) DaysInState(enteredAt *time.Time, now time.Time) float64 {
	since := d.s.CreatedAt
	if enteredAt != nil {
		since = *enteredAt
	}
	if now.Before(since) {
		return 0
	}
	return now.Sub(since).Hours() / 24
}

// Rule es una regla de bloqueo de una transición, tal como la configura workflow.
type Rule struct {
	Key    string
	Params map[string]any
	Blocks bool
}

// Blockers evalúa las reglas de una transición y devuelve los mensajes de las que la impiden.
func (d Defect) Blockers(rules []Rule) []string {
	var messages []string
	for _, rule := range rules {
		if !rule.Blocks {
			continue
		}
		switch rule.Key {
		case RuleSeverityIsNot:
			if severity, _ := rule.Params["severity"].(string); severity == string(d.s.Severity) {
				messages = append(messages, fmt.Sprintf("Un DEF de severidad %s no admite este paso.", severityName(d.s.Severity)))
			}
		default:
			// Una regla que el dominio no conoce bloquea: mejor parar que dejar pasar sin evaluar.
			messages = append(messages, fmt.Sprintf("La regla «%s» no se puede evaluar. Avisa a QA.", rule.Key))
		}
	}
	return messages
}

func validTitle(raw string, errs *axis.FieldErrors) string {
	title := strings.TrimSpace(raw)
	switch {
	case title == "":
		errs.Add(axis.Required("title", "Escribe un título."))
	case utf8.RuneCountInString(title) > maxTitleLength:
		errs.Add(axis.InvalidField("title", "El título no puede pasar de 200 caracteres."))
	}
	return title
}

func validDescription(raw string, errs *axis.FieldErrors) string {
	description := strings.TrimSpace(raw)
	if description == "" {
		errs.Add(axis.Required("description", "Describe qué pasó y cómo reproducirlo."))
	}
	return description
}

func validSeverity(raw string, errs *axis.FieldErrors) Severity {
	severity := Severity(raw)
	switch {
	case raw == "":
		errs.Add(axis.Required("severity", "Elige la severidad."))
	case severity != SeverityHigh && severity != SeverityMedium && severity != SeverityLow:
		errs.Add(axis.InvalidField("severity", "La severidad debe ser alta, media o baja."))
	}
	return severity
}

func severityName(s Severity) string {
	switch s {
	case SeverityHigh:
		return "alta"
	case SeverityMedium:
		return "media"
	default:
		return "baja"
	}
}

func userId(u *UserRef) string {
	if u == nil {
		return ""
	}
	return string(u.Id)
}
