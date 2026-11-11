// Package idgen inyecta la generación de llaves.
package idgen

import (
	"fmt"

	"github.com/google/uuid"
)

// IdGenerator genera llaves técnicas.
type IdGenerator interface {
	NewId() (string, error)
}

// UUIDv7 genera UUID versión 7, ordenables por tiempo.
type UUIDv7 struct{}

// NewId devuelve un UUIDv7 en texto.
func (UUIDv7) NewId() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generar uuid: %w", err)
	}
	return id.String(), nil
}
