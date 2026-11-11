// Package api expone los flujos por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/workflow/application/usecase"
	"kaoru/internal/feature/workflow/domain"
)

// Handler atiende las rutas de workflow.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/workflows/:entityType", h.getWorkflow)
}

type stateDTO struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Position  *int   `json:"position,omitempty"`
	IsInitial bool   `json:"isInitial"`
	IsFinal   bool   `json:"isFinal"`
}

type ruleDTO struct {
	ID      string         `json:"id"`
	RuleKey string         `json:"ruleKey"`
	Params  map[string]any `json:"params"`
	Route   string         `json:"route,omitempty"`
	Blocks  bool           `json:"blocks"`
	Enabled bool           `json:"enabled"`
}

type transitionDTO struct {
	From           string    `json:"from"`
	To             string    `json:"to"`
	Permission     string    `json:"permission"`
	RequiresReason bool      `json:"requiresReason"`
	Rules          []ruleDTO `json:"rules"`
}

type workflowDTO struct {
	EntityType  string          `json:"entityType"`
	States      []stateDTO      `json:"states"`
	Transitions []transitionDTO `json:"transitions"`
}

func (h *Handler) getWorkflow(c *gin.Context) {
	wf, err := h.uc.GetWorkflow.Execute(c.Request.Context(), usecase.GetWorkflowInput{EntityType: c.Param("entityType")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkflowDTO(wf))
}

func toWorkflowDTO(wf domain.Workflow) workflowDTO {
	dto := workflowDTO{
		EntityType:  string(wf.EntityType),
		States:      make([]stateDTO, len(wf.States)),
		Transitions: make([]transitionDTO, len(wf.Transitions)),
	}
	for i, s := range wf.States {
		dto.States[i] = stateDTO{Key: s.Key, Name: s.Name, Position: s.Position, IsInitial: s.IsInitial, IsFinal: s.IsFinal}
	}
	for i, t := range wf.Transitions {
		rules := make([]ruleDTO, len(t.Rules))
		for j, r := range t.Rules {
			params := r.Params
			if params == nil {
				params = map[string]any{}
			}
			rules[j] = ruleDTO{ID: r.Id, RuleKey: r.Key, Params: params, Route: r.Route, Blocks: r.Blocks, Enabled: r.Enabled}
		}
		dto.Transitions[i] = transitionDTO{
			From: t.From, To: t.To, Permission: t.Permission, RequiresReason: t.RequiresReason, Rules: rules,
		}
	}
	return dto
}
