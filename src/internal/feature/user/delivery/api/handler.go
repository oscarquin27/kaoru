// Package api expone el perfil del usuario por HTTP.
package api

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/user/application/usecase"
)

// Handler atiende las rutas de user.
type Handler struct {
	uc *usecase.UseCases
	// synced recuerda el perfil ya guardado de cada usuario para no escribir en cada petición.
	synced sync.Map
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/me", h.getMe)
}

func profileInput(p oidc.Principal) usecase.ProfileInput {
	return usecase.ProfileInput{Id: p.Sub, Username: p.Username, DisplayName: p.Name, Email: p.Email}
}

// EnsureProfile guarda el perfil del token la primera vez que se ve en este proceso,
// y de nuevo si cambia. Va después del middleware de OIDC.
func (h *Handler) EnsureProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		in := profileInput(oidc.PrincipalFrom(c))
		if seen, ok := h.synced.Load(in.Id); !ok || seen != in {
			if err := h.uc.SyncProfile.Execute(c.Request.Context(), in); err != nil {
				problem.Write(c, err)
				return
			}
			h.synced.Store(in.Id, in)
		}
		c.Next()
	}
}

type meDTO struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	LogoutURL   string   `json:"logoutUrl"`
	AccountURL  string   `json:"accountUrl,omitempty"`
}

func (h *Handler) getMe(c *gin.Context) {
	p := oidc.PrincipalFrom(c)
	out, err := h.uc.GetMe.Execute(c.Request.Context(), usecase.GetMeInput{Profile: profileInput(p), Roles: p.Roles})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, meDTO{
		ID: string(out.Profile.Id), Username: out.Profile.Username, DisplayName: out.Profile.DisplayName,
		Email: out.Profile.Email, Roles: out.Roles, Permissions: out.Permissions,
		LogoutURL: out.LogoutURL, AccountURL: out.AccountURL,
	})
}
