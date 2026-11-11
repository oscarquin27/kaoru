// Package domain modela el REQ: el requerimiento, sus enlaces, su contrato de entrada
// y las reglas que deciden si puede avanzar.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

const maxTitleLength = 200

// State es el estado del REQ en su flujo configurable.
type State struct {
	Id   string
	Key  string
	Name string
}

// System es el sistema bajo prueba al que pertenece el REQ.
type System struct {
	Key          string
	Name         string
	DefaultRoute string
}

// ChangeRef es el CHG que agrupa al REQ.
type ChangeRef struct {
	Id        string
	Code      string
	Title     string
	Route     string
	SystemKey string
}

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// Snapshot es el contenido de un REQ. Con él se crea, se rehidrata y se lee.
type Snapshot struct {
	Id                 string
	Code               shared.Code
	Title              string
	Description        string
	AcceptanceCriteria string
	System             System
	// Change es nil mientras ningún CHG agrupa al REQ.
	Change *ChangeRef
	State  State
	// Owner es el responsable actual; nil si no tiene.
	Owner     *UserRef
	CreatedBy UserRef
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Requirement es un REQ.
type Requirement struct {
	s Snapshot
}

// NewParams son los datos para registrar un REQ.
type NewParams struct {
	Id                 string
	Code               shared.Code
	Title              string
	Description        string
	AcceptanceCriteria string
	System             System
	Change             *ChangeRef
	State              State
	Owner              *UserRef
	CreatedBy          UserRef
	Now                time.Time
}

// New valida y registra un REQ en el estado inicial de su flujo.
func New(p NewParams) (Requirement, error) {
	var errs axis.FieldErrors
	title := validTitle(p.Title, &errs)
	validChange(p.Change, p.System, &errs)
	if err := errs.Err(); err != nil {
		return Requirement{}, err
	}
	return Requirement{s: Snapshot{
		Id: p.Id, Code: p.Code, Title: title, Description: strings.TrimSpace(p.Description),
		AcceptanceCriteria: strings.TrimSpace(p.AcceptanceCriteria), System: p.System, Change: p.Change,
		State: p.State, Owner: p.Owner, CreatedBy: p.CreatedBy, CreatedAt: p.Now, UpdatedAt: p.Now,
	}}, nil
}

// Rehydrate arma un REQ ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Requirement { return Requirement{s: s} }

// Snapshot devuelve el contenido del REQ.
func (r Requirement) Snapshot() Snapshot { return r.s }

// Route es la ruta que aplica al REQ: la de su CHG o, si no tiene, la de su sistema.
func (r Requirement) Route() string {
	if r.s.Change != nil {
		return r.s.Change.Route
	}
	return r.s.System.DefaultRoute
}

// ChangeEdit cambia el CHG del REQ: Set dice si se toca; Value nil lo quita.
type ChangeEdit struct {
	Set   bool
	Value *ChangeRef
}

// OwnerEdit cambia el responsable: Set dice si se toca; Value nil lo quita.
type OwnerEdit struct {
	Set   bool
	Value *UserRef
}

// Edit son los campos a cambiar; un puntero nil deja el campo como está.
type Edit struct {
	Title              *string
	Description        *string
	AcceptanceCriteria *string
	Change             ChangeEdit
	Owner              OwnerEdit
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (r *Requirement) Apply(e Edit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := r.s
	if e.Title != nil {
		next.Title = validTitle(*e.Title, &errs)
	}
	if e.Description != nil {
		next.Description = strings.TrimSpace(*e.Description)
	}
	if e.AcceptanceCriteria != nil {
		next.AcceptanceCriteria = strings.TrimSpace(*e.AcceptanceCriteria)
	}
	if e.Change.Set {
		next.Change = e.Change.Value
		validChange(next.Change, next.System, &errs)
	}
	if e.Owner.Set {
		next.Owner = e.Owner.Value
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
	add("title", r.s.Title, next.Title)
	add("description", r.s.Description, next.Description)
	add("acceptanceCriteria", r.s.AcceptanceCriteria, next.AcceptanceCriteria)
	add("change", changeCode(r.s.Change), changeCode(next.Change))
	add("owner", ownerId(r.s.Owner), ownerId(next.Owner))
	if len(changes) > 0 {
		next.UpdatedAt = now
		r.s = next
	}
	return changes, nil
}

// MoveTo pasa el REQ a otro estado. Que la transición valga lo deciden el flujo y sus reglas.
func (r *Requirement) MoveTo(state State, now time.Time) FieldChange {
	change := FieldChange{Field: "state", Old: r.s.State.Key, New: state.Key}
	r.s.State = state
	r.s.UpdatedAt = now
	return change
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

func validChange(change *ChangeRef, system System, errs *axis.FieldErrors) {
	if change != nil && change.SystemKey != system.Key {
		errs.Add(axis.InvalidField("change", "El CHG es de otro sistema."))
	}
}

func changeCode(c *ChangeRef) string {
	if c == nil {
		return ""
	}
	return c.Code
}

func ownerId(o *UserRef) string {
	if o == nil {
		return ""
	}
	return string(o.Id)
}
