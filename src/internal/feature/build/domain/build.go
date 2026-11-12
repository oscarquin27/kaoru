// Package domain modela el BUILD: un artefacto inmutable con su manifiesto.
package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

const maxVersionLength = 80

var checksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// System es el sistema bajo prueba al que pertenece el BUILD.
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

// RequirementRef es un REQ incluido en el manifiesto.
type RequirementRef struct {
	Id        string
	Code      string
	Title     string
	SystemKey string
}

// Snapshot es el contenido de un BUILD.
type Snapshot struct {
	Id      string
	Code    shared.Code
	System  System
	Version string
	// Checksum es el SHA-256 del artefacto, en minúsculas.
	Checksum string
	// El manifiesto: los REQ que incluye y, como texto de una línea por elemento, migraciones y dependencias.
	Requirements []RequirementRef
	Migrations   string
	Dependencies string
	Notes        string
	BuiltBy      UserRef
	CreatedAt    time.Time
}

// Build es un BUILD-. No tiene métodos para cambiar su contenido: es inmutable por regla de negocio.
type Build struct {
	s Snapshot
}

// NewParams son los datos para registrar un BUILD.
type NewParams struct {
	Id           string
	Code         shared.Code
	System       System
	Version      string
	Checksum     string
	Requirements []RequirementRef
	Migrations   string
	Dependencies string
	Notes        string
	BuiltBy      shared.UserId
	Now          time.Time
}

// New valida y registra un BUILD. Los REQ del manifiesto deben ser de su mismo sistema.
func New(p NewParams) (Build, error) {
	var errs axis.FieldErrors
	s := Snapshot{
		Id: p.Id, Code: p.Code, System: p.System,
		Version:      strings.TrimSpace(p.Version),
		Checksum:     strings.ToLower(strings.TrimSpace(p.Checksum)),
		Requirements: p.Requirements, Migrations: strings.TrimSpace(p.Migrations),
		Dependencies: strings.TrimSpace(p.Dependencies), Notes: strings.TrimSpace(p.Notes),
		BuiltBy: UserRef{Id: p.BuiltBy}, CreatedAt: p.Now,
	}
	switch {
	case s.Version == "":
		errs.Add(axis.Required("version", "Indica la versión del BUILD."))
	case utf8.RuneCountInString(s.Version) > maxVersionLength:
		errs.Add(axis.InvalidField("version", "La versión no puede pasar de 80 caracteres."))
	}
	switch {
	case s.Checksum == "":
		errs.Add(axis.Required("checksum", "Pega el checksum SHA-256 del artefacto."))
	case !checksumPattern.MatchString(s.Checksum):
		errs.Add(axis.InvalidField("checksum", "El checksum debe ser un SHA-256: 64 caracteres hexadecimales."))
	}
	for _, r := range s.Requirements {
		if r.SystemKey != p.System.Key {
			errs.Add(axis.InvalidField("requirements", r.Code+" es de otro sistema: un BUILD solo incluye REQ de su sistema."))
			break
		}
	}
	if err := errs.Err(); err != nil {
		return Build{}, err
	}
	return Build{s: s}, nil
}

// Rehydrate arma un BUILD ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Build { return Build{s: s} }

// Snapshot devuelve el contenido del BUILD.
func (b Build) Snapshot() Snapshot { return b.s }
