// Package api expone los CHG por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/change/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de change.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/changes", h.list)
	r.POST("/api/changes", h.create)
	r.GET("/api/changes/:code", h.get)
	r.PATCH("/api/changes/:code", h.update)
	r.POST("/api/changes/:code/transitions", h.transition)
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
	out, err := h.uc.ListChanges.Execute(c.Request.Context(), usecase.ListChangesInput{
		SystemKey: c.Query("system"), StateKey: c.Query("state"), Query: c.Query("q"), Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	dto := pageDTO{Items: make([]changeDTO, len(out.Changes)), Page: page, Size: size, Total: out.Total}
	for i, v := range out.Changes {
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
	view, err := h.uc.CreateChange.Execute(c.Request.Context(), usecase.CreateChangeInput{
		Actor: actorOf(c), Title: body.Title, Description: body.Description, SystemKey: body.System,
		Route: body.Route, WindowStart: body.WindowStart, WindowEnd: body.WindowEnd,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(view))
}

func (h *Handler) get(c *gin.Context) {
	view, err := h.uc.GetChange.Execute(c.Request.Context(), usecase.GetChangeInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(view))
}

func (h *Handler) update(c *gin.Context) {
	var body patchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	view, err := h.uc.UpdateChange.Execute(c.Request.Context(), usecase.UpdateChangeInput{
		Actor: actorOf(c), Code: c.Param("code"), Edit: body.toEdit(),
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(view))
}

func (h *Handler) transition(c *gin.Context) {
	var body transitionDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	view, err := h.uc.TransitionChange.Execute(c.Request.Context(), usecase.TransitionChangeInput{
		Actor: actorOf(c), Code: c.Param("code"), To: body.To, Reason: body.Reason,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(view))
}
