//go:build integration

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/filestore"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/idgen"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/migrations"
)

// Las pruebas de integración corren contra un Postgres real y limpio:
// tools/with-postgres.sh lo levanta y exporta estas dos variables.
const (
	ownerURLEnv = "KAORU_TEST_OWNER_URL"
	appURLEnv   = "KAORU_TEST_APP_URL"
)

var users = tokenVerifier{
	"manager": {Sub: "sub-manager", Username: "cmora", Name: "C. Mora", Email: "cmora@lab.invalid", Roles: []string{"change-manager", "offline_access"}},
	"qa":      {Sub: "sub-qa", Username: "arojas", Name: "A. Rojas", Roles: []string{"qa"}},
	"ops":     {Sub: "sub-ops", Username: "lpaz", Name: "L. Paz", Roles: []string{"ops"}},
	"dev":     {Sub: "sub-dev", Username: "jdiaz", Name: "J. Díaz", Roles: []string{"dev"}},
	"admin":   {Sub: "sub-admin", Username: "mleon", Name: "M. León", Roles: []string{"qa-admin"}},
	"auditor": {Sub: "sub-auditor", Username: "vsoto", Name: "V. Soto", Roles: []string{"auditor"}},
}

type api struct {
	t       *testing.T
	handler http.Handler
	ownerDB string
}

