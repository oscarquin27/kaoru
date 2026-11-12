package usecase

import (
	"context"

	"kaoru/internal/feature/environment/application/port"
	"kaoru/internal/feature/environment/domain"
)

// ListDeployments devuelve una página de despliegues, del más reciente al más antiguo.
type ListDeployments struct {
	d Deps
}

// ListDeploymentsInput es la entrada de ListDeployments. Un filtro vacío no filtra.
type ListDeploymentsInput struct {
	EnvironmentCode string
	BuildCode       string
	PackageCode     string
	Page            int
	Size            int
}

// ListDeploymentsOutput es una página de DEP.
type ListDeploymentsOutput struct {
	Deployments []domain.DeploymentSnapshot
	Total       int
}

// Execute lista los DEP. Leer no exige permiso.
func (uc *ListDeployments) Execute(ctx context.Context, in ListDeploymentsInput) (ListDeploymentsOutput, error) {
	deployments, total, err := uc.d.Repository.ListDeployments(ctx, port.DeploymentFilter{
		EnvironmentCode: in.EnvironmentCode, BuildCode: in.BuildCode, PackageCode: in.PackageCode, Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListDeploymentsOutput{}, err
	}
	out := ListDeploymentsOutput{Deployments: make([]domain.DeploymentSnapshot, len(deployments)), Total: total}
	for i, d := range deployments {
		out.Deployments[i] = d.Snapshot()
	}
	return out, nil
}
