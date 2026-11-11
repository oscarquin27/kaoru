package usecase

import (
	"context"

	"kaoru/internal/feature/user/domain"
)

// GetMe devuelve el perfil, los roles y los permisos del usuario en sesión.
type GetMe struct {
	d Deps
}

// GetMeInput es la entrada de GetMe.
type GetMeInput struct {
	Profile ProfileInput
	Roles   []string
}

// GetMeOutput es lo que el front necesita para armar la sesión.
type GetMeOutput struct {
	Profile     domain.Profile
	Roles       []string
	Permissions []string
	LogoutURL   string
	AccountURL  string
}

// Execute arma el perfil con sus permisos.
func (uc *GetMe) Execute(ctx context.Context, in GetMeInput) (GetMeOutput, error) {
	profile, err := domain.NewProfile(in.Profile.Id, in.Profile.Username, in.Profile.DisplayName, in.Profile.Email)
	if err != nil {
		return GetMeOutput{}, err
	}
	roles, permissions, err := uc.d.Permissions.Granted(ctx, in.Roles)
	if err != nil {
		return GetMeOutput{}, err
	}
	return GetMeOutput{
		Profile: profile, Roles: roles, Permissions: permissions,
		LogoutURL: uc.d.LogoutURL, AccountURL: uc.d.AccountURL,
	}, nil
}
