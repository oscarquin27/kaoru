// Package api expone los flujos por HTTP.
package api

import (
	"encoding/json"
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
	r.POST("/api/admin/workflows/:entityType/states", h.addState)
	r.PATCH("/api/admin/workflows/:entityType/states/:key", h.updateState)
	r.DELETE("/api/admin/workflows/:entityType/states/:key", h.removeState)
	r.PUT("/api/admin/workflows/:entityType/transitions/:from/:to", h.setTransition)
	r.DELETE("/api/admin/workflows/:entityType/transitions/:from/:to", h.removeTransition)
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

type newStateDTO struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Position *int   `json:"position"`
	IsFinal  bool   `json:"isFinal"`
}

// nullableInt distingue «no vino» de «vino null» en un PATCH.
type nullableInt struct {
	set   bool
	value *int
}

func (n *nullableInt) UnmarshalJSON(data []byte) error {
	n.set = true
	if string(data) == "null" {
		return nil
	}
	return json.Unmarshal(data, &n.value)
}

type statePatchDTO struct {
	Name     *string     `json:"name"`
	Position nullableInt `json:"position"`
}

type transitionPutDTO struct {
	Permission     string `json:"permission"`
	RequiresReason bool   `json:"requiresReason"`
}

// respond escribe el flujo como quedó, o el error.
func respond(c *gin.Context, status int, wf domain.Workflow, err error) {
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(status, toWorkflowDTO(wf))
}

func (h *Handler) addState(c *gin.Context) {
	var body newStateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	wf, err := h.uc.AddState.Execute(c.Request.Context(), usecase.AddStateInput{
		Actor: actorOf(c), EntityType: c.Param("entityType"),
		Key: body.Key, Name: body.Name, Position: body.Position, IsFinal: body.IsFinal,
	})
	respond(c, http.StatusCreated, wf, err)
}

func (h *Handler) updateState(c *gin.Context) {
	var body statePatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	wf, err := h.uc.UpdateState.Execute(c.Request.Context(), usecase.UpdateStateInput{
		Actor: actorOf(c), EntityType: c.Param("entityType"), Key: c.Param("key"),
		Edit: domain.StateEdit{Name: body.Name, PositionSet: body.Position.set, Position: body.Position.value},
	})
	respond(c, http.StatusOK, wf, err)
}

func (h *Handler) removeState(c *gin.Context) {
	wf, err := h.uc.RemoveState.Execute(c.Request.Context(), usecase.RemoveStateInput{
		Actor: actorOf(c), EntityType: c.Param("entityType"), Key: c.Param("key"),
	})
	respond(c, http.StatusOK, wf, err)
}

func (h *Handler) setTransition(c *gin.Context) {
	var body transitionPutDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	wf, err := h.uc.SetTransition.Execute(c.Request.Context(), usecase.SetTransitionInput{
		Actor: actorOf(c), EntityType: c.Param("entityType"), From: c.Param("from"), To: c.Param("to"),
		Permission: body.Permission, RequiresReason: body.RequiresReason,
	})
	respond(c, http.StatusOK, wf, err)
}

func (h *Handler) removeTransition(c *gin.Context) {
	wf, err := h.uc.RemoveTransition.Execute(c.Request.Context(), usecase.RemoveTransitionInput{
		Actor: actorOf(c), EntityType: c.Param("entityType"), From: c.Param("from"), To: c.Param("to"),
	})
	respond(c, http.StatusOK, wf, err)
}
