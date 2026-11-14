package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/metrics/domain"
)

// GetEscapedDefects devuelve los DEF abiertos después de que su REQ llegó a producción.
type GetEscapedDefects struct {
	d Deps
}

// GetEscapedDefectsInput es la entrada de GetEscapedDefects. El período es el de creación del DEF.
type GetEscapedDefectsInput struct {
	SystemKey string
	Severity  string
	Period    PeriodInput
	Page      int
	Size      int
}

// GetEscapedDefectsOutput es la cuenta del período con una página de los escapados.
type GetEscapedDefectsOutput struct {
	Period  domain.Period
	Metrics domain.EscapedMetrics
	// Rows es la página pedida de Metrics.Escaped; Total, cuántos escaparon.
	Rows  []domain.DefectRelease
	Total int
}

// Execute cuenta los escapados del período. Leer no exige permiso.
func (uc *GetEscapedDefects) Execute(ctx context.Context, in GetEscapedDefectsInput) (GetEscapedDefectsOutput, error) {
	now := uc.d.Clock.Now()
	var errs axis.FieldErrors
	checkSeverity(in.Severity, &errs)
	period := in.Period.resolve(now, &errs)
	if err := errs.Err(); err != nil {
		return GetEscapedDefectsOutput{}, err
	}
	defects, err := uc.d.Repository.DefectReleases(ctx, domain.DefectFilter{SystemKey: in.SystemKey, Severity: in.Severity, Period: period})
	if err != nil {
		return GetEscapedDefectsOutput{}, err
	}
	m := domain.MeasureEscaped(defects, period, now)
	return GetEscapedDefectsOutput{Period: period, Metrics: m, Rows: page(m.Escaped, in.Page, in.Size), Total: len(m.Escaped)}, nil
}
