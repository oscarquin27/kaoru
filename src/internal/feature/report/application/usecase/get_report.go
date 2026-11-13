package usecase

import (
	"context"

	"kaoru/internal/feature/report/domain"
)

// GetReport devuelve la ficha de un RPT con su contenido.
type GetReport struct {
	d Deps
}

// GetReportInput es la entrada de GetReport.
type GetReportInput struct {
	Code string
}

// Execute devuelve NOT_FOUND_RESOURCE si el RPT no existe. Leer no exige permiso.
func (uc *GetReport) Execute(ctx context.Context, in GetReportInput) (domain.Snapshot, error) {
	report, err := findReport(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return domain.Snapshot{}, err
	}
	return report.Snapshot(), nil
}
