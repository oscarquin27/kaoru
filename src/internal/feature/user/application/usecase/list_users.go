package usecase

import (
	"context"
	"strings"

	"kaoru/internal/feature/user/domain"
)

// maxListedUsers acota la lista: es para elegir un responsable, no para exportar.
const maxListedUsers = 50

// ListUsers busca usuarios para asignarlos como responsables. Solo conoce a quienes
// ya entraron a Kaoru alguna vez: el directorio vive en Keycloak.
type ListUsers struct {
	d Deps
}

// Execute devuelve los usuarios cuyo nombre o usuario contiene query.
func (uc *ListUsers) Execute(ctx context.Context, query string) ([]domain.Profile, error) {
	return uc.d.Repository.List(ctx, strings.TrimSpace(query), maxListedUsers)
}
