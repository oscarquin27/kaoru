// Package api expone los RUN y los EXP por HTTP.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/execution/application/usecase"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de execution.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas. Un RUN no tiene ruta para editarse ni borrarse.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/runs", h.listRuns)
	r.POST("/api/runs", h.createRun)
	r.GET("/api/runs/:code", h.getRun)
	r.GET("/api/explorations", h.listExplorations)
	r.POST("/api/explorations", h.createExploration)
	r.GET("/api/explorations/:code", h.getExploration)
	r.PATCH("/api/explorations/:code", h.updateExploration)
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type entityRefDTO struct {
	Type  string `json:"type"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

type runDTO struct {
	Code            string       `json:"code"`
	Result          string       `json:"result"`
	BuildVersion    string       `json:"buildVersion"`
	EnvironmentName string       `json:"environmentName"`
	Build           string       `json:"build,omitempty"`
	Environment     string       `json:"environment,omitempty"`
	ExecutedAt      time.Time    `json:"executedAt"`
	TestCase        entityRefDTO `json:"testCase"`
	BuildChecksum   string       `json:"buildChecksum"`
	Notes           string       `json:"notes"`
	ExecutedBy      userRefDTO   `json:"executedBy"`
	CreatedAt       time.Time    `json:"createdAt"`
}

type runCreateDTO struct {
	TestCase    string    `json:"testCase"`
	Build       string    `json:"build"`
	Environment string    `json:"environment"`
	Result      string    `json:"result"`
	Notes       string    `json:"notes"`
	ExecutedAt  time.Time `json:"executedAt"`
}

type explorationDTO struct {
	Code            string     `json:"code"`
	Requirement     string     `json:"requirement"`
	BuildVersion    string     `json:"buildVersion"`
	EnvironmentName string     `json:"environmentName"`
	Build           string     `json:"build,omitempty"`
	Environment     string     `json:"environment,omitempty"`
	ManualChanges   string     `json:"manualChanges"`
	Findings        string     `json:"findings"`
	ExecutedBy      userRefDTO `json:"executedBy"`
	ExecutedAt      time.Time  `json:"executedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type explorationCreateDTO struct {
	Requirement   string    `json:"requirement"`
	Build         string    `json:"build"`
	Environment   string    `json:"environment"`
	ManualChanges string    `json:"manualChanges"`
	Findings      string    `json:"findings"`
	ExecutedAt    time.Time `json:"executedAt"`
}

type explorationPatchDTO struct {
	ManualChanges *string `json:"manualChanges"`
	Findings      *string `json:"findings"`
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func toRunDTO(s domain.RunSnapshot) runDTO {
	return runDTO{
		Code: s.Code.String(), Result: string(s.Result), BuildVersion: s.BuildVersion, EnvironmentName: s.EnvironmentName,
		Build: s.BuildCode, Environment: s.EnvironmentCode,
		ExecutedAt: s.ExecutedAt, TestCase: entityRefDTO{Type: "test_case", Code: s.TestCase.Code, Title: s.TestCase.Title},
		BuildChecksum: s.BuildChecksum, Notes: s.Notes,
		ExecutedBy: userRefDTO{ID: string(s.ExecutedBy.Id), DisplayName: s.ExecutedBy.Name}, CreatedAt: s.CreatedAt,
	}
}

func toExplorationDTO(s domain.ExplorationSnapshot) explorationDTO {
	return explorationDTO{
		Code: s.Code.String(), Requirement: s.Requirement.Code, BuildVersion: s.BuildVersion,
		EnvironmentName: s.EnvironmentName, Build: s.BuildCode, Environment: s.EnvironmentCode, ManualChanges: s.ManualChanges, Findings: s.Findings,
		ExecutedBy: userRefDTO{ID: string(s.ExecutedBy.Id), DisplayName: s.ExecutedBy.Name},
		ExecutedAt: s.ExecutedAt, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func (h *Handler) listRuns(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListRuns.Execute(c.Request.Context(), usecase.ListRunsInput{
		TestCaseCode: c.Query("testCase"), RequirementCode: c.Query("requirement"),
		BuildVersion: c.Query("buildVersion"), Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]runDTO, len(out.Runs))
	for i, s := range out.Runs {
		items[i] = toRunDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) createRun(c *gin.Context) {
	var body runCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateRun.Execute(c.Request.Context(), usecase.CreateRunInput{
		Actor: actorOf(c), TestCaseCode: body.TestCase, BuildCode: body.Build,
		EnvironmentCode: body.Environment, Result: body.Result, Notes: body.Notes, ExecutedAt: body.ExecutedAt,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toRunDTO(out))
}

func (h *Handler) getRun(c *gin.Context) {
	out, err := h.uc.GetRun.Execute(c.Request.Context(), usecase.GetRunInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toRunDTO(out))
}

func (h *Handler) listExplorations(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListExplorations.Execute(c.Request.Context(), usecase.ListExplorationsInput{
		RequirementCode: c.Query("requirement"), Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]explorationDTO, len(out.Explorations))
	for i, s := range out.Explorations {
		items[i] = toExplorationDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) createExploration(c *gin.Context) {
	var body explorationCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateExploration.Execute(c.Request.Context(), usecase.CreateExplorationInput{
		Actor: actorOf(c), RequirementCode: body.Requirement, BuildCode: body.Build,
		EnvironmentCode: body.Environment, ManualChanges: body.ManualChanges, Findings: body.Findings,
		ExecutedAt: body.ExecutedAt,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toExplorationDTO(out))
}

func (h *Handler) getExploration(c *gin.Context) {
	out, err := h.uc.GetExploration.Execute(c.Request.Context(), usecase.GetExplorationInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toExplorationDTO(out))
}

func (h *Handler) updateExploration(c *gin.Context) {
	var body explorationPatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateExploration.Execute(c.Request.Context(), usecase.UpdateExplorationInput{
		Actor: actorOf(c), Code: c.Param("code"), ManualChanges: body.ManualChanges, Findings: body.Findings,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toExplorationDTO(out))
}
