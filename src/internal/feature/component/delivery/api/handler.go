// Package api expone el catálogo de componentes por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/component/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de component.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/components", h.list)
	r.POST("/api/components", h.create)
	r.GET("/api/components/:code", h.get)
	r.PATCH("/api/components/:code", h.update)
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
	out, err := h.uc.ListComponents.Execute(c.Request.Context(), usecase.ListComponentsInput{
		SystemKey: c.Query("system"), Kind: c.Query("kind"), Query: c.Query("q"),
		IncludeRetired: c.Query("includeRetired") == "true", Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	dto := pageDTO{Items: make([]componentDTO, len(out.Components)), Page: page, Size: size, Total: out.Total}
	for i, v := range out.Components {
		dto.Items[i] = toDTO(v)
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) create(c *gin.Context) {
	var body createDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateComponent.Execute(c.Request.Context(), usecase.CreateComponentInput{
		Actor: actorOf(c), SystemKey: body.System, Kind: body.Kind, Name: body.Name, Description: body.Description,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetComponent.Execute(c.Request.Context(), usecase.GetComponentInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *Handler) update(c *gin.Context) {
	var body patchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateComponent.Execute(c.Request.Context(), usecase.UpdateComponentInput{
		Actor: actorOf(c), Code: c.Param("code"), Edit: body.toEdit(),
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}
