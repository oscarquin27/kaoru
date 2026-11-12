// Package domain modela el COMP: un componente del catálogo de un sistema.
package domain

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Kind es la clase de componente.
type Kind string

// Clases de componente.
const (
	KindMicroservice Kind = "microservice"
	KindLibrary      Kind = "library"
	KindScript       Kind = "script"
	KindKafkaTopic   Kind = "kafka_topic"
	KindDBSchema     Kind = "db_schema"
	KindOther        Kind = "other"
)

const maxNameLength = 120

// ErrNameTaken es el error de un nombre que ya usa otro componente del mismo sistema.
var ErrNameTaken = axis.Invalid(axis.InvalidField("name", "Ya hay un componente con ese nombre en el sistema."))

// ParseKind valida una clase de componente.
func ParseKind(raw string) (Kind, bool) {
	switch k := Kind(raw); k {
	case KindMicroservice, KindLibrary, KindScript, KindKafkaTopic, KindDBSchema, KindOther:
		return k, true
	default:
		return "", false
	}
}

// System es el sistema bajo prueba al que pertenece el COMP.
type System struct {
	Key          string
	Name         string
	DefaultRoute string
}

// Snapshot es el contenido de un COMP. Con él se crea, se rehidrata y se lee.
type Snapshot struct {
	Id          string
	Code        shared.Code
	System      System
	Kind        Kind
	Name        string
	Description string
	// RetiredAt no es nil si el COMP está retirado: conserva su historia, pero ya no se ofrece para declarar impacto.
	RetiredAt *time.Time
	CreatedBy shared.UserId
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Component es un COMP.
type Component struct {
	s Snapshot
}

// NewParams son los datos para agregar un COMP al catálogo.
type NewParams struct {
	Id          string
	Code        shared.Code
	System      System
	Kind        string
	Name        string
	Description string
	CreatedBy   shared.UserId
	Now         time.Time
}

// New valida y crea un COMP vigente.
func New(p NewParams) (Component, error) {
	var errs axis.FieldErrors
	kind := validKind(p.Kind, &errs)
	name := validName(p.Name, &errs)
	if err := errs.Err(); err != nil {
		return Component{}, err
	}
	return Component{s: Snapshot{
		Id: p.Id, Code: p.Code, System: p.System, Kind: kind, Name: name,
		Description: strings.TrimSpace(p.Description), CreatedBy: p.CreatedBy, CreatedAt: p.Now, UpdatedAt: p.Now,
	}}, nil
}

// Rehydrate arma un COMP ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Component { return Component{s: s} }

// Snapshot devuelve el contenido del COMP.
func (c Component) Snapshot() Snapshot { return c.s }

// Retired dice si el COMP está retirado.
func (c Component) Retired() bool { return c.s.RetiredAt != nil }

// Edit son los campos a cambiar; un puntero nil deja el campo como está. El sistema no cambia.
type Edit struct {
	Kind        *string
	Name        *string
	Description *string
	// Retired retira el COMP (true) o lo vuelve a dejar vigente (false).
	Retired *bool
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (c *Component) Apply(e Edit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := c.s
	if e.Kind != nil {
		next.Kind = validKind(*e.Kind, &errs)
	}
	if e.Name != nil {
		next.Name = validName(*e.Name, &errs)
	}
	if e.Description != nil {
		next.Description = strings.TrimSpace(*e.Description)
	}
	if e.Retired != nil && *e.Retired != c.Retired() {
		next.RetiredAt = nil
		if *e.Retired {
			next.RetiredAt = &now
		}
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
	add("kind", string(c.s.Kind), string(next.Kind))
	add("name", c.s.Name, next.Name)
	add("description", c.s.Description, next.Description)
	add("retired", strconv.FormatBool(c.Retired()), strconv.FormatBool(next.RetiredAt != nil))
	if len(changes) > 0 {
		next.UpdatedAt = now
		c.s = next
	}
	return changes, nil
}

func validKind(raw string, errs *axis.FieldErrors) Kind {
	kind, ok := ParseKind(raw)
	switch {
	case raw == "":
		errs.Add(axis.Required("kind", "Elige la clase de componente."))
	case !ok:
		errs.Add(axis.InvalidField("kind", "La clase de componente no es válida."))
	}
	return kind
}

func validName(raw string, errs *axis.FieldErrors) string {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		errs.Add(axis.Required("name", "Escribe un nombre."))
	case utf8.RuneCountInString(name) > maxNameLength:
		errs.Add(axis.InvalidField("name", "El nombre no puede pasar de 120 caracteres."))
	}
	return name
}
