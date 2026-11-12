// Package api expone los ambientes y los despliegues por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/environment/application/usecase"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de environment.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas. Un despliegue no tiene ruta para editarse ni borrarse.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/environments", h.listEnvironments)
	r.POST("/api/environments", h.createEnvironment)
	r.GET("/api/environments/:code", h.getEnvironment)
	r.PATCH("/api/environments/:code", h.updateEnvironment)
	r.GET("/api/deployments", h.listDeployments)
	r.POST("/api/deployments", h.registerDeployment)
	r.GET("/api/deployments/:code", h.getDeployment)
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func (h *Handler) listEnvironments(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListEnvironments.Execute(c.Request.Context(), usecase.ListEnvironmentsInput{
		SystemKey: c.Query("system"), Kind: c.Query("kind"), IncludeRetired: c.Query("includeRetired") == "true",
		Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]environmentDTO, len(out.Environments))
	for i, v := range out.Environments {
		items[i] = toEnvironmentDTO(v)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) createEnvironment(c *gin.Context) {
	var body environmentCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateEnvironment.Execute(c.Request.Context(), usecase.CreateEnvironmentInput{
		Actor: actorOf(c), SystemKey: body.System, Name: body.Name, Kind: body.Kind, OS: body.OS, Notes: body.Notes,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toEnvironmentDTO(out))
}

func (h *Handler) getEnvironment(c *gin.Context) {
	out, err := h.uc.GetEnvironment.Execute(c.Request.Context(), usecase.GetEnvironmentInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toEnvironmentDTO(out))
}

func (h *Handler) updateEnvironment(c *gin.Context) {
	var body environmentPatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateEnvironment.Execute(c.Request.Context(), usecase.UpdateEnvironmentInput{
		Actor: actorOf(c), Code: c.Param("code"),
		Edit: domain.Edit{Name: body.Name, Kind: body.Kind, OS: body.OS, Notes: body.Notes, Retired: body.Retired},
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toEnvironmentDTO(out))
}

func (h *Handler) listDeployments(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListDeployments.Execute(c.Request.Context(), usecase.ListDeploymentsInput{
		EnvironmentCode: c.Query("environment"), BuildCode: c.Query("build"), PackageCode: c.Query("package"),
		Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]deploymentDTO, len(out.Deployments))
	for i, s := range out.Deployments {
		items[i] = toDeploymentDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) registerDeployment(c *gin.Context) {
	var body deploymentCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.RegisterDeployment.Execute(c.Request.Context(), usecase.RegisterDeploymentInput{
		Actor: actorOf(c), EnvironmentCode: body.Environment, BuildCode: body.Build, PackageCode: body.Package,
		Result: body.Result, Notes: body.Notes, ExecutedAt: body.ExecutedAt,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDeploymentDTO(out))
}

func (h *Handler) getDeployment(c *gin.Context) {
	out, err := h.uc.GetDeployment.Execute(c.Request.Context(), usecase.GetDeploymentInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDeploymentDTO(out))
}
