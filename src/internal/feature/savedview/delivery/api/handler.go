// Package api expone las vistas guardadas por HTTP.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/savedview/application/usecase"
	"kaoru/internal/feature/savedview/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de savedview.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/saved-views", h.list)
	r.POST("/api/saved-views", h.save)
	r.DELETE("/api/saved-views/:id", h.remove)
}

type viewDTO struct {
	ID        string    `json:"id"`
	Screen    string    `json:"screen"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	CreatedAt time.Time `json:"createdAt"`
}

type saveDTO struct {
	Screen string `json:"screen"`
	Name   string `json:"name"`
	Query  string `json:"query"`
}

func toDTO(v domain.View) viewDTO {
	return viewDTO{ID: v.Id, Screen: v.Screen, Name: v.Name, Query: v.Query, CreatedAt: v.CreatedAt}
}

func owner(c *gin.Context) shared.UserId { return shared.UserId(oidc.PrincipalFrom(c).Sub) }

func (h *Handler) list(c *gin.Context) {
	views, err := h.uc.ListViews.Execute(c.Request.Context(), owner(c), c.Query("screen"))
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]viewDTO, len(views))
	for i, v := range views {
		items[i] = toDTO(v)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) save(c *gin.Context) {
	var body saveDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	view, err := h.uc.SaveView.Execute(c.Request.Context(), usecase.SaveViewInput{
		Owner: owner(c), Screen: body.Screen, Name: body.Name, Query: body.Query,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(view))
}

func (h *Handler) remove(c *gin.Context) {
	if err := h.uc.DeleteView.Execute(c.Request.Context(), owner(c), c.Param("id")); err != nil {
		problem.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
