// Package api expone los DEF por HTTP.
package api

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/defect/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de defect.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/defects", h.list)
	r.POST("/api/defects", h.create)
	r.GET("/api/defects/:code", h.get)
	r.PATCH("/api/defects/:code", h.update)
	r.POST("/api/defects/:code/transitions", h.transition)
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type stateDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type defectDTO struct {
	Code        string      `json:"code"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Requirement string      `json:"requirement"`
	Run         string      `json:"run"`
	Severity    string      `json:"severity"`
	State       stateDTO    `json:"state"`
	DaysInState float64     `json:"daysInState"`
	Assignee    *userRefDTO `json:"assignee,omitempty"`
	CreatedBy   userRefDTO  `json:"createdBy"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	// Acceptance sale solo si el DEF se aceptó al aprobar su REQ con defectos.
	Acceptance *acceptanceDTO `json:"acceptance,omitempty"`
}

type acceptanceDTO struct {
	By     userRefDTO `json:"by"`
	Reason string     `json:"reason"`
	At     time.Time  `json:"at"`
}

type createDTO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Requirement string `json:"requirement"`
	Run         string `json:"run"`
	Severity    string `json:"severity"`
	Assignee    string `json:"assignee"`
}

// nullableString distingue «no vino» de «vino null» en un PATCH.
type nullableString struct {
	set   bool
	value string
}

func (n *nullableString) UnmarshalJSON(data []byte) error {
	n.set = true
	if string(data) == "null" {
		return nil
	}
	return json.Unmarshal(data, &n.value)
}

type patchDTO struct {
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Severity    *string        `json:"severity"`
	Assignee    nullableString `json:"assignee"`
}

type transitionDTO struct {
	To     string `json:"to"`
	Reason string `json:"reason"`
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func toDTO(out usecase.DefectOutput) defectDTO {
	s := out.Defect
	dto := defectDTO{
		Code: s.Code.String(), Title: s.Title, Description: s.Description, Requirement: s.Requirement.Code,
		Run: s.Run.Code, Severity: string(s.Severity), State: stateDTO{Key: s.State.Key, Name: s.State.Name},
		// Un decimal basta para leer «3,5 días».
		DaysInState: math.Round(out.DaysInState*10) / 10,
		CreatedBy:   userRefDTO{ID: string(s.CreatedBy.Id), DisplayName: s.CreatedBy.Name},
		CreatedAt:   s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if s.Assignee != nil {
		dto.Assignee = &userRefDTO{ID: string(s.Assignee.Id), DisplayName: s.Assignee.Name}
	}
	if a := s.Acceptance; a != nil {
		dto.Acceptance = &acceptanceDTO{By: userRefDTO{ID: string(a.By.Id), DisplayName: a.By.Name}, Reason: a.Reason, At: a.At}
	}
	return dto
}

func (h *Handler) list(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListDefects.Execute(c.Request.Context(), usecase.ListDefectsInput{
		StateKey: c.Query("state"), RequirementCode: c.Query("requirement"), Severity: c.Query("severity"),
		AssigneeId: c.Query("assignee"), Query: c.Query("q"), Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]defectDTO, len(out.Defects))
	for i, d := range out.Defects {
		items[i] = toDTO(d)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) create(c *gin.Context) {
	var body createDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateDefect.Execute(c.Request.Context(), usecase.CreateDefectInput{
		Actor: actorOf(c), Title: body.Title, Description: body.Description, RequirementCode: body.Requirement,
		RunCode: body.Run, Severity: body.Severity, AssigneeId: body.Assignee,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetDefect.Execute(c.Request.Context(), usecase.GetDefectInput{Code: c.Param("code")})
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
	out, err := h.uc.UpdateDefect.Execute(c.Request.Context(), usecase.UpdateDefectInput{
		Actor: actorOf(c), Code: c.Param("code"), Title: body.Title, Description: body.Description,
		Severity: body.Severity, AssigneeSet: body.Assignee.set, AssigneeId: body.Assignee.value,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *Handler) transition(c *gin.Context) {
	var body transitionDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.TransitionDefect.Execute(c.Request.Context(), usecase.TransitionDefectInput{
		Actor: actorOf(c), Code: c.Param("code"), To: body.To, Reason: body.Reason,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}
