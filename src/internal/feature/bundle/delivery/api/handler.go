// Package api expone los PKG por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/bundle/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de bundle.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/packages", h.list)
	r.POST("/api/packages", h.create)
	r.GET("/api/packages/:code", h.get)
	r.PATCH("/api/packages/:code", h.update)
	r.POST("/api/packages/:code/transitions", h.transition)
	r.PUT("/api/packages/:code/builds/:build", h.addBuild)
	r.DELETE("/api/packages/:code/builds/:build", h.removeBuild)
	r.PUT("/api/packages/:code/requirements/:requirement", h.addRequirement)
	r.DELETE("/api/packages/:code/requirements/:requirement", h.removeRequirement)
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

// write responde con el PKG o con el problema.
func write(c *gin.Context, status int, out usecase.PackageOutput, err error) {
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(status, toDTO(out))
}

func (h *Handler) list(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListPackages.Execute(c.Request.Context(), usecase.ListPackagesInput{
		SystemKey: c.Query("system"), StateKey: c.Query("state"), Query: c.Query("q"), Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]packageDTO, len(out.Packages))
	for i, v := range out.Packages {
		items[i] = toDTO(v)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) create(c *gin.Context) {
	var body createDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreatePackage.Execute(c.Request.Context(), usecase.CreatePackageInput{
		Actor: actorOf(c), Title: body.Title, Description: body.Description, SystemKey: body.System,
	})
	write(c, http.StatusCreated, out, err)
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetPackage.Execute(c.Request.Context(), usecase.GetPackageInput{Code: c.Param("code")})
	write(c, http.StatusOK, out, err)
}

func (h *Handler) update(c *gin.Context) {
	var body patchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdatePackage.Execute(c.Request.Context(), usecase.UpdatePackageInput{
		Actor: actorOf(c), Code: c.Param("code"), Edit: body.toEdit(),
	})
	write(c, http.StatusOK, out, err)
}

func (h *Handler) transition(c *gin.Context) {
	var body transitionDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.TransitionPackage.Execute(c.Request.Context(), usecase.TransitionPackageInput{
		Actor: actorOf(c), Code: c.Param("code"), To: body.To, Reason: body.Reason,
	})
	write(c, http.StatusOK, out, err)
}

func (h *Handler) addBuild(c *gin.Context) {
	out, err := h.uc.AddPackageBuild.Execute(c.Request.Context(), usecase.AddPackageBuildInput{
		Actor: actorOf(c), Code: c.Param("code"), BuildCode: c.Param("build"),
	})
	write(c, http.StatusOK, out, err)
}

func (h *Handler) removeBuild(c *gin.Context) {
	out, err := h.uc.RemovePackageBuild.Execute(c.Request.Context(), usecase.RemovePackageBuildInput{
		Actor: actorOf(c), Code: c.Param("code"), BuildCode: c.Param("build"),
	})
	write(c, http.StatusOK, out, err)
}

func (h *Handler) addRequirement(c *gin.Context) {
	out, err := h.uc.AddPackageRequirement.Execute(c.Request.Context(), usecase.AddPackageRequirementInput{
		Actor: actorOf(c), Code: c.Param("code"), RequirementCode: c.Param("requirement"),
	})
	write(c, http.StatusOK, out, err)
}

func (h *Handler) removeRequirement(c *gin.Context) {
	out, err := h.uc.RemovePackageRequirement.Execute(c.Request.Context(), usecase.RemovePackageRequirementInput{
		Actor: actorOf(c), Code: c.Param("code"), RequirementCode: c.Param("requirement"),
	})
	write(c, http.StatusOK, out, err)
}
