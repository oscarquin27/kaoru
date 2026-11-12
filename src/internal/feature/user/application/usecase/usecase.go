// Package usecase contiene los casos de uso de user.
package usecase

import (
	"errors"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/feature/user/application/port"
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Permissions port.Permissions
	Clock       clock.Clock
	// LogoutURL y AccountURL son enlaces de la sesión que el front recibe del API.
	LogoutURL  string
	AccountURL string
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	SyncProfile *SyncProfile
	GetMe       *GetMe
	ListUsers   *ListUsers
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Permissions == nil || d.Clock == nil || d.LogoutURL == "" {
		return nil, errors.New("user: faltan dependencias")
	}
	return &UseCases{SyncProfile: &SyncProfile{d: d}, GetMe: &GetMe{d: d}, ListUsers: &ListUsers{d: d}}, nil
}

// ProfileInput son los datos de identidad que el handler saca del token.
type ProfileInput struct {
	Id          string
	Username    string
	DisplayName string
	Email       string
}
