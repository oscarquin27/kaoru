package oidc_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/oidc"
)

const (
	issuer   = "https://kaoru.lab.invalid/realms/kaoru"
	clientID = "kaoru-web"
)

// realm es un emisor de laboratorio: una llave RSA y un JWKS servido por HTTP.
type realm struct {
	key    *rsa.PrivateKey
	server *httptest.Server
}

func newRealm(t *testing.T) *realm {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	r := &realm{key: key}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &key.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig"},
		}})
	}))
	t.Cleanup(r.server.Close)
	return r
}

func sign(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "k1"))
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(claims)
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	token, err := jws.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func validClaims() map[string]any {
	now := time.Now()
	return map[string]any{
		"iss": issuer, "sub": "user-1", "aud": "account", "azp": clientID, "typ": "Bearer",
		"iat": now.Unix(), "nbf": now.Add(-time.Minute).Unix(), "exp": now.Add(5 * time.Minute).Unix(),
		"preferred_username": "arojas", "name": "A. Rojas", "email": "arojas@lab.invalid",
		"realm_access": map[string]any{"roles": []string{"qa", "offline_access"}},
	}
}

func TestVerify(t *testing.T) {
	r := newRealm(t)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	verifier := oidc.NewVerifier(context.Background(), oidc.Config{Issuer: issuer, JWKSURL: r.server.URL, ClientID: clientID})

	with := func(mutate func(c map[string]any)) string {
		c := validClaims()
		mutate(c)
		return sign(t, r.key, c)
	}
	cases := []struct {
		name     string
		token    string
		wantCode axis.Code
	}{
		{"válido", with(func(map[string]any) {}), ""},
		{"vencido", with(func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() }), axis.InvalidToken},
		{"todavía no vigente", with(func(c map[string]any) { c["nbf"] = time.Now().Add(time.Hour).Unix() }), axis.InvalidToken},
		{"otro emisor", with(func(c map[string]any) { c["iss"] = "https://otro.invalid/realms/kaoru" }), axis.InvalidToken},
		{"otro cliente", with(func(c map[string]any) { c["azp"] = "otra-app" }), axis.InvalidToken},
		{"sin azp", with(func(c map[string]any) { delete(c, "azp") }), axis.InvalidToken},
		{"token de refresco", with(func(c map[string]any) { c["typ"] = "Refresh" }), axis.InvalidToken},
		{"ID token", with(func(c map[string]any) { c["typ"] = "ID" }), axis.InvalidToken},
		{"sin sub", with(func(c map[string]any) { delete(c, "sub") }), axis.InvalidToken},
		{"firmado con otra llave", sign(t, otherKey, validClaims()), axis.InvalidToken},
		{"no es un JWT", "no-es-un-token", axis.InvalidToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			principal, err := verifier.Verify(context.Background(), tc.token)
			if tc.wantCode != "" {
				if !axis.IsCode(err, tc.wantCode) {
					t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if principal.Sub != "user-1" || principal.Username != "arojas" || principal.Name != "A. Rojas" ||
				principal.Email != "arojas@lab.invalid" || !slices.Equal(principal.Roles, []string{"qa", "offline_access"}) {
				t.Errorf("identidad inesperada: %+v", principal)
			}
		})
	}
}

func TestVerifyReportsUnavailableWhenKeysCannotBeFetched(t *testing.T) {
	r := newRealm(t)
	token := sign(t, r.key, validClaims())
	r.server.Close()

	verifier := oidc.NewVerifier(context.Background(), oidc.Config{Issuer: issuer, JWKSURL: r.server.URL, ClientID: clientID})
	_, err := verifier.Verify(context.Background(), token)
	if !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Fatalf("sin llaves debe ser SERVICE_UNAVAILABLE (503), no un 401; obtuve %v", err)
	}
}
