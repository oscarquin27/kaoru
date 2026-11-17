// Package api expone el alta y la edición de sistemas por HTTP. La lista (GET /api/systems) la
// sirve traceability.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/system/application/usecase"
	"kaoru/internal/feature/system/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de system.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas. Un sistema no se borra.
func (h *Handler) Register(r gin.IRouter) {
	r.POST("/api/systems", h.create)
	r.PATCH("/api/systems/:key", h.update)
}

type systemDTO struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	DefaultRoute string `json:"defaultRoute"`
}

type systemCreateDTO struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	DefaultRoute string `json:"defaultRoute"`
}

type systemPatchDTO struct {
	Name         *string `json:"name"`
	DefaultRoute *string `json:"defaultRoute"`
}

func toDTO(s domain.Snapshot) systemDTO {
	return systemDTO{Key: s.Key, Name: s.Name, DefaultRoute: string(s.DefaultRoute)}
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func (h *Handler) create(c *gin.Context) {
	var body systemCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateSystem.Execute(c.Request.Context(), usecase.CreateSystemInput{
		Actor: actorOf(c), Key: body.Key, Name: body.Name, DefaultRoute: body.DefaultRoute,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(out))
}

func (h *Handler) update(c *gin.Context) {
	var body systemPatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateSystem.Execute(c.Request.Context(), usecase.UpdateSystemInput{
		Actor: actorOf(c), Key: c.Param("key"), Edit: domain.Edit{Name: body.Name, DefaultRoute: body.DefaultRoute},
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}
