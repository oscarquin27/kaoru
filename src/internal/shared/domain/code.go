// Package domain reúne los tipos de negocio que comparten dos o más features.
package domain

import (
	"fmt"
	"regexp"

	"kaoru/internal/core/pkg/axis"
)

// Prefijos de los IDs legibles.
const (
	PrefixChange = "CHG"
)

// Code es el ID legible e inmutable de una entidad: prefijo, año y secuencia (REQ-2026-0042).
type Code struct {
	value string
}

var codePattern = regexp.MustCompile(`^[A-Z]+-\d{4}-\d{4,}$`)

// ErrInvalidCode es el error de un ID legible mal formado.
var ErrInvalidCode = axis.New(axis.InvalidRequest, "El ID no tiene el formato PREFIJO-AÑO-NÚMERO.")

// ParseCode valida un ID legible.
func ParseCode(raw string) (Code, error) {
	if !codePattern.MatchString(raw) {
		return Code{}, ErrInvalidCode
	}
	return Code{value: raw}, nil
}

// NewCode arma el ID legible a partir de su prefijo, año y secuencia.
func NewCode(prefix string, year, sequence int) Code {
	return Code{value: fmt.Sprintf("%s-%04d-%04d", prefix, year, sequence)}
}

func (c Code) String() string { return c.value }

// UserId identifica a un usuario: es el `sub` de Keycloak.
type UserId string

// Actor es quien ejecuta un caso de uso, con los roles que trae su token.
type Actor struct {
	Id    UserId
	Roles []string
}
