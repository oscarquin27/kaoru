package bootstrap

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/filestore"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/oidc"
	corepg "kaoru/internal/core/pkg/postgres"
)

// tokenVerifier es un verificador de laboratorio: el token es el nombre de un usuario conocido.
type tokenVerifier map[string]oidc.Principal

func (v tokenVerifier) Verify(_ context.Context, raw string) (oidc.Principal, error) {
	p, ok := v[raw]
	if !ok {
		return oidc.Principal{}, axis.New(axis.InvalidToken, "La sesión no es válida.")
	}
	return p, nil
}

const testOrigin = "https://kaoru.lab.invalid"

func testConfig() Config {
	return Config{
		HTTP:      HTTPConfig{Addr: ":0", InternalAddr: ":0", PublicOrigin: testOrigin, MaxBodyBytes: 1 << 20},
		Documents: DocumentsConfig{MaxBytes: 2 << 20, AllowedExtensions: []string{".pdf", ".png", ".md", ".txt", ".log", ".xlsx"}},
		OIDC:      OIDCConfig{LogoutURL: "/oauth2/sign_out", AccountURL: "https://kaoru.lab.invalid/account"},
		Log:       LogConfig{Format: "text"},
	}
}

var ginParam = regexp.MustCompile(`:(\w+)`)

// Las rutas registradas y las del contrato son las mismas, con el mismo método (reglas §A.16.1).
func TestRegisteredRoutesAreInTheContract(t *testing.T) {
	raw, err := os.ReadFile("../../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}

	files, err := filestore.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(RouterDeps{
		Config: testConfig(), Database: corepg.NewDatabase(nil), FileStore: files, Verifier: tokenVerifier{},
		Logger: newLogger(os.Stderr, "text"), Clock: clock.System{}, IdGenerator: idgen.UUIDv7{},
	})
	if err != nil {
		t.Fatal(err)
	}
	routes := router.Routes()
	if len(routes) == 0 {
		t.Fatal("el router no registró rutas")
	}
	for _, route := range routes {
		path := ginParam.ReplaceAllString(route.Path, "{$1}")
		if _, ok := contract.Paths[path][strings.ToLower(route.Method)]; !ok {
			t.Errorf("%s %s está registrada pero no figura en api/openapi.yaml", route.Method, path)
		}
	}

	operations := 0
	for _, methods := range contract.Paths {
		for method := range methods {
			if method != "parameters" {
				operations++
			}
		}
	}
	// El contrato del MVP está completo: una operación que se agregue al contrato debe implementarse.
	if len(routes) != operations {
		t.Errorf("implementadas %d de %d operaciones del contrato", len(routes), operations)
	}
}

func TestParseConfig(t *testing.T) {
	const valid = `{
		"http": {"addr": ":8080", "internalAddr": ":9090", "publicOrigin": "${KAORU_PUBLIC_URL}", "maxBodyBytes": 1048576},
		"postgres": {"ownerUrl": "postgres://o:${PW}@db/kaoru", "appUrl": "postgres://a:${PW}@db/kaoru"},
		"oidc": {"issuer": "https://x/realms/kaoru", "jwksUrl": "http://kc/certs", "clientId": "kaoru-web", "logoutUrl": "/oauth2/sign_out"},
		"documents": {"dir": "/var/lib/kaoru/documents", "maxBytes": 26214400, "allowedExtensions": [".pdf"]},
		"log": {"format": "json"}
	}`
	env := func(vars map[string]string) func(string) (string, bool) {
		return func(name string) (string, bool) { v, ok := vars[name]; return v, ok }
	}
	full := map[string]string{"KAORU_PUBLIC_URL": "https://kaoru.lab.invalid", "PW": `cl"ave\rara`}

	cfg, err := ParseConfig([]byte(valid), env(full))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.PublicOrigin != "https://kaoru.lab.invalid" || cfg.Postgres.AppURL != `postgres://a:cl"ave\rara@db/kaoru` {
		t.Errorf("no reemplazó las variables: %+v", cfg)
	}

	cases := []struct {
		name string
		raw  string
		vars map[string]string
		want string
	}{
		{"falta una variable", valid, map[string]string{"PW": "x"}, "KAORU_PUBLIC_URL"},
		{"variable vacía", valid, map[string]string{"KAORU_PUBLIC_URL": "", "PW": "x"}, "KAORU_PUBLIC_URL"},
		{"JSON ilegible", `{`, full, "ilegible"},
		{"falta una sección", `{"log": {"format": "json"}, "http": {"maxBodyBytes": 1}}`, full, "postgres.appUrl"},
		{"sin tipos de adjunto permitidos", strings.Replace(valid, `[".pdf"]`, `[]`, 1), full, "allowedExtensions"},
		{"formato de log desconocido", strings.Replace(valid, `"json"`, `"xml"`, 1), full, "log.format"},
		{"límite de cuerpo en cero", strings.Replace(valid, "1048576", "0", 1), full, "maxBodyBytes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(tc.raw), env(tc.vars))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("esperaba un error que nombre %q, obtuve %v", tc.want, err)
			}
		})
	}
}
