// Package api expone los hitos, el calendario, las actividades y la planificación por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/planning/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de planning.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas. Un hito no se borra.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/milestones", h.list)
	r.POST("/api/milestones", h.create)
	r.PATCH("/api/milestones/:id", h.update)
	r.GET("/api/calendar", h.calendar)
	r.GET("/api/activities", h.listActivities)
	r.POST("/api/activities", h.createActivity)
	r.GET("/api/activities/:code", h.getActivity)
	r.PATCH("/api/activities/:code", h.updateActivity)
	r.GET("/api/planning/schedule", h.schedule)
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
	out, err := h.uc.ListMilestones.Execute(c.Request.Context(), usecase.ListMilestonesInput{Status: c.Query("status"), Page: page, Size: size})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]milestoneDTO, len(out.Milestones))
	for i, m := range out.Milestones {
		items[i] = toMilestoneDTO(m)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) create(c *gin.Context) {
	var body milestoneInputDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateMilestone.Execute(c.Request.Context(), usecase.CreateMilestoneInput{
		Actor: actorOf(c), Title: body.Title, DueDate: body.DueDate, Owner: body.Owner, Notes: body.Notes,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toMilestoneDTO(out))
}

func (h *Handler) update(c *gin.Context) {
	var body milestonePatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateMilestone.Execute(c.Request.Context(), usecase.UpdateMilestoneInput{
		Actor: actorOf(c), Id: c.Param("id"), Title: body.Title, DueDate: body.DueDate, Owner: body.Owner,
		Notes: body.Notes, Completed: body.Completed,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toMilestoneDTO(out))
}

func (h *Handler) calendar(c *gin.Context) {
	out, err := h.uc.GetCalendar.Execute(c.Request.Context())
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toCalendarDTO(out))
}
