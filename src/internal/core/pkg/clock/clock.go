// Package clock inyecta el tiempo para que dominio y casos de uso no llamen a time.Now.
package clock

import "time"

// Clock da la hora actual.
type Clock interface {
	Now() time.Time
}

// System es el reloj real, en UTC y con la precisión de Postgres (microsegundos).
type System struct{}

// Now devuelve la hora actual.
func (System) Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }
