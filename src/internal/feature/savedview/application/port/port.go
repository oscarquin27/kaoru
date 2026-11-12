// Package port declara lo que necesitan los casos de uso de savedview.
package port

import (
	"context"

	"kaoru/internal/feature/savedview/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda las vistas de cada usuario.
type Repository interface {
	// Insert devuelve DUPLICATE_RESOURCE si el dueño ya tiene una vista con ese nombre en esa lista.
	Insert(ctx context.Context, view domain.View) error
	// List devuelve las vistas del dueño en una lista, por nombre.
	List(ctx context.Context, owner shared.UserId, screen string) ([]domain.View, error)
	Count(ctx context.Context, owner shared.UserId, screen string) (int, error)
	// Delete devuelve NOT_FOUND_RESOURCE si la vista no existe o es de otra persona.
	Delete(ctx context.Context, owner shared.UserId, id string) error
}
