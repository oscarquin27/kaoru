// Package api expone las métricas por HTTP.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/metrics/application/usecase"
	"kaoru/internal/feature/metrics/domain"
)

// Handler atiende las rutas de metrics.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/metrics/coverage", h.coverage)
	r.GET("/api/metrics/defect-states", h.defectStates)
	r.GET("/api/metrics/regression", h.regression)
	r.GET("/api/metrics/escaped-defects", h.escapedDefects)
}

func (h *Handler) coverage(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.GetCoverage.Execute(c.Request.Context(), usecase.GetCoverageInput{
		Filter: domain.CoverageFilter{
			SystemKey: c.Query("system"), ChangeCode: c.Query("change"), PackageCode: c.Query("package"), StateKey: c.Query("state"),
		},
		Status: c.Query("status"), Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	s := out.Summary
	dto := coverageDTO{
		pageDTO: pageDTO{Page: page, Size: size, Total: out.Total},
		Summary: coverageSummaryDTO{
			Requirements: s.Requirements, Below: s.ByStatus[domain.CoverageBelow], NoTestCases: s.ByStatus[domain.CoverageNoTestCases],
			Unassessed: s.ByStatus[domain.CoverageUnassessed], Meets: s.ByStatus[domain.CoverageMeets],
			RequiredTestCases: s.Required, PassedTestCases: s.Passed,
		},
		Items: make([]coverageRowDTO, len(out.Rows)),
	}
	for i, r := range out.Rows {
		dto.Items[i] = coverageRowDTO{
			Requirement: toCodeTitle(r.Requirement), System: r.SystemKey, State: toState(r.State), RiskLevel: r.RiskLevel,
			Required: r.Required, Passed: r.Passed, Percent: r.Percent(), MinimumPercent: r.MinimumPercent, Status: string(r.Status()),
		}
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) defectStates(c *gin.Context) {
	period, err := periodParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.GetDefectStates.Execute(c.Request.Context(), usecase.GetDefectStatesInput{
		SystemKey: c.Query("system"), Severity: c.Query("severity"), Period: period,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	m := out.Metrics
	dto := defectStatesDTO{
		Period: toPeriod(out.Period), Total: m.Total, States: make([]stateMetricDTO, len(m.States)),
		Resolution: resolutionDTO{Defects: m.Resolved, MedianDays: rounded(m.ResolutionMedianDays), P90Days: rounded(m.ResolutionP90Days)},
	}
	for i, s := range m.States {
		dto.States[i] = stateMetricDTO{
			State: toState(s.State), IsFinal: s.IsFinal, Current: s.Current, Defects: s.Defects, Visits: s.Visits,
			MedianDays: rounded(s.MedianDays), P90Days: rounded(s.P90Days),
		}
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) regression(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	period, err := periodParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.GetRegression.Execute(c.Request.Context(), usecase.GetRegressionInput{
		SystemKey: c.Query("system"), Period: period, Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	s := out.Summary
	dto := regressionDTO{
		pageDTO: pageDTO{Page: page, Size: size, Total: out.Total}, Period: toPeriod(out.Period),
		Summary: regressionSummaryDTO{
			Packages: s.Packages, Proposed: s.Proposed, Final: s.Final, Added: s.Added, Removed: s.Removed, MedianFinal: rounded(s.MedianFinal),
		},
		Items: make([]regressionRowDTO, len(out.Rows)),
	}
	for i, r := range out.Rows {
		dto.Items[i] = regressionRowDTO{
			Package: toCodeTitle(r.Package), System: r.SystemKey, State: toState(r.State), ConfirmedAt: r.ConfirmedAt,
			Proposed: r.Proposed, Final: r.Final, Added: r.Added, Removed: r.Removed,
			ByOrigin: originDTO{Impact: r.Impact, Critical: r.Critical, Manual: r.Manual}, Passed: r.Passed,
			SystemTestCases: r.SystemTestCases,
		}
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) escapedDefects(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	period, err := periodParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.GetEscapedDefects.Execute(c.Request.Context(), usecase.GetEscapedDefectsInput{
		SystemKey: c.Query("system"), Severity: c.Query("severity"), Period: period, Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	m := out.Metrics
	dto := escapedDTO{
		pageDTO: pageDTO{Page: page, Size: size, Total: out.Total}, Period: toPeriod(out.Period),
		Summary: escapedSummaryDTO{Defects: m.Defects, Escaped: len(m.Escaped), RatePercent: m.RatePercent},
		Months:  make([]monthDTO, len(m.Months)), Items: make([]escapedRowDTO, len(out.Rows)),
	}
	for i, month := range m.Months {
		dto.Months[i] = monthDTO{Month: month.Month, Defects: month.Defects, Escaped: month.Escaped}
	}
	for i, d := range out.Rows {
		dto.Items[i] = escapedRowDTO{
			Defect: toCodeTitle(d.Defect), Severity: d.Severity, State: toState(d.State), Requirement: d.RequirementCode,
			Package: d.PackageCode, ReleasedAt: *d.ReleasedAt, CreatedAt: d.CreatedAt, DaysAfterRelease: oneDecimal(d.DaysAfterRelease()),
		}
	}
	c.JSON(http.StatusOK, dto)
}

// periodParams lee `from` y `to` (2026-11-11, en UTC). `to` es inclusivo: el período termina al empezar el día siguiente.
func periodParams(c *gin.Context) (usecase.PeriodInput, error) {
	var in usecase.PeriodInput
	var errs axis.FieldErrors
	if raw := c.Query("from"); raw != "" {
		if from, err := time.Parse(time.DateOnly, raw); err != nil {
			errs.Add(axis.InvalidField("from", "La fecha debe tener el formato 2026-11-11."))
		} else {
			in.From = &from
		}
	}
	if raw := c.Query("to"); raw != "" {
		if to, err := time.Parse(time.DateOnly, raw); err != nil {
			errs.Add(axis.InvalidField("to", "La fecha debe tener el formato 2026-11-11."))
		} else {
			end := to.AddDate(0, 0, 1)
			in.To = &end
		}
	}
	return in, errs.Err()
}
