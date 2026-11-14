package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/metrics/domain"
)

// GetRegression devuelve el tamaño de la regresión de cada PKG para compararlos.
type GetRegression struct {
	d Deps
}

// GetRegressionInput es la entrada de GetRegression. El período es el de la confirmación.
type GetRegressionInput struct {
	SystemKey string
	Period    PeriodInput
	Page      int
	Size      int
}

// GetRegressionOutput es una página de PKG con el resumen de todo el período.
type GetRegressionOutput struct {
	Period  domain.Period
	Summary domain.RegressionSummary
	Rows    []domain.RegressionRow
	Total   int
}

// Execute lee las regresiones confirmadas. Leer no exige permiso.
func (uc *GetRegression) Execute(ctx context.Context, in GetRegressionInput) (GetRegressionOutput, error) {
	var errs axis.FieldErrors
	period := in.Period.resolve(uc.d.Clock.Now(), &errs)
	if err := errs.Err(); err != nil {
		return GetRegressionOutput{}, err
	}
	rows, err := uc.d.Repository.RegressionSizes(ctx, domain.RegressionFilter{SystemKey: in.SystemKey, Period: period})
	if err != nil {
		return GetRegressionOutput{}, err
	}
	return GetRegressionOutput{
		Period: period, Summary: domain.SummarizeRegression(rows), Rows: page(rows, in.Page, in.Size), Total: len(rows),
	}, nil
}
