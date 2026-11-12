package usecase

import (
	"context"
)

// FindEnvironment identifica un ambiente por su código. Es la lectura que usan otros features
// para registrar algo sobre un ENV del catálogo (un RUN, una exploración).
type FindEnvironment struct {
	d Deps
}

// FindEnvironmentInput es la entrada de FindEnvironment.
type FindEnvironmentInput struct {
	Code string
}

// EnvironmentRef es lo que identifica a un ambiente y dice si admite ejecuciones.
type EnvironmentRef struct {
	Id   string
	Code string
	Name string
	// Kind es clean, baseline, offline o production.
	Kind      string
	SystemKey string
	Retired   bool
}

// Execute devuelve ErrEnvironmentNotFound si el ambiente no existe.
func (uc *FindEnvironment) Execute(ctx context.Context, in FindEnvironmentInput) (EnvironmentRef, error) {
	view, err := findEnvironment(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return EnvironmentRef{}, err
	}
	s := view.Environment.Snapshot()
	return EnvironmentRef{
		Id: s.Id, Code: s.Code.String(), Name: s.Name, Kind: string(s.Kind), SystemKey: s.System.Key, Retired: s.RetiredAt != nil,
	}, nil
}
