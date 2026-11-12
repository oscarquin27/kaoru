// Package api expone los BUILD por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/build/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de build.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas. Un BUILD no tiene ruta para editarse ni borrarse.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/builds", h.list)
	r.POST("/api/builds", h.register)
	r.GET("/api/builds/:code", h.get)
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
	out, err := h.uc.ListBuilds.Execute(c.Request.Context(), usecase.ListBuildsInput{
		SystemKey: c.Query("system"), RequirementCode: c.Query("requirement"), Query: c.Query("q"), Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	dto := pageDTO{Items: make([]buildDTO, len(out.Builds)), Page: page, Size: size, Total: out.Total}
	for i, s := range out.Builds {
		dto.Items[i] = toDTO(s)
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) register(c *gin.Context) {
	var body registerDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.RegisterBuild.Execute(c.Request.Context(), usecase.RegisterBuildInput{
		Actor: actorOf(c), SystemKey: body.System, Version: body.Version, Checksum: body.Checksum,
		RequirementCodes: body.Requirements, Migrations: body.Migrations, Dependencies: body.Dependencies, Notes: body.Notes,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetBuild.Execute(c.Request.Context(), usecase.GetBuildInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}
