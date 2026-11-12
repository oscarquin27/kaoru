// Package domain modela los ambientes (ENV) y los despliegues (DEP) que se registran sobre ellos.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Kind es el tipo de un ambiente.
type Kind string

// Tipos de ambiente.
const (
	KindClean      Kind = "clean"
	KindBaseline   Kind = "baseline"
	KindOffline    Kind = "offline"
	KindProduction Kind = "production"
)

// OS es el sistema operativo de un ambiente.
type OS string

// Sistemas operativos.
const (
	OSWindows OS = "windows"
	OSLinux   OS = "linux"
)

const maxNameLength = 80

// System es el sistema bajo prueba al que pertenece el ambiente.
type System struct {
	Key          string
	Name         string
	DefaultRoute string
}

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// Snapshot es el contenido de un ENV.
type Snapshot struct {
	Id     string
	Code   shared.Code
	System System
	Name   string
	Kind   Kind
	OS     OS
	Notes  string
	// RetiredAt es nil mientras el ambiente está en uso. Retirado, no admite despliegues ni ejecuciones.
	RetiredAt *time.Time
	CreatedBy UserRef
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Environment es un ENV-: un ambiente de un sistema.
type Environment struct {
	s Snapshot
}

// NewParams son los datos para registrar un ENV.
type NewParams struct {
	Id        string
	Code      shared.Code
	System    System
	Name      string
	Kind      string
	OS        string
	Notes     string
	CreatedBy shared.UserId
	Now       time.Time
}

// New valida y registra un ENV.
func New(p NewParams) (Environment, error) {
	var errs axis.FieldErrors
	s := Snapshot{
		Id: p.Id, Code: p.Code, System: p.System,
		Name: validName(p.Name, &errs), Kind: validKind(p.Kind, &errs), OS: validOS(p.OS, &errs),
		Notes: strings.TrimSpace(p.Notes), CreatedBy: UserRef{Id: p.CreatedBy}, CreatedAt: p.Now, UpdatedAt: p.Now,
	}
	if err := errs.Err(); err != nil {
		return Environment{}, err
	}
	return Environment{s: s}, nil
}

// Rehydrate arma un ENV ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Environment { return Environment{s: s} }

// Snapshot devuelve el contenido del ENV.
func (e Environment) Snapshot() Snapshot { return e.s }

// Edit son los campos a cambiar; un puntero nil deja el campo como está. El sistema no cambia.
type Edit struct {
	Name    *string
	Kind    *string
	OS      *string
	Notes   *string
	Retired *bool
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (e *Environment) Apply(edit Edit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := e.s
	if edit.Name != nil {
		next.Name = validName(*edit.Name, &errs)
	}
	if edit.Kind != nil {
		next.Kind = validKind(*edit.Kind, &errs)
		// Producción exige otro permiso para desplegar: un ambiente no entra ni sale de ese tipo editándolo.
		if (next.Kind == KindProduction) != (e.s.Kind == KindProduction) {
			errs.Add(axis.InvalidField("kind", "Un ambiente no pasa a producción ni deja de serlo. Retíralo y registra otro."))
		}
	}
	if edit.OS != nil {
		next.OS = validOS(*edit.OS, &errs)
	}
	if edit.Notes != nil {
		next.Notes = strings.TrimSpace(*edit.Notes)
	}
	if edit.Retired != nil && *edit.Retired != (e.s.RetiredAt != nil) {
		next.RetiredAt = nil
		if *edit.Retired {
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
	add("name", e.s.Name, next.Name)
	add("kind", string(e.s.Kind), string(next.Kind))
	add("os", string(e.s.OS), string(next.OS))
	add("notes", e.s.Notes, next.Notes)
	add("retired", formatBool(e.s.RetiredAt != nil), formatBool(next.RetiredAt != nil))
	if len(changes) > 0 {
		next.UpdatedAt = now
		e.s = next
	}
	return changes, nil
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

func validKind(raw string, errs *axis.FieldErrors) Kind {
	switch kind := Kind(raw); kind {
	case KindClean, KindBaseline, KindOffline, KindProduction:
		return kind
	case "":
		errs.Add(axis.Required("kind", "Elige el tipo de ambiente."))
	default:
		errs.Add(axis.InvalidField("kind", "El tipo debe ser limpio, línea base, offline o producción."))
	}
	return ""
}

func validOS(raw string, errs *axis.FieldErrors) OS {
	switch os := OS(raw); os {
	case OSWindows, OSLinux:
		return os
	case "":
		errs.Add(axis.Required("os", "Elige el sistema operativo."))
	default:
		errs.Add(axis.InvalidField("os", "El sistema operativo debe ser Windows o Linux."))
	}
	return ""
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
