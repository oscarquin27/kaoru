package usecase

import (
	"context"
)

// HasProductionDeployment dice si un PKG tiene un despliegue exitoso en un ambiente de producción.
// Es la lectura que usa el paquete para su regla «desplegado en producción».
type HasProductionDeployment struct {
	d Deps
}

// HasProductionDeploymentInput es la entrada de HasProductionDeployment.
type HasProductionDeploymentInput struct {
	PackageCode string
}

// Execute devuelve false si el PKG no existe o no tiene ese despliegue.
func (uc *HasProductionDeployment) Execute(ctx context.Context, in HasProductionDeploymentInput) (bool, error) {
	return uc.d.Repository.HasProductionDeployment(ctx, in.PackageCode)
}
