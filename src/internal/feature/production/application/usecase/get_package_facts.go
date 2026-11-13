package usecase

import (
	"context"

	"kaoru/internal/feature/production/domain"
)

// GetPackageFacts dice cómo está un PKG en producción: su último checklist, su último DIAG y su
// último SMOKE. Es la lectura que usa el paquete para sus reglas de bloqueo; no se expone por HTTP.
type GetPackageFacts struct {
	d Deps
}

// GetPackageFactsInput es la entrada de GetPackageFacts.
type GetPackageFactsInput struct {
	PackageCode string
}

// Execute devuelve los hechos vacíos si el PKG no existe o no tiene nada registrado.
func (uc *GetPackageFacts) Execute(ctx context.Context, in GetPackageFactsInput) (domain.PackageFacts, error) {
	return uc.d.Repository.PackageFacts(ctx, in.PackageCode)
}
