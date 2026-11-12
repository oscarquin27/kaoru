// Package environment implementa el puerto Environments con los casos de uso de environment.
package environment

import (
	"context"

	environmentuc "kaoru/internal/feature/environment/application/usecase"
	environmentdomain "kaoru/internal/feature/environment/domain"
	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/domain"
)

// Environments busca ambientes en el feature environment.
type Environments struct {
	find *environmentuc.FindEnvironment
}

var _ port.Environments = (*Environments)(nil)

// New crea el adaptador.
func New(find *environmentuc.FindEnvironment) *Environments { return &Environments{find: find} }

// Find conserva el código de error de environment.
func (e *Environments) Find(ctx context.Context, code string) (domain.EnvironmentRef, error) {
	ref, err := e.find.Execute(ctx, environmentuc.FindEnvironmentInput{Code: code})
	if err != nil {
		return domain.EnvironmentRef{}, err
	}
	return domain.EnvironmentRef{
		Id: ref.Id, Code: ref.Code, Name: ref.Name, SystemKey: ref.SystemKey,
		Production: ref.Kind == string(environmentdomain.KindProduction), Retired: ref.Retired,
	}, nil
}
