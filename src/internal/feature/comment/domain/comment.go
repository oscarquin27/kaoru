// Package domain modela los comentarios de la pestaña Actividad.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

const maxBodyLength = 5000

// EntityRef es la entidad comentada.
type EntityRef struct {
	Type string
	Id   string
	Code string
}

// Comment es un comentario. No se edita ni se borra: forma parte de la historia.
type Comment struct {
	Id         string
	Entity     EntityRef
	Body       string
	AuthorId   shared.UserId
	AuthorName string
	CreatedAt  time.Time
}

// New valida y crea un comentario.
func New(id string, entity EntityRef, body string, author shared.UserId, authorName string, now time.Time) (Comment, error) {
	body = strings.TrimSpace(body)
	switch {
	case body == "":
		return Comment{}, axis.Invalid(axis.Required("body", "Escribe el comentario."))
	case utf8.RuneCountInString(body) > maxBodyLength:
		return Comment{}, axis.Invalid(axis.InvalidField("body", "El comentario no puede pasar de 5000 caracteres."))
	}
	return Comment{Id: id, Entity: entity, Body: body, AuthorId: author, AuthorName: authorName, CreatedAt: now}, nil
}
