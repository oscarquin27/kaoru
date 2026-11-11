package usecase

import (
	"context"

	"kaoru/internal/feature/traceability/domain"
)

// GetMatrix devuelve una página de la matriz de trazabilidad con sus indicadores.
type GetMatrix struct {
	d Deps
}

// GetMatrixInput es la entrada de GetMatrix.
type GetMatrixInput struct {
	Filter domain.MatrixFilter
	Page   int
	Size   int
}

// GetMatrixOutput es una página de la matriz.
type GetMatrixOutput struct {
	Rows       []domain.MatrixRow
	Indicators domain.Indicators
	Total      int
}

// Execute lee la página y los indicadores de todo lo filtrado. Leer no exige permiso.
func (uc *GetMatrix) Execute(ctx context.Context, in GetMatrixInput) (GetMatrixOutput, error) {
	indicators, total, err := uc.d.Repository.MatrixIndicators(ctx, in.Filter)
	if err != nil {
		return GetMatrixOutput{}, err
	}
	rows, err := uc.d.Repository.Matrix(ctx, in.Filter, in.Page, in.Size)
	if err != nil {
		return GetMatrixOutput{}, err
	}
	return GetMatrixOutput{Rows: rows, Indicators: indicators, Total: total}, nil
}
