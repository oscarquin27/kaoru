// Package domain modela el perfil del usuario, leído del token.
package domain

import (
	"strings"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// ErrInvalidProfile es el error de un token sin identificador de usuario.
var ErrInvalidProfile = axis.New(axis.InvalidToken, "La sesión no trae la identidad del usuario.")

// Profile es lo que Kaoru guarda de un usuario. Las contraseñas y los roles viven en Keycloak.
type Profile struct {
	Id          shared.UserId
	Username    string
	DisplayName string
	Email       string
}

// NewProfile valida el perfil. Si el token no trae nombre, se muestra el usuario.
func NewProfile(id, username, displayName, email string) (Profile, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	if id == "" || username == "" {
		return Profile{}, ErrInvalidProfile
	}
	if displayName == "" {
		displayName = username
	}
	return Profile{Id: shared.UserId(id), Username: username, DisplayName: displayName, Email: strings.TrimSpace(email)}, nil
}
