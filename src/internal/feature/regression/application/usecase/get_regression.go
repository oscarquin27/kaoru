package usecase

import (
	"context"
)

// GetRegression devuelve la selección de regresión de un PKG. Leer no exige permiso.
type GetRegression struct {
	d Deps
}

// GetRegressionInput es la entrada de GetRegression.
type GetRegressionInput struct {
	PackageCode string
}

// Execute devuelve ErrPackageNotFound si el PKG no existe y ErrNotProposed si todavía no se propuso.
func (uc *GetRegression) Execute(ctx context.Context, in GetRegressionInput) (RegressionOutput, error) {
	pkg, err := findPackage(ctx, uc.d.Repository, in.PackageCode)
	if err != nil {
		return RegressionOutput{}, err
	}
	return readOutput(ctx, uc.d.Repository, pkg)
}
