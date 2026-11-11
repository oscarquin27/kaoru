// Package api expone los flujos por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/workflow/application/usecase"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
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
	r.GET("/api/admin/role-permissions", h.listRolePermissions)
	r.PUT("/api/admin/role-permissions/:role", h.setRolePermissions)
	r.PATCH("/api/admin/blocking-rules/:id", h.updateBlockingRule)
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

type permissionDTO struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type rolePermissionsDTO struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

type rolePermissionsPageDTO struct {
	Permissions []permissionDTO      `json:"permissions"`
	Roles       []rolePermissionsDTO `json:"roles"`
}

type setPermissionsDTO struct {
	Permissions []string `json:"permissions"`
}

type rulePatchDTO struct {
	Enabled *bool          `json:"enabled"`
	Blocks  *bool          `json:"blocks"`
	Params  map[string]any `json:"params"`
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func toRolePermissionsDTO(out usecase.RolePermissionsOutput) rolePermissionsPageDTO {
	dto := rolePermissionsPageDTO{
		Permissions: make([]permissionDTO, len(out.Permissions)), Roles: make([]rolePermissionsDTO, len(out.Roles)),
	}
	for i, p := range out.Permissions {
		dto.Permissions[i] = permissionDTO{Key: p.Key, Description: p.Description}
	}
	for i, r := range out.Roles {
		dto.Roles[i] = rolePermissionsDTO{Role: r.Role, Permissions: r.Permissions}
	}
	return dto
}

func toRuleDTO(r domain.Rule) ruleDTO {
	params := r.Params
	if params == nil {
		params = map[string]any{}
	}
	return ruleDTO{ID: r.Id, RuleKey: r.Key, Params: params, Route: r.Route, Blocks: r.Blocks, Enabled: r.Enabled}
}

func (h *Handler) listRolePermissions(c *gin.Context) {
	out, err := h.uc.ListRolePermissions.Execute(c.Request.Context(), usecase.ListRolePermissionsInput{Actor: actorOf(c)})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toRolePermissionsDTO(out))
}

func (h *Handler) setRolePermissions(c *gin.Context) {
	var body setPermissionsDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.SetRolePermissions.Execute(c.Request.Context(), usecase.SetRolePermissionsInput{
		Actor: actorOf(c), Role: c.Param("role"), Permissions: body.Permissions,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toRolePermissionsDTO(out))
}

func (h *Handler) updateBlockingRule(c *gin.Context) {
	var body rulePatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	rule, err := h.uc.UpdateBlockingRule.Execute(c.Request.Context(), usecase.UpdateBlockingRuleInput{
		Actor: actorOf(c), RuleId: c.Param("id"), Enabled: body.Enabled, Blocks: body.Blocks, Params: body.Params,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toRuleDTO(rule))
}
