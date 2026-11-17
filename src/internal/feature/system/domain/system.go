// Package domain modela el catálogo de sistemas bajo prueba (SIMF, SGLPAR, SyPago…).
package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
)

// Route es la ruta por omisión de los REQ sin CHG del sistema.
type Route string

// Rutas por omisión posibles. El fix menor es solo de un CHG: no es la ruta de un sistema.
const (
	RouteHigh  Route = "high"
	RouteLight Route = "light"
)

const maxNameLength = 80

// La clave va en filtros, enlaces y vistas guardadas: corta, en mayúsculas y sin espacios.
var keyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,11}$`)

// Snapshot es el contenido de un sistema.
type Snapshot struct {
	Id           string
	Key          string
	Name         string
	DefaultRoute Route
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// System es un sistema bajo prueba.
type System struct {
	s Snapshot
}

// NewParams son los datos para registrar un sistema.
type NewParams struct {
	Id           string
	Key          string
	Name         string
	DefaultRoute string
	Now          time.Time
}

// New valida y registra un sistema. La clave se guarda en mayúsculas.
func New(p NewParams) (System, error) {
	var errs axis.FieldErrors
	s := Snapshot{
		Id: p.Id, Key: validKey(p.Key, &errs), Name: validName(p.Name, &errs),
		DefaultRoute: validRoute(p.DefaultRoute, &errs), CreatedAt: p.Now, UpdatedAt: p.Now,
	}
	if err := errs.Err(); err != nil {
		return System{}, err
	}
	return System{s: s}, nil
}

// Rehydrate arma un sistema ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) System { return System{s: s} }

// Snapshot devuelve el contenido del sistema.
func (s System) Snapshot() Snapshot { return s.s }

// Edit son los campos a cambiar; un puntero nil deja el campo como está. La clave no cambia: la
// usan los filtros, los enlaces y las vistas guardadas.
type Edit struct {
	Name         *string
	DefaultRoute *string
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (s *System) Apply(edit Edit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := s.s
	if edit.Name != nil {
		next.Name = validName(*edit.Name, &errs)
	}
	if edit.DefaultRoute != nil {
		next.DefaultRoute = validRoute(*edit.DefaultRoute, &errs)
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
	add("name", s.s.Name, next.Name)
	add("defaultRoute", string(s.s.DefaultRoute), string(next.DefaultRoute))
	if len(changes) > 0 {
		next.UpdatedAt = now
		s.s = next
	}
	return changes, nil
}

// NormalizeKey es la clave tal como se guarda: sin espacios alrededor y en mayúsculas.
func NormalizeKey(raw string) string { return strings.ToUpper(strings.TrimSpace(raw)) }

func validKey(raw string, errs *axis.FieldErrors) string {
	key := NormalizeKey(raw)
	switch {
	case key == "":
		errs.Add(axis.Required("key", "Escribe una clave."))
	case !keyPattern.MatchString(key):
		errs.Add(axis.InvalidField("key", "La clave lleva de 2 a 12 letras o dígitos, sin espacios, y empieza con una letra."))
	}
	return key
}

func validName(raw string, errs *axis.FieldErrors) string {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		errs.Add(axis.Required("name", "Escribe un nombre."))
	case utf8.RuneCountInString(name) > maxNameLength:
		errs.Add(axis.InvalidField("name", "El nombre no puede pasar de 80 caracteres."))
	}
	return name
}

func validRoute(raw string, errs *axis.FieldErrors) Route {
	switch route := Route(raw); route {
	case RouteHigh, RouteLight:
		return route
	case "":
		errs.Add(axis.Required("defaultRoute", "Elige la ruta por omisión."))
	default:
		errs.Add(axis.InvalidField("defaultRoute", "La ruta por omisión debe ser alta o ligera."))
	}
	return ""
}
