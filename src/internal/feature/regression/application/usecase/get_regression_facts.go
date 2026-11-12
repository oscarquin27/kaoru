package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/regression/domain"
)

// GetRegressionFacts dice cómo está la regresión de un PKG. Es la lectura que usa el paquete
// para sus reglas «regresión confirmada» y «regresión aprobada».
type GetRegressionFacts struct {
	d Deps
}

// GetRegressionFactsInput es la entrada de GetRegressionFacts.
type GetRegressionFactsInput struct {
	PackageCode string
}

// RegressionFacts es el estado de la regresión de un PKG.
type RegressionFacts struct {
	// Exists es false si la regresión todavía no se propuso.
	Exists    bool
	Confirmed bool
	// Unpassed son los códigos de los TC de la selección final cuyo último RUN sobre un BUILD
	// del PKG no está aprobado: los pendientes y los fallidos.
	Unpassed []string
}

// Execute devuelve ErrPackageNotFound si el PKG no existe. Leer no exige permiso.
func (uc *GetRegressionFacts) Execute(ctx context.Context, in GetRegressionFactsInput) (RegressionFacts, error) {
	pkg, err := findPackage(ctx, uc.d.Repository, in.PackageCode)
	if err != nil {
		return RegressionFacts{}, err
	}
	view, err := uc.d.Repository.FindSelection(ctx, pkg.Id)
	if axis.IsCode(err, axis.NotFoundResource) {
		return RegressionFacts{}, nil
	}
	if err != nil {
		return RegressionFacts{}, err
	}
	return RegressionFacts{Exists: true, Confirmed: view.Selection.Confirmed(), Unpassed: domain.Unpassed(progress(view))}, nil
}
