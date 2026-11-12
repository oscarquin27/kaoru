package usecase

import (
	"context"
)

// GetPackage devuelve la ficha de un PKG: su contenido y qué le falta para avanzar. Leer no exige permiso.
type GetPackage struct {
	d Deps
}

// GetPackageInput es la entrada de GetPackage.
type GetPackageInput struct {
	Code string
}

// Execute devuelve ErrPackageNotFound si el PKG no existe.
func (uc *GetPackage) Execute(ctx context.Context, in GetPackageInput) (PackageOutput, error) {
	return readPackage(ctx, uc.d, in.Code)
}
