// Package api expone los informes de conformidad por HTTP.
package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/mdpdf"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/report/application/usecase"
	"kaoru/internal/feature/report/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de report.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas. Un RPT no tiene ruta para editarse ni borrarse.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/reports", h.list)
	r.POST("/api/reports", h.issue)
	r.GET("/api/reports/:code", h.get)
	r.GET("/api/reports/:code/export", h.export)
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func (h *Handler) list(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListReports.Execute(c.Request.Context(), usecase.ListReportsInput{PackageCode: c.Query("package"), Page: page, Size: size})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]reportDTO, len(out.Reports))
	for i, s := range out.Reports {
		items[i] = toReportDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) issue(c *gin.Context) {
	var body issueDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.IssueReport.Execute(c.Request.Context(), usecase.IssueReportInput{Actor: actorOf(c), PackageCode: body.Package})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toReportDTO(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetReport.Execute(c.Request.Context(), usecase.GetReportInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toReportDTO(out))
}

// export descarga el Markdown tal como se emitió o el PDF que se genera de él, con la huella en el pie.
func (h *Handler) export(c *gin.Context) {
	format := c.Query("format")
	if format != "markdown" && format != "pdf" {
		problem.Write(c, axis.Invalid(axis.InvalidField("format", "El formato debe ser markdown o pdf.")))
		return
	}
	s, err := h.uc.GetReport.Execute(c.Request.Context(), usecase.GetReportInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	name := fmt.Sprintf("informe-%s-v%d", s.Code, s.Version)
	c.Header("X-Content-SHA256", s.Checksum)
	if format == "markdown" {
		c.Header("Content-Disposition", `attachment; filename="`+name+`.md"`)
		c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(s.Content))
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+name+`.pdf"`)
	c.Data(http.StatusOK, "application/pdf", mdpdf.Render(strings.TrimPrefix(s.Content, domain.BrandLine), pdfOptions(s)))
}

func pdfOptions(s domain.Snapshot) mdpdf.Options {
	title := domain.Title(s.Code.String())
	return mdpdf.Options{
		Title: title, Author: s.IssuedBy.Name, CreatedAt: s.CreatedAt, Brand: domain.BrandName, Motto: domain.BrandMotto,
		Footer: []string{
			fmt.Sprintf("%s · %s · versión %d · emitido el %s", domain.BrandName, title, s.Version, s.CreatedAt.UTC().Format("2006-01-02 15:04 UTC")),
			"SHA-256 del Markdown emitido: " + s.Checksum,
		},
	}
}
