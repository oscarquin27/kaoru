// Package api expone el checklist de producción, los diagnósticos y las pruebas de humo por HTTP.
package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/production/application/usecase"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de production.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas. Nada de este feature tiene ruta para editarse ni borrarse.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/checklist-templates", h.listTemplates)
	r.POST("/api/checklist-templates", h.publishTemplate)
	r.GET("/api/checklists", h.listChecklists)
	r.POST("/api/checklists", h.startChecklist)
	r.GET("/api/checklists/:code", h.getChecklist)
	r.POST("/api/checklists/:code/steps/:position/signature", h.signStep)
	r.GET("/api/diagnostics", h.listDiagnostics)
	r.POST("/api/diagnostics", h.submitDiagnostic)
	r.GET("/api/diagnostics/:code", h.getDiagnostic)
	r.POST("/api/diagnostics/:code/review", h.reviewDiagnostic)
	r.GET("/api/smoke-tests", h.listSmokeTests)
	r.POST("/api/smoke-tests", h.registerSmokeTest)
	r.GET("/api/smoke-tests/:code", h.getSmokeTest)
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

// listInput lee la página y el filtro por PKG que comparten las tres listas.
func listInput(c *gin.Context) (usecase.ListInput, error) {
	page, size, err := httpserver.PageParams(c)
	return usecase.ListInput{PackageCode: c.Query("package"), Page: page, Size: size}, err
}

func (h *Handler) listTemplates(c *gin.Context) {
	out, err := h.uc.ListChecklistTemplates.Execute(c.Request.Context(), usecase.ListChecklistTemplatesInput{SystemKey: c.Query("system")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]templateDTO, len(out))
	for i, s := range out {
		items[i] = toTemplateDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) publishTemplate(c *gin.Context) {
	var body templateCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	steps := make([]domain.StepInput, len(body.Steps))
	for i, s := range body.Steps {
		steps[i] = domain.StepInput{Title: s.Title, Description: s.Description}
	}
	out, err := h.uc.PublishChecklistTemplate.Execute(c.Request.Context(), usecase.PublishChecklistTemplateInput{
		Actor: actorOf(c), SystemKey: body.System, Steps: steps,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTemplateDTO(out))
}

func (h *Handler) listChecklists(c *gin.Context) {
	in, err := listInput(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListChecklists.Execute(c.Request.Context(), in)
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]checklistDTO, len(out.Checklists))
	for i, o := range out.Checklists {
		items[i] = toChecklistDTO(o)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": in.Page, "size": in.Size, "total": out.Total})
}

func (h *Handler) startChecklist(c *gin.Context) {
	var body checklistCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.StartChecklist.Execute(c.Request.Context(), usecase.StartChecklistInput{
		Actor: actorOf(c), PackageCode: body.Package, EnvironmentCode: body.Environment,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toChecklistDTO(out))
}

func (h *Handler) getChecklist(c *gin.Context) {
	out, err := h.uc.GetChecklist.Execute(c.Request.Context(), usecase.GetChecklistInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toChecklistDTO(out))
}

func (h *Handler) signStep(c *gin.Context) {
	var body signDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	// Un paso que no es un número no existe: el dominio responde por la posición cero.
	position, _ := strconv.Atoi(c.Param("position"))
	out, err := h.uc.SignChecklistStep.Execute(c.Request.Context(), usecase.SignChecklistStepInput{
		Actor: actorOf(c), Code: c.Param("code"), Position: position, Comment: body.Comment,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toChecklistDTO(out))
}

func (h *Handler) listDiagnostics(c *gin.Context) {
	in, err := listInput(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListDiagnostics.Execute(c.Request.Context(), in)
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]diagnosticDTO, len(out.Diagnostics))
	for i, s := range out.Diagnostics {
		items[i] = toDiagnosticDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": in.Page, "size": in.Size, "total": out.Total})
}

func (h *Handler) submitDiagnostic(c *gin.Context) {
	var body diagnosticCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.SubmitDiagnostic.Execute(c.Request.Context(), usecase.SubmitDiagnosticInput{
		Actor: actorOf(c), DeploymentCode: body.Deployment, Notes: body.Notes,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDiagnosticDTO(out))
}

func (h *Handler) getDiagnostic(c *gin.Context) {
	out, err := h.uc.GetDiagnostic.Execute(c.Request.Context(), usecase.GetDiagnosticInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDiagnosticDTO(out))
}

func (h *Handler) reviewDiagnostic(c *gin.Context) {
	var body reviewCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ReviewDiagnostic.Execute(c.Request.Context(), usecase.ReviewDiagnosticInput{
		Actor: actorOf(c), Code: c.Param("code"), Result: body.Result, Notes: body.Notes,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDiagnosticDTO(out))
}

func (h *Handler) listSmokeTests(c *gin.Context) {
	in, err := listInput(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListSmokeTests.Execute(c.Request.Context(), in)
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]smokeTestDTO, len(out.SmokeTests))
	for i, s := range out.SmokeTests {
		items[i] = toSmokeTestDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": in.Page, "size": in.Size, "total": out.Total})
}

func (h *Handler) registerSmokeTest(c *gin.Context) {
	var body smokeTestCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.RegisterSmokeTest.Execute(c.Request.Context(), usecase.RegisterSmokeTestInput{
		Actor: actorOf(c), DeploymentCode: body.Deployment, Result: body.Result, Notes: body.Notes,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toSmokeTestDTO(out))
}

func (h *Handler) getSmokeTest(c *gin.Context) {
	out, err := h.uc.GetSmokeTest.Execute(c.Request.Context(), usecase.GetSmokeTestInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toSmokeTestDTO(out))
}
