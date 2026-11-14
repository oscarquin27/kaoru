package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/metrics/domain"
)

// GetDefectStates devuelve cuánto tiempo pasan los DEF en cada estado, con los cerrados incluidos.
type GetDefectStates struct {
	d Deps
}

// GetDefectStatesInput es la entrada de GetDefectStates. El período es el de creación del DEF.
type GetDefectStatesInput struct {
	SystemKey string
	Severity  string
	Period    PeriodInput
}

// GetDefectStatesOutput es el tiempo en estado del período.
type GetDefectStatesOutput struct {
	Period  domain.Period
	Metrics domain.DefectStateMetrics
}

// Execute lee la historia de los DEF y la mide hasta ahora. Leer no exige permiso.
func (uc *GetDefectStates) Execute(ctx context.Context, in GetDefectStatesInput) (GetDefectStatesOutput, error) {
	now := uc.d.Clock.Now()
	var errs axis.FieldErrors
	checkSeverity(in.Severity, &errs)
	period := in.Period.resolve(now, &errs)
	if err := errs.Err(); err != nil {
		return GetDefectStatesOutput{}, err
	}
	states, err := uc.d.Repository.DefectStates(ctx)
	if err != nil {
		return GetDefectStatesOutput{}, err
	}
	histories, err := uc.d.Repository.DefectHistories(ctx, domain.DefectFilter{SystemKey: in.SystemKey, Severity: in.Severity, Period: period})
	if err != nil {
		return GetDefectStatesOutput{}, err
	}
	return GetDefectStatesOutput{Period: period, Metrics: domain.MeasureDefectStates(states, histories, now)}, nil
}
