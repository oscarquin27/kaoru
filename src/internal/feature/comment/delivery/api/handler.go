// Package api expone los comentarios por HTTP.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/comment/application/usecase"
	"kaoru/internal/feature/comment/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de comment.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/comments", h.list)
	r.POST("/api/comments", h.create)
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type commentDTO struct {
	ID        string     `json:"id"`
	Body      string     `json:"body"`
	Author    userRefDTO `json:"author"`
	CreatedAt time.Time  `json:"createdAt"`
}

type createDTO struct {
	EntityType string `json:"entityType"`
	EntityCode string `json:"entityCode"`
	Body       string `json:"body"`
}

func toDTO(c domain.Comment) commentDTO {
	return commentDTO{ID: c.Id, Body: c.Body, Author: userRefDTO{ID: string(c.AuthorId), DisplayName: c.AuthorName}, CreatedAt: c.CreatedAt}
}

func (h *Handler) list(c *gin.Context) {
	comments, err := h.uc.ListComments.Execute(c.Request.Context(), usecase.ListCommentsInput{
		EntityType: c.Query("entityType"), EntityCode: c.Query("entityCode"),
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]commentDTO, len(comments))
	for i, comment := range comments {
		items[i] = toDTO(comment)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) create(c *gin.Context) {
	var body createDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	p := oidc.PrincipalFrom(c)
	comment, err := h.uc.CreateComment.Execute(c.Request.Context(), usecase.CreateCommentInput{
		Actor:      shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles},
		EntityType: body.EntityType, EntityCode: body.EntityCode, Body: body.Body,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(comment))
}
