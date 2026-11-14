package usecase_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/metrics/application/usecase"
	"kaoru/internal/feature/metrics/domain"
)

var now = time.Date(2026, 11, 14, 15, 0, 0, 0, time.UTC)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// repo es el doble del repositorio: devuelve lo que tiene y guarda los filtros que recibió.
type repo struct {
	coverage   []domain.CoverageRow
	states     []domain.WorkflowState
	histories  []domain.DefectHistory
	regression []domain.RegressionRow
	releases   []domain.DefectRelease
	err        error

	coverageFilter   domain.CoverageFilter
	defectFilter     domain.DefectFilter
	regressionFilter domain.RegressionFilter
}

func (r *repo) Coverage(_ context.Context, f domain.CoverageFilter) ([]domain.CoverageRow, error) {
	r.coverageFilter = f
	return slices.Clone(r.coverage), r.err
}

func (r *repo) DefectStates(context.Context) ([]domain.WorkflowState, error) { return r.states, r.err }

func (r *repo) DefectHistories(_ context.Context, f domain.DefectFilter) ([]domain.DefectHistory, error) {
	r.defectFilter = f
	return r.histories, r.err
}

func (r *repo) RegressionSizes(_ context.Context, f domain.RegressionFilter) ([]domain.RegressionRow, error) {
	r.regressionFilter = f
	return r.regression, r.err
}

func (r *repo) DefectReleases(_ context.Context, f domain.DefectFilter) ([]domain.DefectRelease, error) {
	r.defectFilter = f
	return r.releases, r.err
}

func setup(t *testing.T, r *repo) *usecase.UseCases {
	t.Helper()
	uc, err := usecase.New(usecase.Deps{Repository: r, Clock: fixedClock{now}})
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

func date(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

func fieldsOf(err error) []string {
	var general *axis.GeneralError
	if !errors.As(err, &general) {
		return nil
	}
	var out []string
	for _, f := range general.Fields() {
		out = append(out, f.Field)
	}
	return out
}

func TestNewNeedsEveryDependency(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{Clock: fixedClock{now}}); err == nil {
		t.Error("sin repositorio")
	}
	if _, err := usecase.New(usecase.Deps{Repository: &repo{}}); err == nil {
		t.Error("sin reloj")
	}
}

func TestGetCoverageFiltersSortsAndPages(t *testing.T) {
	minimum := 100.0
	r := &repo{coverage: []domain.CoverageRow{
		{Requirement: domain.CodeTitle{Code: "REQ-1"}, RiskLevel: "high", Required: 2, Passed: 2, MinimumPercent: &minimum},
		{Requirement: domain.CodeTitle{Code: "REQ-2"}, RiskLevel: "high", Required: 2, Passed: 1, MinimumPercent: &minimum},
		{Requirement: domain.CodeTitle{Code: "REQ-3"}},
		{Requirement: domain.CodeTitle{Code: "REQ-4"}, RiskLevel: "high", Required: 4, Passed: 0, MinimumPercent: &minimum},
	}}
	uc := setup(t, r)
	ctx := context.Background()
	filter := domain.CoverageFilter{SystemKey: "SIMF", ChangeCode: "CHG-2026-0001", PackageCode: "PKG-2026-0001", StateKey: "in_testing"}

	out, err := uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{Filter: filter, Page: 1, Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	if r.coverageFilter != filter {
		t.Errorf("filtro al repositorio: %+v", r.coverageFilter)
	}
	codes := func(rows []domain.CoverageRow) []string {
		var out []string
		for _, row := range rows {
			out = append(out, row.Requirement.Code)
		}
		return out
	}
	if out.Total != 4 || !slices.Equal(codes(out.Rows), []string{"REQ-4", "REQ-2"}) || out.Summary.Requirements != 4 {
		t.Errorf("página 1: %+v", out)
	}
	out, _ = uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{Page: 3, Size: 2})
	if out.Total != 4 || len(out.Rows) != 0 {
		t.Errorf("después de la última página: %+v", out)
	}

	// El estado acota la lista, no el resumen.
	out, _ = uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{Status: "below", Page: 1, Size: 25})
	if out.Total != 2 || out.Summary.Requirements != 4 || out.Summary.ByStatus[domain.CoverageMeets] != 1 {
		t.Errorf("por debajo del mínimo: %+v", out)
	}

	if _, err := uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{Status: "bajo", Page: 1, Size: 25}); !slices.Equal(fieldsOf(err), []string{"status"}) {
		t.Errorf("estado desconocido: %v", err)
	}
	r.err = errors.New("base caída")
	if _, err := uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{Page: 1, Size: 25}); !errors.Is(err, r.err) {
		t.Errorf("error del repositorio: %v", err)
	}
}

