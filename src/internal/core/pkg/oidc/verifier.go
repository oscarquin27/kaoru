// Package oidc verifica el token que inyecta el perímetro y expone la identidad.
package oidc

import (
	"context"
	"strings"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"kaoru/internal/core/pkg/axis"
)

// Principal es la identidad verificada de la petición.
type Principal struct {
	Sub      string
	Username string
	Name     string
	Email    string
	Roles    []string
}

// Verifier verifica un token de acceso.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (Principal, error)
}

// Config son los datos del realm.
type Config struct {
	// Issuer es el emisor tal como aparece en el token (la URL pública del realm).
	Issuer string
	// JWKSURL es de donde el backend baja las llaves (la URL interna).
	JWKSURL string
	// ClientID es el cliente de la app; el token debe traerlo en `azp`.
	ClientID string
}

const jwksTimeout = 5 * time.Second

// JWTVerifier verifica firma, emisor, vigencia, cliente y tipo.
type JWTVerifier struct {
	verifier *gooidc.IDTokenVerifier
	clientID string
}

var _ Verifier = (*JWTVerifier)(nil)

// NewVerifier crea el verificador. Las llaves se bajan al primer uso y quedan en caché.
// ctx gobierna las descargas de llaves: debe vivir lo que viva el proceso.
func NewVerifier(ctx context.Context, cfg Config) *JWTVerifier {
	keys := &keySet{inner: gooidc.NewRemoteKeySet(ctx, cfg.JWKSURL)}
	return &JWTVerifier{
		// El `aud` de un token de acceso de Keycloak no es el cliente; se comprueba `azp`.
		verifier: gooidc.NewVerifier(cfg.Issuer, keys, &gooidc.Config{SkipClientIDCheck: true}),
		clientID: cfg.ClientID,
	}
}

type claims struct {
	AuthorizedParty string `json:"azp"`
	Type            string `json:"typ"`
	Username        string `json:"preferred_username"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	RealmAccess     struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// Verify devuelve la identidad del token o un error con código:
// INVALID_TOKEN si el token no vale y SERVICE_UNAVAILABLE si no se pudieron bajar las llaves.
func (v *JWTVerifier) Verify(ctx context.Context, rawToken string) (Principal, error) {
	fetch := &fetchState{}
	token, err := v.verifier.Verify(context.WithValue(ctx, fetchKey{}, fetch), rawToken)
	if err != nil {
		if fetch.failed {
			return Principal{}, axis.Wrap(axis.ServiceUnavailable, "No se pudo comprobar la sesión. Intenta de nuevo.", err)
		}
		return Principal{}, axis.Wrap(axis.InvalidToken, "La sesión no es válida.", err)
	}
	var c claims
	if err := token.Claims(&c); err != nil {
		return Principal{}, axis.Wrap(axis.InvalidToken, "La sesión no es válida.", err)
	}
	if c.AuthorizedParty != v.clientID || c.Type != "Bearer" || token.Subject == "" {
		return Principal{}, axis.New(axis.InvalidToken, "La sesión no es válida.")
	}
	return Principal{
		Sub:      token.Subject,
		Username: c.Username,
		Name:     c.Name,
		Email:    c.Email,
		Roles:    c.RealmAccess.Roles,
	}, nil
}

type fetchKey struct{}

type fetchState struct{ failed bool }

// keySet limita el tiempo de descarga de llaves y anota si falló la descarga,
// porque go-oidc devuelve ese fallo sin envolver.
type keySet struct {
	inner *gooidc.RemoteKeySet
}

func (k *keySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	timed, cancel := context.WithTimeout(ctx, jwksTimeout)
	defer cancel()
	payload, err := k.inner.VerifySignature(timed, jwt)
	if err != nil && (timed.Err() != nil || strings.Contains(err.Error(), "fetching keys")) {
		if state, ok := ctx.Value(fetchKey{}).(*fetchState); ok {
			state.failed = true
		}
	}
	return payload, err
}
