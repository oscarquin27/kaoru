// Package api expone las lecturas de trazabilidad por HTTP.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/traceability/application/usecase"
	"kaoru/internal/feature/traceability/domain"
)

// Handler atiende las rutas de traceability.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/systems", h.systems)
	r.GET("/api/search", h.search)
	r.GET("/api/requirements/:code/overview", h.overview)
	r.GET("/api/traceability/matrix", h.matrix)
	r.GET("/api/traceability/matrix/export", h.export)
}

func (h *Handler) systems(c *gin.Context) {
	systems, err := h.uc.ListSystems.Execute(c.Request.Context())
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]systemDTO, len(systems))
	for i, s := range systems {
		items[i] = systemDTO{Key: s.Key, Name: s.Name, DefaultRoute: s.DefaultRoute}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) search(c *gin.Context) {
	hits, err := h.uc.Search.Execute(c.Request.Context(), usecase.SearchInput{Query: c.Query("q")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]searchHitDTO, len(hits))
	for i, hit := range hits {
		items[i] = searchHitDTO{Type: hit.Kind, Code: hit.Code, Title: hit.Title}
		if hit.State.Key != "" {
			state := toState(hit.State)
			items[i].State = &state
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) overview(c *gin.Context) {
	out, err := h.uc.GetRequirementOverview.Execute(c.Request.Context(), usecase.GetRequirementOverviewInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toOverview(out))
}

func (h *Handler) matrix(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	filter, err := matrixFilter(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.GetMatrix.Execute(c.Request.Context(), usecase.GetMatrixInput{Filter: filter, Page: page, Size: size})
	if err != nil {
		problem.Write(c, err)
		return
	}
	dto := matrixDTO{
		Indicators: indicatorsDTO{
			CoveragePercent: out.Indicators.CoveragePercent, RequiredTestCases: out.Indicators.RequiredTests,
			PassedTestCases: out.Indicators.PassedTests, BlockedRequirements: out.Indicators.BlockedRequirements,
			OpenDefects: out.Indicators.OpenDefects,
		},
		Items: make([]matrixRowDTO, len(out.Rows)), Page: page, Size: size, Total: out.Total,
	}
	for i, r := range out.Rows {
		dto.Items[i] = toMatrixRow(r)
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) export(c *gin.Context) {
	format := c.Query("format")
	if format != "csv" && format != "xlsx" {
		problem.Write(c, axis.Invalid(axis.InvalidField("format", "El formato debe ser csv o xlsx.")))
		return
	}
	filter, err := matrixFilter(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	rows, err := h.uc.ExportMatrix.Execute(c.Request.Context(), usecase.ExportMatrixInput{Filter: filter})
	if err != nil {
		problem.Write(c, err)
		return
	}
	if format == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="matriz-trazabilidad.csv"`)
		_ = writeCSV(c.Writer, rows)
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="matriz-trazabilidad.xlsx"`)
	_ = writeXLSX(c.Writer, rows)
}

// matrixFilter lee los filtros de la consulta. `from` y `to` son fechas (2026-11-11); `to` es inclusivo.
func matrixFilter(c *gin.Context) (domain.MatrixFilter, error) {
	filter := domain.MatrixFilter{
		ChangeCode: c.Query("change"), SystemKey: c.Query("system"), StateKey: c.Query("state"),
		RiskLevel: c.Query("risk"), OwnerId: c.Query("owner"),
		PackageCode: c.Query("package"), ComponentCode: c.Query("component"),
	}
	var errs axis.FieldErrors
	if raw := c.Query("from"); raw != "" {
		if from, err := time.Parse(time.DateOnly, raw); err != nil {
			errs.Add(axis.InvalidField("from", "La fecha debe tener el formato 2026-11-11."))
		} else {
			filter.CreatedFrom = &from
		}
	}
	if raw := c.Query("to"); raw != "" {
		if to, err := time.Parse(time.DateOnly, raw); err != nil {
			errs.Add(axis.InvalidField("to", "La fecha debe tener el formato 2026-11-11."))
		} else {
			end := to.AddDate(0, 0, 1)
			filter.CreatedTo = &end
		}
	}
	return filter, errs.Err()
}
