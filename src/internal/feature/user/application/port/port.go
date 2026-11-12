// Package port declara lo que necesitan los casos de uso de user.
package port

import (
	"context"
	"time"

	"kaoru/internal/feature/user/domain"
)

// Repository guarda el perfil de los usuarios.
type Repository interface {
	// Upsert crea el perfil o lo actualiza si cambió.
	Upsert(ctx context.Context, profile domain.Profile, now time.Time) error
	// List devuelve hasta limit usuarios cuyo nombre o usuario contiene query, por nombre.
	List(ctx context.Context, query string, limit int) ([]domain.Profile, error)
}

// Permissions lee lo que pueden hacer los roles de un actor.
type Permissions interface {
	// Granted devuelve los roles que Kaoru reconoce y los permisos que suman.
	Granted(ctx context.Context, roles []string) (known []string, permissions []string, err error)
}
