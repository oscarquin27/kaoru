// Package domain modela las vistas guardadas: los filtros de una lista, con nombre.
package domain

import (
	"net/url"
	"slices"
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

const (
	maxName  = 60
	maxQuery = 2000
	// MaxPerScreen es cuántas vistas puede guardar una persona en una misma lista.
	MaxPerScreen = 30
)

// Screens son las listas que admiten vistas guardadas.
var Screens = []string{"requirements", "changes", "defects", "matrix", "test-cases", "audit"}

// Errores de dominio.
var (
	ErrNotFound = axis.New(axis.NotFoundResource, "La vista no existe.")
	ErrTooMany  = axis.New(axis.InvalidAction, "Ya tienes el máximo de vistas guardadas en esta lista. Quita una para guardar otra.")
)

// View es una vista guardada. Es personal: solo la ve y la quita su dueño.
type View struct {
	Id        string
	OwnerId   shared.UserId
	Screen    string
	Name      string
	Query     string
	CreatedAt time.Time
}

// New valida una vista nueva. Junta todos los errores de campo.
func New(id string, owner shared.UserId, screen, name, query string, now time.Time) (View, error) {
	var errs []axis.FieldError
	if !slices.Contains(Screens, screen) {
		errs = append(errs, axis.InvalidField("screen", "Esa lista no admite vistas guardadas."))
	}
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		errs = append(errs, axis.Required("name", "Ponle un nombre a la vista."))
	case len([]rune(name)) > maxName:
		errs = append(errs, axis.InvalidField("name", "El nombre admite hasta 60 caracteres."))
	}
	query = strings.TrimPrefix(strings.TrimSpace(query), "?")
	values, err := url.ParseQuery(query)
	switch {
	case err != nil || len(query) > maxQuery:
		errs = append(errs, axis.InvalidField("query", "Los filtros no son válidos."))
	case len(values) == 0:
		errs = append(errs, axis.Required("query", "Elige algún filtro antes de guardar la vista."))
	}
	if len(errs) > 0 {
		return View{}, axis.Invalid(errs...)
	}
	return View{Id: id, OwnerId: owner, Screen: screen, Name: name, Query: values.Encode(), CreatedAt: now}, nil
}
