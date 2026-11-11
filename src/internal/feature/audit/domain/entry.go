// Package domain modela el registro de auditoría encadenado por hash.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Action es la clase de hecho que se audita.
type Action string

// Acciones auditables.
const (
	ActionCreate     Action = "create"
	ActionUpdate     Action = "update"
	ActionTransition Action = "transition"
	ActionDecision   Action = "decision"
	ActionDelete     Action = "delete"
)

// ErrInvalidEntry es el error de un registro al que le falta un dato obligatorio.
var ErrInvalidEntry = axis.New(axis.InternalServerError, "registro de auditoría incompleto")

// Entry es un registro de auditoría. No cambia después de creado.
type Entry struct {
	Id         string
	Seq        int64
	OccurredAt time.Time
	ActorId    shared.UserId
	EntityType string
	EntityId   string
	EntityCode string
	Action     Action
	Field      string
	OldValue   string
	NewValue   string
	Reason     string
	PrevHash   string
	Hash       string
}

// NewEntry valida un registro nuevo. El hash se calcula al encadenarlo.
func NewEntry(id string, occurredAt time.Time, actor shared.UserId, entityType, entityId, entityCode string,
	action Action, field, oldValue, newValue, reason string) (Entry, error) {
	switch action {
	case ActionCreate, ActionUpdate, ActionTransition, ActionDecision, ActionDelete:
	default:
		return Entry{}, ErrInvalidEntry
	}
	if id == "" || actor == "" || entityType == "" || entityId == "" || occurredAt.IsZero() {
		return Entry{}, ErrInvalidEntry
	}
	return Entry{
		Id: id,
		// Postgres guarda microsegundos: el hash debe calcularse con lo mismo que se va a leer.
		OccurredAt: occurredAt.UTC().Truncate(time.Microsecond),
		ActorId:    actor,
		EntityType: entityType,
		EntityId:   entityId,
		EntityCode: entityCode,
		Action:     action,
		Field:      field,
		OldValue:   oldValue,
		NewValue:   newValue,
		Reason:     reason,
	}, nil
}

// Chained devuelve el registro enlazado al hash del anterior.
func (e Entry) Chained(prevHash string) Entry {
	e.PrevHash = prevHash
	e.Hash = e.computeHash(prevHash)
	return e
}

// Intact dice si el registro conserva el contenido con el que se encadenó
// y si sigue al registro cuyo hash se pasa.
func (e Entry) Intact(prevHash string) bool {
	return e.PrevHash == prevHash && e.Hash == e.computeHash(prevHash)
}

// computeHash es el SHA-256 del hash anterior y del contenido. Cada campo va precedido
// de su longitud para que dos contenidos distintos nunca den la misma cadena.
func (e Entry) computeHash(prevHash string) string {
	parts := []string{
		prevHash, e.Id, e.OccurredAt.UTC().Format(time.RFC3339Nano), string(e.ActorId),
		e.EntityType, e.EntityId, e.EntityCode, string(e.Action),
		e.Field, e.OldValue, e.NewValue, e.Reason,
	}
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(strconv.Itoa(len(p)))
		b.WriteByte(':')
		b.WriteString(p)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
