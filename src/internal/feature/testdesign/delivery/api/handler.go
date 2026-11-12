// Package api expone los UC y los TC por HTTP.
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/testdesign/application/usecase"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de testdesign.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/requirements/:code/functional-cases", h.listFunctionalCases)
	r.POST("/api/requirements/:code/functional-cases", h.createFunctionalCase)
	r.GET("/api/functional-cases/:code", h.getFunctionalCase)
	r.PATCH("/api/functional-cases/:code", h.updateFunctionalCase)
	r.GET("/api/test-cases", h.listTestCases)
	r.POST("/api/test-cases", h.createTestCase)
	r.GET("/api/test-cases/:code", h.getTestCase)
	r.PATCH("/api/test-cases/:code", h.updateTestCase)
	r.GET("/api/test-cases/:code/components", h.listTestCaseComponents)
	r.PUT("/api/test-cases/:code/components", h.setTestCaseComponents)
}

type functionalCaseDTO struct {
	Code        string    `json:"code"`
	Requirement string    `json:"requirement"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TestCases   []string  `json:"testCases"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type functionalCaseBody struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type systemDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type testCaseDTO struct {
	Code            string    `json:"code"`
	Title           string    `json:"title"`
	System          systemDTO `json:"system"`
	Preconditions   string    `json:"preconditions"`
	Steps           string    `json:"steps"`
	ExpectedResult  string    `json:"expectedResult"`
	IsCritical      bool      `json:"isCritical"`
	Retired         bool      `json:"retired"`
	FunctionalCases []string  `json:"functionalCases"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type testCasePageDTO struct {
	Items []testCaseDTO `json:"items"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Total int           `json:"total"`
}

type testCaseCreateDTO struct {
	Title           string   `json:"title"`
	System          string   `json:"system"`
	Preconditions   string   `json:"preconditions"`
	Steps           string   `json:"steps"`
	ExpectedResult  string   `json:"expectedResult"`
	IsCritical      bool     `json:"isCritical"`
	FunctionalCases []string `json:"functionalCases"`
}

type testCasePatchDTO struct {
	Title           *string   `json:"title"`
	Preconditions   *string   `json:"preconditions"`
	Steps           *string   `json:"steps"`
	ExpectedResult  *string   `json:"expectedResult"`
	IsCritical      *bool     `json:"isCritical"`
	FunctionalCases *[]string `json:"functionalCases"`
	Retired         *bool     `json:"retired"`
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func toFunctionalCaseDTO(s domain.FunctionalCaseSnapshot) functionalCaseDTO {
	return functionalCaseDTO{
		Code: s.Code.String(), Requirement: s.Requirement.Code, Title: s.Title, Description: s.Description,
		TestCases: s.TestCases, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func toTestCaseDTO(s domain.TestCaseSnapshot) testCaseDTO {
	dto := testCaseDTO{
		Code: s.Code.String(), Title: s.Title, System: systemDTO{Key: s.SystemKey, Name: s.SystemName},
		Preconditions: s.Preconditions, Steps: s.Steps, ExpectedResult: s.ExpectedResult, IsCritical: s.IsCritical,
		Retired: s.RetiredAt != nil, FunctionalCases: make([]string, len(s.FunctionalCases)),
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	for i, f := range s.FunctionalCases {
		dto.FunctionalCases[i] = f.Code
	}
	return dto
}

func (h *Handler) listFunctionalCases(c *gin.Context) {
	out, err := h.uc.ListFunctionalCases.Execute(c.Request.Context(), usecase.ListFunctionalCasesInput{RequirementCode: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]functionalCaseDTO, len(out))
	for i, s := range out {
		items[i] = toFunctionalCaseDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) createFunctionalCase(c *gin.Context) {
	var body functionalCaseBody
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	in := usecase.CreateFunctionalCaseInput{Actor: actorOf(c), RequirementCode: c.Param("code")}
	if body.Title != nil {
		in.Title = *body.Title
	}
	if body.Description != nil {
		in.Description = *body.Description
	}
	out, err := h.uc.CreateFunctionalCase.Execute(c.Request.Context(), in)
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toFunctionalCaseDTO(out))
}

func (h *Handler) getFunctionalCase(c *gin.Context) {
	out, err := h.uc.GetFunctionalCase.Execute(c.Request.Context(), usecase.GetFunctionalCaseInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toFunctionalCaseDTO(out))
}

func (h *Handler) updateFunctionalCase(c *gin.Context) {
	var body functionalCaseBody
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateFunctionalCase.Execute(c.Request.Context(), usecase.UpdateFunctionalCaseInput{
		Actor: actorOf(c), Code: c.Param("code"), Title: body.Title, Description: body.Description,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toFunctionalCaseDTO(out))
}

func (h *Handler) listTestCases(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	in := usecase.ListTestCasesInput{
		SystemKey: c.Query("system"), Query: c.Query("q"), RequirementCode: c.Query("requirement"), Page: page, Size: size,
	}
	if raw := c.Query("critical"); raw != "" {
		critical, err := strconv.ParseBool(raw)
		if err != nil {
			problem.Write(c, axis.Invalid(axis.InvalidField("critical", "Debe ser true o false.")))
			return
		}
		in.Critical = &critical
	}
	out, err := h.uc.ListTestCases.Execute(c.Request.Context(), in)
	if err != nil {
		problem.Write(c, err)
		return
	}
	dto := testCasePageDTO{Items: make([]testCaseDTO, len(out.TestCases)), Page: page, Size: size, Total: out.Total}
	for i, s := range out.TestCases {
		dto.Items[i] = toTestCaseDTO(s)
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) createTestCase(c *gin.Context) {
	var body testCaseCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateTestCase.Execute(c.Request.Context(), usecase.CreateTestCaseInput{
		Actor: actorOf(c), Title: body.Title, SystemKey: body.System, Preconditions: body.Preconditions,
		Steps: body.Steps, ExpectedResult: body.ExpectedResult, IsCritical: body.IsCritical, FunctionalCases: body.FunctionalCases,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTestCaseDTO(out))
}

func (h *Handler) getTestCase(c *gin.Context) {
	out, err := h.uc.GetTestCase.Execute(c.Request.Context(), usecase.GetTestCaseInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toTestCaseDTO(out))
}

func (h *Handler) updateTestCase(c *gin.Context) {
	var body testCasePatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateTestCase.Execute(c.Request.Context(), usecase.UpdateTestCaseInput{
		Actor: actorOf(c), Code: c.Param("code"), Title: body.Title, Preconditions: body.Preconditions,
		Steps: body.Steps, ExpectedResult: body.ExpectedResult, IsCritical: body.IsCritical,
		FunctionalCases: body.FunctionalCases, Retired: body.Retired,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toTestCaseDTO(out))
}

type componentRefDTO struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// componentsDTO es el cuerpo que declara los componentes de un TC; sin `components` no vale.
type componentsDTO struct {
	Components []string `json:"components"`
}

func toComponentDTOs(refs []domain.ComponentRef) []componentRefDTO {
	items := make([]componentRefDTO, len(refs))
	for i, r := range refs {
		items[i] = componentRefDTO{Code: r.Code, Name: r.Name, Kind: r.Kind}
	}
	return items
}

func (h *Handler) listTestCaseComponents(c *gin.Context) {
	refs, err := h.uc.ListTestCaseComponents.Execute(c.Request.Context(), usecase.ListTestCaseComponentsInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toComponentDTOs(refs)})
}

func (h *Handler) setTestCaseComponents(c *gin.Context) {
	var body componentsDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	refs, err := h.uc.SetTestCaseComponents.Execute(c.Request.Context(), usecase.SetTestCaseComponentsInput{
		Actor: actorOf(c), Code: c.Param("code"), Components: body.Components,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toComponentDTOs(refs)})
}
