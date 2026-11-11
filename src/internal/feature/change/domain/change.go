// Package domain modela el CHG: un cambio aprobado que agrupa requerimientos.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Route es la ruta del CHG: define cuánto proceso exige.
type Route string

// Rutas del CHG.
const (
	RouteHigh     Route = "high"
	RouteLight    Route = "light"
	RouteMinorFix Route = "minor_fix"
)

const maxTitleLength = 200

// ParseRoute valida una ruta.
func ParseRoute(raw string) (Route, bool) {
	switch r := Route(raw); r {
	case RouteHigh, RouteLight, RouteMinorFix:
		return r, true
	default:
		return "", false
	}
}

// State es el estado del CHG en su flujo configurable.
type State struct {
	Id   string
	Key  string
	Name string
}

// System es el sistema bajo prueba al que pertenece el CHG.
type System struct {
	Key          string
	Name         string
	DefaultRoute Route
}

// Snapshot es el contenido de un CHG. Con él se crea, se rehidrata y se lee.
type Snapshot struct {
	Id          string
	Code        shared.Code
	Title       string
	Description string
	System      System
	Route       Route
	State       State
	// WindowStart y WindowEnd son la ventana de cambio, que define change management.
	WindowStart *time.Time
	WindowEnd   *time.Time
	CreatedBy   shared.UserId
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Change es un CHG.
type Change struct {
	s Snapshot
}

// NewParams son los datos para crear un CHG. Route vacío toma la ruta del sistema.
type NewParams struct {
	Id          string
	Code        shared.Code
	Title       string
	Description string
	System      System
	Route       string
	WindowStart *time.Time
	WindowEnd   *time.Time
	State       State
	CreatedBy   shared.UserId
	Now         time.Time
}

// New valida y crea un CHG en el estado inicial de su flujo.
func New(p NewParams) (Change, error) {
	var errs axis.FieldErrors
	title := validTitle(p.Title, &errs)
	route := p.System.DefaultRoute
	if p.Route != "" {
		route = validRoute(p.Route, &errs)
	}
	validWindow(p.WindowStart, p.WindowEnd, &errs)
	if err := errs.Err(); err != nil {
		return Change{}, err
	}
	return Change{s: Snapshot{
		Id: p.Id, Code: p.Code, Title: title, Description: strings.TrimSpace(p.Description),
		System: p.System, Route: route, State: p.State,
		WindowStart: p.WindowStart, WindowEnd: p.WindowEnd,
		CreatedBy: p.CreatedBy, CreatedAt: p.Now, UpdatedAt: p.Now,
	}}, nil
}

// Rehydrate arma un CHG ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Change { return Change{s: s} }

// Snapshot devuelve el contenido del CHG.
func (c Change) Snapshot() Snapshot { return c.s }

// TimeEdit es el cambio de una fecha opcional: Set dice si se toca; Value nil la quita.
type TimeEdit struct {
	Set   bool
	Value *time.Time
}

// Edit son los campos a cambiar; un puntero nil deja el campo como está.
type Edit struct {
	Title       *string
	Description *string
	Route       *string
	WindowStart TimeEdit
	WindowEnd   TimeEdit
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (c *Change) Apply(e Edit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := c.s
	if e.Title != nil {
		next.Title = validTitle(*e.Title, &errs)
	}
	if e.Description != nil {
		next.Description = strings.TrimSpace(*e.Description)
	}
	if e.Route != nil {
		next.Route = validRoute(*e.Route, &errs)
	}
	if e.WindowStart.Set {
		next.WindowStart = e.WindowStart.Value
	}
	if e.WindowEnd.Set {
		next.WindowEnd = e.WindowEnd.Value
	}
	validWindow(next.WindowStart, next.WindowEnd, &errs)
	if err := errs.Err(); err != nil {
		return nil, err
	}

	var changes []FieldChange
	add := func(field, old, new string) {
		if old != new {
			changes = append(changes, FieldChange{Field: field, Old: old, New: new})
		}
	}
	add("title", c.s.Title, next.Title)
	add("description", c.s.Description, next.Description)
	add("route", string(c.s.Route), string(next.Route))
	add("windowStart", formatTime(c.s.WindowStart), formatTime(next.WindowStart))
	add("windowEnd", formatTime(c.s.WindowEnd), formatTime(next.WindowEnd))
	if len(changes) > 0 {
		next.UpdatedAt = now
		c.s = next
	}
	return changes, nil
}

// MoveTo pasa el CHG a otro estado. Que la transición valga lo decide el flujo configurado.
func (c *Change) MoveTo(state State, now time.Time) FieldChange {
	change := FieldChange{Field: "state", Old: c.s.State.Key, New: state.Key}
	c.s.State = state
	c.s.UpdatedAt = now
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

func validRoute(raw string, errs *axis.FieldErrors) Route {
	route, ok := ParseRoute(raw)
	if !ok {
		errs.Add(axis.InvalidField("route", "La ruta debe ser alta, ligera o fix menor."))
	}
	return route
}

func validWindow(start, end *time.Time, errs *axis.FieldErrors) {
	switch {
	case end == nil:
	case start == nil:
		errs.Add(axis.Required("windowStart", "Indica cuándo empieza la ventana."))
	case !end.After(*start):
		errs.Add(axis.InvalidField("windowEnd", "La ventana debe terminar después de empezar."))
	}
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