func newAPI(t *testing.T) *api {
	t.Helper()
	ownerURL, appURL := os.Getenv(ownerURLEnv), os.Getenv(appURLEnv)
	if ownerURL == "" || appURL == "" {
		t.Fatalf("faltan %s y %s: ejecuta las pruebas con tools/with-postgres.sh", ownerURLEnv, appURLEnv)
	}
	ctx := context.Background()
	if _, err := corepg.Migrate(ctx, ownerURL, migrations.FS); err != nil {
		t.Fatal(err)
	}
	pool, err := corepg.NewPool(ctx, appURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	files, err := filestore.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(RouterDeps{
		Config: testConfig(), Database: corepg.NewDatabase(pool), FileStore: files, Metrics: httpserver.NewMetrics(), Verifier: users,
		Logger: newLogger(testWriter{t}, "text"), Clock: clock.System{}, IdGenerator: idgen.UUIDv7{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &api{t: t, handler: router, ownerDB: ownerURL}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(bytes.TrimSpace(p)))
	return len(p), nil
}

// call hace una petición como el usuario dado y decodifica la respuesta en out.
func (a *api) call(user, method, path string, body any, wantStatus int, out any) {
	a.t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			a.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	if user != "" {
		req.Header.Set("Authorization", "Bearer "+user)
	}
	if method != http.MethodGet {
		req.Header.Set("Origin", testOrigin)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		a.t.Fatalf("%s %s como %q: status %d, quiero %d: %s", method, path, user, rec.Code, wantStatus, rec.Body.String())
	}
	if wantStatus >= 400 && rec.Header().Get("Content-Type") != "application/problem+json" {
		a.t.Errorf("%s %s: un error debe ser problem+json", method, path)
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			a.t.Fatalf("%s %s: respuesta ilegible: %v: %s", method, path, err, rec.Body.String())
		}
	}
}

func (a *api) asOwner(sql string) {
	a.t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, a.ownerDB)
	if err != nil {
		a.t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, sql); err != nil {
		a.t.Fatal(err)
	}
}

type changeBody struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Route  string `json:"route"`
	System struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"system"`
	State struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"state"`
	CreatedBy struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"createdBy"`
	WindowStart *string `json:"windowStart"`
}

type problemBody struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Errors []struct {
		Field string `json:"field"`
		Code  string `json:"code"`
	} `json:"errors"`
}

type verification struct {
	Intact         bool   `json:"intact"`
	Entries        int    `json:"entries"`
	FirstBrokenSeq *int64 `json:"firstBrokenSeq"`
}

type auditPage struct {
	Total int `json:"total"`
	Items []struct {
		Seq        int64  `json:"seq"`
		Action     string `json:"action"`
		Field      string `json:"field"`
		OldValue   string `json:"oldValue"`
		NewValue   string `json:"newValue"`
		Reason     string `json:"reason"`
		EntityCode string `json:"entityCode"`
		Actor      struct {
			DisplayName string `json:"displayName"`
		} `json:"actor"`
	} `json:"items"`
}

var changeCode = regexp.MustCompile(`^CHG-\d{4}-\d{4}$`)

func TestSessionAndPermissions(t *testing.T) {
	a := newAPI(t)
	var p problemBody

	a.call("", "GET", "/api/me", nil, 401, &p)
	if p.Code != "NOT_AUTHENTICATED" {
		t.Errorf("sin token: %+v", p)
	}
	a.call("desconocido", "GET", "/api/me", nil, 401, &p)
	if p.Code != "INVALID_TOKEN" {
		t.Errorf("token inválido: %+v", p)
	}
	a.call("manager", "GET", "/api/no-existe", nil, 404, &p)

	var me struct {
		ID          string   `json:"id"`
		DisplayName string   `json:"displayName"`
		Roles       []string `json:"roles"`
		Permissions []string `json:"permissions"`
		LogoutURL   string   `json:"logoutUrl"`
	}
	a.call("manager", "GET", "/api/me", nil, 200, &me)
	if me.ID != "sub-manager" || me.DisplayName != "C. Mora" || me.LogoutURL != "/oauth2/sign_out" {
		t.Errorf("perfil inesperado: %+v", me)
	}
	if !slices.Equal(me.Roles, []string{"change-manager"}) {
		t.Errorf("solo deben quedar los roles de Kaoru: %v", me.Roles)
	}
	if !slices.Contains(me.Permissions, "chg.create") || slices.Contains(me.Permissions, "run.create") {
		t.Errorf("permisos de change-manager inesperados: %v", me.Permissions)
	}
	a.call("auditor", "GET", "/api/me", nil, 200, &me)
	if !slices.Equal(me.Permissions, []string{"audit.read"}) {
		t.Errorf("el auditor solo lee la auditoría: %v", me.Permissions)
	}
}

func TestWorkflowDefaults(t *testing.T) {
	a := newAPI(t)
	var wf struct {
		States      []struct{ Key string } `json:"states"`
		Transitions []struct {
			From, To string
			Rules    []struct{ RuleKey string }
		} `json:"transitions"`
	}
	a.call("qa", "GET", "/api/workflows/requirement", nil, 200, &wf)
	if len(wf.States) != 14 || wf.States[0].Key != "registered" || len(wf.Transitions) != 24 {
		t.Errorf("flujo del REQ: %d estados, %d transiciones", len(wf.States), len(wf.Transitions))
	}
	rules := 0
	for _, tr := range wf.Transitions {
		rules += len(tr.Rules)
	}
	if rules != 6 {
		t.Errorf("reglas de bloqueo del REQ = %d, quiero 6", rules)
	}
	a.call("qa", "GET", "/api/workflows/paquete", nil, 404, nil)
}

func TestChangeLifecycleIsAudited(t *testing.T) {
	a := newAPI(t)
	var p problemBody

	// Crear: solo quien tiene chg.create, y solo desde el origen de la app.
	a.call("qa", "POST", "/api/changes", map[string]any{"title": "x", "system": "SYPAGO"}, 403, &p)
	if p.Code != "PERMISSION_DENIED" {
		t.Errorf("qa no crea CHG: %+v", p)
	}
	a.call("manager", "POST", "/api/changes", map[string]any{"route": "alta", "windowEnd": "2026-12-01T10:00:00Z"}, 422, &p)
	if p.Code != "INVALID_REQUEST" || len(p.Errors) != 1 || p.Errors[0].Field != "system" {
		t.Errorf("sin sistema: %+v", p)
	}
	a.call("manager", "POST", "/api/changes", map[string]any{"system": "SYPAGO", "route": "alta", "windowEnd": "2026-12-01T10:00:00Z"}, 422, &p)
	if len(p.Errors) != 3 {
		t.Errorf("un formulario junta todos sus errores: %+v", p.Errors)
	}

	var created changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": "  Límite diario en pagos  ", "system": "SYPAGO"}, 201, &created)
	if !changeCode.MatchString(created.Code) || created.Title != "Límite diario en pagos" || created.Route != "light" ||
		created.State.Key != "registered" || created.State.Name != "Registrado" || created.System.Name != "SyPago" ||
		created.CreatedBy.DisplayName != "C. Mora" {
		t.Fatalf("CHG creado inesperado: %+v", created)
	}
	path := "/api/changes/" + created.Code

	// Leer no exige permiso.
	var got changeBody
	a.call("ops", "GET", path, nil, 200, &got)
	if got != created {
		t.Errorf("lo leído difiere de lo creado:\n%+v\n%+v", got, created)
	}
	a.call("ops", "GET", "/api/changes/CHG-2026-9999", nil, 404, nil)
	a.call("ops", "GET", "/api/changes/basura", nil, 404, nil)

	// Editar.
	a.call("ops", "PATCH", path, map[string]any{"title": "Otro"}, 403, nil)
	a.call("manager", "PATCH", path, map[string]any{"route": "high", "windowStart": "2026-12-01T08:00:00Z"}, 200, &got)
	if got.Route != "high" || got.WindowStart == nil || *got.WindowStart != "2026-12-01T08:00:00Z" {
		t.Errorf("edición no aplicada: %+v", got)
	}
	got = changeBody{} // la respuesta omite windowStart; sin limpiar quedaría el valor anterior
	a.call("manager", "PATCH", path, map[string]any{"windowStart": nil}, 200, &got)
	if got.WindowStart != nil {
		t.Errorf("null debe quitar la ventana: %+v", got)
	}

	// Transiciones.
	a.call("qa", "POST", path+"/transitions", map[string]any{"to": "approved"}, 403, nil)
	a.call("manager", "POST", path+"/transitions", map[string]any{"to": "closed"}, 409, &p)
	if p.Code != "INVALID_ACTION" || p.Detail != "No se puede pasar de «Registrado» a «Cerrado»." {
		t.Errorf("transición que el flujo no permite: %+v", p)
	}
	a.call("manager", "POST", path+"/transitions", map[string]any{"to": "rejected"}, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "reason" || p.Errors[0].Code != "required" {
		t.Errorf("rechazar exige motivo: %+v", p)
	}
	a.call("manager", "POST", path+"/transitions", map[string]any{"to": "approved"}, 200, &got)
	if got.State.Key != "approved" {
		t.Errorf("no quedó aprobado: %+v", got.State)
	}

	// Lista filtrada.
	var page struct {
		Total int          `json:"total"`
		Items []changeBody `json:"items"`
	}
	a.call("qa", "GET", "/api/changes?state=approved&q="+created.Code, nil, 200, &page)
	if page.Total != 1 || page.Items[0].Code != created.Code {
		t.Errorf("lista filtrada: %+v", page)
	}
	a.call("qa", "GET", "/api/changes?state=closed&q="+created.Code, nil, 200, &page)
	if page.Total != 0 || page.Items == nil {
		t.Errorf("una lista vacía es [] y no null: %+v", page)
	}
	a.call("qa", "GET", "/api/changes?size=500", nil, 422, nil)

	// Auditoría: solo con audit.read, y con todo lo que pasó.
	a.call("ops", "GET", "/api/audit?entityCode="+created.Code, nil, 403, nil)
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+created.Code, nil, 200, &audit)
	var seen []string
	for i := len(audit.Items) - 1; i >= 0; i-- {
		item := audit.Items[i]
		seen = append(seen, item.Action+":"+item.Field)
		if item.Actor.DisplayName != "C. Mora" || item.EntityCode != created.Code {
			t.Errorf("registro inesperado: %+v", item)
		}
	}
	want := []string{"create:", "update:route", "update:windowStart", "update:windowStart", "transition:state"}
	if !slices.Equal(seen, want) {
		t.Errorf("auditoría = %v, quiero %v", seen, want)
	}
	if last := audit.Items[0]; last.OldValue != "registered" || last.NewValue != "approved" {
		t.Errorf("la transición debe guardar el estado anterior y el nuevo: %+v", last)
	}
}

func TestAuditChainSurvivesConcurrencyAndDetectsTampering(t *testing.T) {
	a := newAPI(t)

	const parallel = 24
	codes := make([]string, parallel)
	var wg sync.WaitGroup
	for i := range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var created changeBody
			a.call("manager", "POST", "/api/changes", map[string]any{"title": fmt.Sprintf("Concurrente %d", i), "system": "SIMF"}, 201, &created)
			codes[i] = created.Code
		}()
	}
	wg.Wait()
	slices.Sort(codes)
	if len(slices.Compact(slices.Clone(codes))) != parallel {
		t.Fatalf("los códigos deben ser únicos aun en paralelo: %v", codes)
	}

	var v verification
	a.call("auditor", "GET", "/api/audit/verification", nil, 200, &v)
	if !v.Intact || v.Entries < parallel || v.FirstBrokenSeq != nil {
		t.Fatalf("tras escrituras en paralelo la cadena debe estar íntegra: %+v", v)
	}
	a.call("ops", "GET", "/api/audit/verification", nil, 403, nil)

	// Alguien con acceso de dueño altera un registro: la verificación lo detecta.
	// Al terminar se restaura, porque las demás pruebas comparten la base.
	const target = `WHERE seq = (SELECT min(seq) + 3 FROM audit_entry)`
	a.asOwner(`UPDATE audit_entry SET new_value = coalesce(new_value, '') || '·alterado' ` + target)
	t.Cleanup(func() {
		a.asOwner(`UPDATE audit_entry SET new_value = nullif(replace(new_value, '·alterado', ''), '') ` + target)
		a.call("auditor", "GET", "/api/audit/verification", nil, 200, &v)
		if !v.Intact {
			t.Errorf("al restaurar el registro la cadena debe volver a estar íntegra: %+v", v)
		}
	})
	a.call("auditor", "GET", "/api/audit/verification", nil, 200, &v)
	if v.Intact || v.FirstBrokenSeq == nil {
		t.Fatalf("debía detectar el registro alterado: %+v", v)
	}
}
