// Package api expone el RISK de un REQ por HTTP.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/risk/application/usecase"
	"kaoru/internal/feature/risk/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de risk.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/requirements/:code/risk", h.get)
	r.PUT("/api/requirements/:code/risk", h.assess)
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type riskDTO struct {
	Code            string     `json:"code"`
	Level           string     `json:"level"`
	Justification   string     `json:"justification"`
	Characteristics []string   `json:"characteristics"`
	AssessedBy      userRefDTO `json:"assessedBy"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type assessDTO struct {
	Level           string   `json:"level"`
	Justification   string   `json:"justification"`
	Characteristics []string `json:"characteristics"`
}

func toDTO(s domain.Snapshot) riskDTO {
	dto := riskDTO{
		Code: s.Code.String(), Level: string(s.Level), Justification: s.Justification,
		Characteristics: make([]string, len(s.Characteristics)),
		AssessedBy:      userRefDTO{ID: string(s.AssessedBy), DisplayName: s.AssessedByName},
		UpdatedAt:       s.UpdatedAt,
	}
	for i, c := range s.Characteristics {
		dto.Characteristics[i] = string(c)
	}
	return dto
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetRisk.Execute(c.Request.Context(), usecase.GetRiskInput{RequirementCode: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *Handler) assess(c *gin.Context) {
	var body assessDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	p := oidc.PrincipalFrom(c)
	out, err := h.uc.AssessRisk.Execute(c.Request.Context(), usecase.AssessRiskInput{
		Actor:           shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles},
		RequirementCode: c.Param("code"), Level: body.Level, Justification: body.Justification,
		Characteristics: body.Characteristics,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}
