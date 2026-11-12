// Package api expone la selección de regresión de un PKG por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/regression/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de regression.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas. Quitar la confirmación no tiene ruta: lo hace el paquete.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/packages/:code/regression", h.get)
	r.POST("/api/packages/:code/regression/proposal", h.propose)
	r.PUT("/api/packages/:code/regression/items/:testCase", h.adjust)
	r.POST("/api/packages/:code/regression/confirmation", h.confirm)
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func write(c *gin.Context, out usecase.RegressionOutput, err error) {
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetRegression.Execute(c.Request.Context(), usecase.GetRegressionInput{PackageCode: c.Param("code")})
	write(c, out, err)
}

func (h *Handler) propose(c *gin.Context) {
	out, err := h.uc.ProposeRegression.Execute(c.Request.Context(), usecase.ProposeRegressionInput{
		Actor: actorOf(c), PackageCode: c.Param("code"),
	})
	write(c, out, err)
}

func (h *Handler) adjust(c *gin.Context) {
	var body adjustDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.AdjustRegressionItem.Execute(c.Request.Context(), usecase.AdjustRegressionItemInput{
		Actor: actorOf(c), PackageCode: c.Param("code"), TestCaseCode: c.Param("testCase"),
		Included: body.Included, Reason: body.Reason,
	})
	write(c, out, err)
}

func (h *Handler) confirm(c *gin.Context) {
	out, err := h.uc.ConfirmRegression.Execute(c.Request.Context(), usecase.ConfirmRegressionInput{
		Actor: actorOf(c), PackageCode: c.Param("code"),
	})
	write(c, out, err)
}