func TestGetDefectStatesPeriodAndMeasure(t *testing.T) {
	r := &repo{
		states: []domain.WorkflowState{{State: domain.StateRef{Key: "open", Name: "Abierto"}}},
		histories: []domain.DefectHistory{
			{Id: "a", CreatedAt: now.Add(-48 * time.Hour), Current: "open"},
		},
	}
	uc := setup(t, r)
	ctx := context.Background()

	// Sin período: los últimos 12 meses, sin fin. La estadía de lo abierto cuenta hasta el reloj.
	out, err := uc.GetDefectStates.Execute(ctx, usecase.GetDefectStatesInput{SystemKey: "SIMF", Severity: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if want := (domain.DefectFilter{SystemKey: "SIMF", Severity: "high", Period: domain.Period{From: *date(2025, 12, 1)}}); r.defectFilter.SystemKey != want.SystemKey ||
		r.defectFilter.Severity != want.Severity || !r.defectFilter.Period.From.Equal(want.Period.From) || r.defectFilter.Period.To != nil {
		t.Errorf("filtro al repositorio: %+v", r.defectFilter)
	}
	if !out.Period.From.Equal(*date(2025, 12, 1)) || out.Metrics.Total != 1 || *out.Metrics.States[0].MedianDays != 2 {
		t.Errorf("salida: %+v", out)
	}

	// Un período pedido se respeta.
	out, _ = uc.GetDefectStates.Execute(ctx, usecase.GetDefectStatesInput{Period: usecase.PeriodInput{From: date(2026, 1, 1), To: date(2026, 7, 1)}})
	if !out.Period.From.Equal(*date(2026, 1, 1)) || !out.Period.To.Equal(*date(2026, 7, 1)) {
		t.Errorf("período pedido: %+v", out.Period)
	}

	_, err = uc.GetDefectStates.Execute(ctx, usecase.GetDefectStatesInput{
		Severity: "crítica", Period: usecase.PeriodInput{From: date(2026, 7, 1), To: date(2026, 7, 1)},
	})
	if !slices.Equal(fieldsOf(err), []string{"severity", "to"}) {
		t.Errorf("severidad y período inválidos: %v", err)
	}
	r.err = errors.New("base caída")
	if _, err := uc.GetDefectStates.Execute(ctx, usecase.GetDefectStatesInput{}); !errors.Is(err, r.err) {
		t.Errorf("error del repositorio: %v", err)
	}
}

func TestGetRegressionSummarizesAndPages(t *testing.T) {
	r := &repo{regression: []domain.RegressionRow{
		{Package: domain.CodeTitle{Code: "PKG-3"}, Proposed: 5, Final: 6, Added: 1},
		{Package: domain.CodeTitle{Code: "PKG-2"}, Proposed: 3, Final: 2, Removed: 1},
		{Package: domain.CodeTitle{Code: "PKG-1"}, Proposed: 4, Final: 4},
	}}
	uc := setup(t, r)
	out, err := uc.GetRegression.Execute(context.Background(), usecase.GetRegressionInput{SystemKey: "SYPAGO", Page: 2, Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	if r.regressionFilter.SystemKey != "SYPAGO" || !r.regressionFilter.Period.From.Equal(*date(2025, 12, 1)) {
		t.Errorf("filtro al repositorio: %+v", r.regressionFilter)
	}
	if out.Total != 3 || len(out.Rows) != 1 || out.Rows[0].Package.Code != "PKG-1" || out.Summary.Final != 12 || *out.Summary.MedianFinal != 4 {
		t.Errorf("salida: %+v", out)
	}
	_, err = uc.GetRegression.Execute(context.Background(), usecase.GetRegressionInput{
		Period: usecase.PeriodInput{From: date(2026, 8, 1), To: date(2026, 7, 1)}, Page: 1, Size: 25,
	})
	if !slices.Equal(fieldsOf(err), []string{"to"}) {
		t.Errorf("período al revés: %v", err)
	}
}

func TestGetEscapedDefectsPagesOnlyTheEscaped(t *testing.T) {
	released := now.AddDate(0, -1, 0)
	r := &repo{releases: []domain.DefectRelease{
		{Defect: domain.CodeTitle{Code: "DEF-3"}, CreatedAt: now.Add(-time.Hour), ReleasedAt: &released},
		{Defect: domain.CodeTitle{Code: "DEF-2"}, CreatedAt: released.Add(-time.Hour), ReleasedAt: &released},
		{Defect: domain.CodeTitle{Code: "DEF-1"}, CreatedAt: released.Add(time.Hour), ReleasedAt: &released},
	}}
	uc := setup(t, r)
	out, err := uc.GetEscapedDefects.Execute(context.Background(), usecase.GetEscapedDefectsInput{Severity: "medium", Page: 2, Size: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.defectFilter.Severity != "medium" || out.Total != 2 || len(out.Rows) != 1 || out.Rows[0].Defect.Code != "DEF-1" ||
		out.Metrics.Defects != 3 || out.Metrics.RatePercent != 66.7 || len(out.Metrics.Months) != 12 {
		t.Errorf("salida: %+v", out)
	}
	if _, err := uc.GetEscapedDefects.Execute(context.Background(), usecase.GetEscapedDefectsInput{Severity: "x", Page: 1, Size: 25}); !slices.Equal(fieldsOf(err), []string{"severity"}) {
		t.Errorf("severidad inválida: %v", err)
	}
}
