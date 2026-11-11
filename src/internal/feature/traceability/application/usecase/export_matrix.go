package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/traceability/domain"
)

const (
	exportBatch = 1000
	// MaxExportRows es el tope de filas de una exportación.
	MaxExportRows = 50000
)

// ErrExportTooLarge es el error de una exportación que pasa el tope de filas.
var ErrExportTooLarge = axis.New(axis.InvalidAction, "La matriz filtrada pasa de 50 000 filas. Acota los filtros para exportarla.")

// ExportMatrix devuelve todas las filas de la matriz filtrada, para exportarlas.
type ExportMatrix struct {
	d Deps
}

// ExportMatrixInput es la entrada de ExportMatrix.
type ExportMatrixInput struct {
	Filter domain.MatrixFilter
}

// Execute lee la matriz completa por lotes. Leer no exige permiso.
func (uc *ExportMatrix) Execute(ctx context.Context, in ExportMatrixInput) ([]domain.MatrixRow, error) {
	_, total, err := uc.d.Repository.MatrixIndicators(ctx, in.Filter)
	if err != nil {
		return nil, err
	}
	if total > MaxExportRows {
		return nil, ErrExportTooLarge
	}
	rows := make([]domain.MatrixRow, 0, total)
	for page := 1; len(rows) < total; page++ {
		batch, err := uc.d.Repository.Matrix(ctx, in.Filter, page, exportBatch)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		rows = append(rows, batch...)
	}
	return rows, nil
}
