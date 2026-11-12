package usecase

import (
	"context"

	"kaoru/internal/feature/traceability/domain"
)

// GetPackageTrace devuelve la trazabilidad de un PKG: REQ → RISK → UC → TC → RUN → DEF, los RUN sobre sus
// BUILD y los DEF no cerrados de sus REQ. Es la lectura que usa el informe de conformidad; no se expone por HTTP.
type GetPackageTrace struct {
	d Deps
}

// GetPackageTraceInput es la entrada de GetPackageTrace.
type GetPackageTraceInput struct {
	PackageCode string
}

// Execute devuelve listas vacías si el PKG no existe: quien pregunta ya sabe si existe.
func (uc *GetPackageTrace) Execute(ctx context.Context, in GetPackageTraceInput) (domain.PackageTrace, error) {
	return uc.d.Repository.PackageTrace(ctx, in.PackageCode)
}
