package oidc

import (
	"strings"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/logging"
	"kaoru/internal/core/pkg/problem"
)

const principalKey = "oidc_principal"

// Middleware exige un token válido en `Authorization: Bearer` y deja el Principal en el contexto.
func Middleware(v Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			problem.Write(c, axis.New(axis.NotAuthenticated, "Inicia sesión para continuar."))
			return
		}
		principal, err := v.Verify(c.Request.Context(), raw)
		if err != nil {
			problem.Write(c, err)
			return
		}
		c.Set(principalKey, principal)
		fields := logging.FieldsFrom(c.Request.Context())
		fields.UserSub = principal.Sub
		c.Request = c.Request.WithContext(logging.WithFields(c.Request.Context(), fields))
		c.Next()
	}
}

// PrincipalFrom devuelve la identidad que dejó Middleware.
func PrincipalFrom(c *gin.Context) Principal {
	p, _ := c.Get(principalKey)
	principal, _ := p.(Principal)
	return principal
}
