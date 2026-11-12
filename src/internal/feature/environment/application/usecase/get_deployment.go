package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// GetDeployment devuelve un despliegue por su ID legible. Leer no exige permiso.
type GetDeployment struct {
	d Deps
}

// GetDeploymentInput es la entrada de GetDeployment.
type GetDeploymentInput struct {
	Code string
}

// Execute busca el DEP; un código mal formado tampoco existe.
func (uc *GetDeployment) Execute(ctx context.Context, in GetDeploymentInput) (domain.DeploymentSnapshot, error) {
	code, err := shared.ParseCode(in.Code)
	if err != nil {
		return domain.DeploymentSnapshot{}, ErrDeploymentNotFound
	}
	deployment, err := uc.d.Repository.FindDeployment(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.DeploymentSnapshot{}, ErrDeploymentNotFound
	}
	return deployment.Snapshot(), err
}
