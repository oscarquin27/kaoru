//go:build integration

package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type buildBody struct {
	Code   string `json:"code"`
	System struct {
		Key string `json:"key"`
	} `json:"system"`
	Version      string `json:"version"`
	Checksum     string `json:"checksum"`
	Requirements []struct {
		Code  string `json:"code"`
		Title string `json:"title"`
	} `json:"requirements"`
	Migrations string `json:"migrations"`
	BuiltBy    struct {
		DisplayName string `json:"displayName"`
	} `json:"builtBy"`
}

type environmentBody struct {
	Code   string `json:"code"`
	System struct {
		Key string `json:"key"`
	} `json:"system"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	OS        string `json:"os"`
	Retired   bool   `json:"retired"`
	Installed *struct {
		Deployment string `json:"deployment"`
		Build      *struct {
			Code     string `json:"code"`
			Version  string `json:"version"`
			Checksum string `json:"checksum"`
		} `json:"build"`
		Package *struct {
			Code  string `json:"code"`
			Title string `json:"title"`
		} `json:"package"`
		DeployedAt time.Time `json:"deployedAt"`
	} `json:"installed"`
}

type deploymentBody struct {
	Code        string `json:"code"`
	Environment struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"environment"`
	Build *struct {
		Code string `json:"code"`
	} `json:"build"`
	Package *struct {
		Code  string `json:"code"`
		Title string `json:"title"`
	} `json:"package"`
	Result     string `json:"result"`
	ExecutedBy struct {
		DisplayName string `json:"displayName"`
	} `json:"executedBy"`
}

type runBody struct {
	Code            string `json:"code"`
	Build           string `json:"build"`
	Environment     string `json:"environment"`
	BuildVersion    string `json:"buildVersion"`
	BuildChecksum   string `json:"buildChecksum"`
	EnvironmentName string `json:"environmentName"`
}

// catalogSeq distingue lo que cada prueba registra en el catálogo: todas comparten la base.
var catalogSeq atomic.Int64

// uniqueChecksum devuelve un SHA-256 que ninguna otra prueba usa.
func (a *api) uniqueChecksum() string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s-%d-%d", a.t.Name(), catalogSeq.Add(1), time.Now().UnixNano()))
	return hex.EncodeToString(sum[:])
}

// uniqueName devuelve un nombre con ese prefijo que ninguna otra prueba usa.
func (a *api) uniqueName(prefix string) string {
	return fmt.Sprintf("%s %d-%d", prefix, time.Now().UnixNano(), catalogSeq.Add(1))
}

// catalog registra por la API un BUILD y un ambiente de prueba de ese sistema y devuelve sus códigos:
// es lo que necesita cualquier prueba que registre un RUN o un EXP.
func (a *api) catalog(system string) (build, environment string) {
	a.t.Helper()
	var b, e codeBody
	a.call("dev", "POST", "/api/builds", map[string]any{"system": system, "version": "2.14.0", "checksum": a.uniqueChecksum()}, 201, &b)
	a.call("qa", "POST", "/api/environments", map[string]any{"system": system, "name": a.uniqueName("QA"), "kind": "baseline", "os": "linux"}, 201, &e)
	return b.Code, e.Code
}

// mustFailAsApp ejecuta una sentencia con el rol de la aplicación y exige que la base la rechace.
func (a *api) mustFailAsApp(sql, what string) {
	a.t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv(appURLEnv))
	if err != nil {
		a.t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, sql); err == nil || !strings.Contains(err.Error(), "permission denied") {
		a.t.Errorf("%s: la base debía negarlo al rol de la aplicación, obtuve %v", what, err)
	}
}

func hasField(p problemBody, field, code string) bool {
	return len(p.Errors) == 1 && p.Errors[0].Field == field && p.Errors[0].Code == code
}

// BUILD: lo registra dev, con su manifiesto; el checksum no se repite y después no cambia.
func TestBuildIsRegisteredOnceAndNeverChanges(t *testing.T) {
	a := newAPI(t)
	var p problemBody

	var req, other requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Límite por BUILD", "system": "SYPAGO"}, 201, &req)
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Firma por BUILD", "system": "SIMF"}, 201, &other)
	checksum := a.uniqueChecksum()
	version := a.uniqueName("3.0.0")
	body := func(extra map[string]any) map[string]any {
		out := map[string]any{"system": "SYPAGO", "version": version, "checksum": strings.ToUpper(checksum),
			"requirements": []string{req.Code}, "migrations": "0042_limite.sql", "dependencies": "libpagos 3.1", "notes": "CI"}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}

	// Permisos: registra dev; QA y change management no.
	a.call("qa", "POST", "/api/builds", body(nil), 403, nil)
	a.call("manager", "POST", "/api/builds", body(nil), 403, nil)
	a.call("", "POST", "/api/builds", body(nil), 401, nil)

	// Validaciones, cada una en su campo.
	a.call("dev", "POST", "/api/builds", map[string]any{"system": "SYPAGO"}, 422, &p)
	if len(p.Errors) != 2 || p.Errors[0].Field != "version" || p.Errors[1].Field != "checksum" {
		t.Errorf("BUILD vacío: %+v", p)
	}
	if a.call("dev", "POST", "/api/builds", body(map[string]any{"system": "OTRO"}), 422, &p); !hasField(p, "system", "invalid") {
		t.Errorf("sistema desconocido: %+v", p)
	}
	if a.call("dev", "POST", "/api/builds", body(map[string]any{"checksum": "abc"}), 422, &p); !hasField(p, "checksum", "invalid") {
		t.Errorf("checksum corto: %+v", p)
	}
	if a.call("dev", "POST", "/api/builds", body(map[string]any{"requirements": []string{"REQ-2026-9999"}}), 422, &p); !hasField(p, "requirements", "invalid") {
		t.Errorf("REQ inexistente: %+v", p)
	}
	if a.call("dev", "POST", "/api/builds", body(map[string]any{"requirements": []string{req.Code, other.Code}}), 422, &p); !hasField(p, "requirements", "invalid") {
		t.Errorf("REQ de otro sistema: %+v", p)
	}

	var build buildBody
	a.call("dev", "POST", "/api/builds", body(nil), 201, &build)
	if !strings.HasPrefix(build.Code, "BUILD-") || build.System.Key != "SYPAGO" || build.Checksum != checksum || build.Version != version ||
		len(build.Requirements) != 1 || build.Requirements[0].Code != req.Code || build.Requirements[0].Title != "Límite por BUILD" ||
		build.Migrations != "0042_limite.sql" || build.BuiltBy.DisplayName != "J. Díaz" {
		t.Fatalf("BUILD inesperado: %+v", build)
	}

	// El mismo artefacto no se registra dos veces, ni con otra versión ni en otro sistema.
	a.call("dev", "POST", "/api/builds", body(map[string]any{"system": "SIMF", "version": "9.9.9", "requirements": []string{}}), 422, &p)
	if !hasField(p, "checksum", "invalid") {
		t.Errorf("checksum repetido: %+v", p)
	}

	// Lecturas: cualquiera con sesión; filtros por sistema, REQ y versión.
	var read buildBody
	a.call("ops", "GET", "/api/builds/"+build.Code, nil, 200, &read)
	if read.Code != build.Code || len(read.Requirements) != 1 {
		t.Errorf("ficha del BUILD: %+v", read)
	}
	a.call("ops", "GET", "/api/builds/BUILD-2026-9999", nil, 404, nil)
	var page struct {
		Total int         `json:"total"`
		Items []buildBody `json:"items"`
	}
	a.call("ops", "GET", "/api/builds?requirement="+req.Code, nil, 200, &page)
	if page.Total != 1 || page.Items[0].Code != build.Code || len(page.Items[0].Requirements) != 1 {
		t.Errorf("BUILD del REQ: %+v", page)
	}
	a.call("ops", "GET", "/api/builds?system=SYPAGO&q="+strings.ReplaceAll(version, " ", "%20"), nil, 200, &page)
	if page.Total != 1 || page.Items[0].Code != build.Code {
		t.Errorf("BUILD por versión: %+v", page)
	}
	a.call("ops", "GET", "/api/builds?system=SIMF&requirement="+req.Code, nil, 200, &page)
	if page.Total != 0 {
		t.Errorf("BUILD de otro sistema: %+v", page)
	}
	// La lista va del más reciente al más antiguo.
	a.catalog("SYPAGO")
	a.call("ops", "GET", "/api/builds?system=SYPAGO&size=2", nil, 200, &page)
	if len(page.Items) != 2 || page.Items[1].Code != build.Code {
		t.Errorf("orden de la lista: %+v", page.Items)
	}

	// La auditoría guarda quién lo registró.
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+build.Code, nil, 200, &audit)
	if audit.Total != 1 || audit.Items[0].Action != "create" || audit.Items[0].Field != "checksum" ||
		audit.Items[0].NewValue != checksum || audit.Items[0].Actor.DisplayName != "J. Díaz" {
		t.Errorf("auditoría del BUILD: %+v", audit)
	}

	// Un BUILD no se edita ni se borra: no hay ruta, y la base se lo niega al rol de la aplicación.
	a.call("dev", "PATCH", "/api/builds/"+build.Code, map[string]any{"version": "x"}, 404, nil)
	a.call("dev", "DELETE", "/api/builds/"+build.Code, nil, 404, nil)
	a.mustFailAsApp("UPDATE build SET version = 'x' WHERE code = '"+build.Code+"'", "editar un BUILD")
	a.mustFailAsApp("DELETE FROM build WHERE code = '"+build.Code+"'", "borrar un BUILD")
	a.mustFailAsApp("DELETE FROM build_requirement", "borrar el manifiesto de un BUILD")
	a.mustFailAsApp("UPDATE build_requirement SET requirement_id = requirement_id", "editar el manifiesto de un BUILD")
}

// ENV y DEP: QA mantiene los ambientes y despliega en los de prueba; en producción despliega change management.
func TestEnvironmentsAndDeployments(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	executedAt := func(ago time.Duration) string { return time.Now().Add(-ago).UTC().Format(time.RFC3339) }

	// Ambientes: permisos, validaciones y nombre único por sistema.
	name := a.uniqueName("QA")
	envBody := map[string]any{"system": "SYPAGO", "name": name, "kind": "baseline", "os": "linux", "notes": "Línea base"}
	a.call("dev", "POST", "/api/environments", envBody, 403, nil)
	a.call("manager", "POST", "/api/environments", envBody, 403, nil)
	a.call("qa", "POST", "/api/environments", map[string]any{"system": "SYPAGO", "kind": "staging"}, 422, &p)
	if len(p.Errors) != 3 || p.Errors[0].Field != "name" || p.Errors[1].Field != "kind" || p.Errors[2].Field != "os" {
		t.Errorf("ENV inválido: %+v", p)
	}
	var env, prod, simf environmentBody
	a.call("qa", "POST", "/api/environments", envBody, 201, &env)
	if !strings.HasPrefix(env.Code, "ENV-") || env.System.Key != "SYPAGO" || env.Kind != "baseline" || env.OS != "linux" || env.Retired || env.Installed != nil {
		t.Fatalf("ENV inesperado: %+v", env)
	}
	if a.call("qa", "POST", "/api/environments", envBody, 422, &p); !hasField(p, "name", "invalid") {
		t.Errorf("nombre repetido en el sistema: %+v", p)
	}
	envBody["system"] = "SIMF" // el mismo nombre vale en otro sistema
	a.call("admin", "POST", "/api/environments", envBody, 201, &simf)
	a.call("qa", "POST", "/api/environments", map[string]any{"system": "SYPAGO", "name": a.uniqueName("Producción"), "kind": "production", "os": "windows"}, 201, &prod)

	// Edición: solo env.manage; el nombre sigue siendo único y cada campo queda auditado.
	a.call("dev", "PATCH", "/api/environments/"+env.Code, map[string]any{"notes": "x"}, 403, nil)
	a.call("qa", "PATCH", "/api/environments/ENV-2026-9999", map[string]any{"notes": "x"}, 404, nil)
	if a.call("qa", "PATCH", "/api/environments/"+prod.Code, map[string]any{"name": name}, 422, &p); !hasField(p, "name", "invalid") {
		t.Errorf("renombrar con un nombre repetido: %+v", p)
	}
	a.call("qa", "PATCH", "/api/environments/"+env.Code, map[string]any{"os": "windows", "notes": "Reinstalado"}, 200, &env)
	if env.OS != "windows" || env.Name != name {
		t.Errorf("edición del ENV: %+v", env)
	}
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+env.Code, nil, 200, &audit)
	if audit.Total != 3 || audit.Items[0].Actor.DisplayName != "A. Rojas" {
		t.Errorf("auditoría del ENV: %+v", audit)
	}

	// Despliegues en un ambiente de prueba: dep.create (QA).
	var buildA, buildB, buildSimf codeBody
	a.call("dev", "POST", "/api/builds", map[string]any{"system": "SYPAGO", "version": "5.0.0", "checksum": a.uniqueChecksum()}, 201, &buildA)
	a.call("dev", "POST", "/api/builds", map[string]any{"system": "SYPAGO", "version": "5.1.0", "checksum": a.uniqueChecksum()}, 201, &buildB)
	a.call("dev", "POST", "/api/builds", map[string]any{"system": "SIMF", "version": "4.2.0", "checksum": a.uniqueChecksum()}, 201, &buildSimf)
	dep := func(environment, build, pkg, result string, ago time.Duration) map[string]any {
		body := map[string]any{"environment": environment, "result": result, "executedAt": executedAt(ago)}
		if build != "" {
			body["build"] = build
		}
		if pkg != "" {
			body["package"] = pkg
		}
		return body
	}
	a.call("dev", "POST", "/api/deployments", dep(env.Code, buildA.Code, "", "success", time.Hour), 403, nil)
	a.call("manager", "POST", "/api/deployments", dep(env.Code, buildA.Code, "", "success", time.Hour), 403, nil)
	if a.call("qa", "POST", "/api/deployments", dep("ENV-2026-9999", buildA.Code, "", "success", time.Hour), 422, &p); !hasField(p, "environment", "invalid") {
		t.Errorf("ambiente inexistente: %+v", p)
	}
	if a.call("qa", "POST", "/api/deployments", dep(env.Code, "", "", "success", time.Hour), 422, &p); !hasField(p, "build", "required") {
		t.Errorf("ni BUILD ni PKG: %+v", p)
	}
	if a.call("qa", "POST", "/api/deployments", dep(env.Code, "BUILD-2026-9999", "", "success", time.Hour), 422, &p); !hasField(p, "build", "invalid") {
		t.Errorf("BUILD inexistente: %+v", p)
	}
	if a.call("qa", "POST", "/api/deployments", dep(env.Code, buildSimf.Code, "", "success", time.Hour), 422, &p); !hasField(p, "build", "invalid") {
		t.Errorf("BUILD de otro sistema: %+v", p)
	}
	if a.call("qa", "POST", "/api/deployments", dep(env.Code, buildA.Code, "", "ok", -time.Hour), 422, &p); len(p.Errors) != 2 {
		t.Errorf("resultado desconocido y fecha futura: %+v", p)
	}

	// Lo instalado: lo que dejó el último despliegue exitoso; un fallido o un revertido posterior no lo cambia.
	var first, second, failed deploymentBody
	a.call("qa", "POST", "/api/deployments", dep(env.Code, buildA.Code, "", "success", 5*time.Hour), 201, &first)
	if !strings.HasPrefix(first.Code, "DEP-") || first.Environment.Code != env.Code || first.Environment.Name != name ||
		first.Build == nil || first.Build.Code != buildA.Code || first.Package != nil || first.ExecutedBy.DisplayName != "A. Rojas" {
		t.Fatalf("DEP inesperado: %+v", first)
	}
	a.call("ops", "GET", "/api/environments/"+env.Code, nil, 200, &env)
	if env.Installed == nil || env.Installed.Deployment != first.Code || env.Installed.Build == nil || env.Installed.Build.Version != "5.0.0" {
		t.Errorf("instalado tras el primer despliegue: %+v", env.Installed)
	}
	a.call("qa", "POST", "/api/deployments", dep(env.Code, buildB.Code, "", "success", 3*time.Hour), 201, &second)
	a.call("qa", "POST", "/api/deployments", dep(env.Code, buildA.Code, "", "failed", 2*time.Hour), 201, &failed)
	a.call("qa", "POST", "/api/deployments", dep(env.Code, buildA.Code, "", "rolled_back", time.Hour), 201, nil)
	a.call("ops", "GET", "/api/environments/"+env.Code, nil, 200, &env)
	if env.Installed == nil || env.Installed.Deployment != second.Code || env.Installed.Build == nil || env.Installed.Build.Code != buildB.Code ||
		env.Installed.Package != nil {
		t.Errorf("instalado tras varios despliegues: %+v", env.Installed)
	}
	var envs struct {
		Total int               `json:"total"`
		Items []environmentBody `json:"items"`
	}
	a.call("ops", "GET", "/api/environments?system=SYPAGO&kind=baseline&size=200", nil, 200, &envs)
	found := false
	for _, e := range envs.Items {
		if e.Kind != "baseline" || e.System.Key != "SYPAGO" {
			t.Errorf("la lista debía filtrar por sistema y tipo: %+v", e)
		}
		if e.Code == env.Code {
			found = e.Installed != nil && e.Installed.Deployment == second.Code
		}
	}
	if !found {
		t.Errorf("la lista debe traer el ambiente con lo instalado: %+v", envs)
	}

	// Producción: solo dep.create_production (change management); QA no.
	a.call("qa", "POST", "/api/deployments", dep(prod.Code, buildB.Code, "", "success", time.Hour), 403, nil)
	a.call("admin", "POST", "/api/deployments", dep(prod.Code, buildB.Code, "", "success", time.Hour), 403, nil)
	var prodDep deploymentBody
	a.call("manager", "POST", "/api/deployments", dep(prod.Code, buildB.Code, "", "success", time.Hour), 201, &prodDep)
	if prodDep.Environment.Kind != "production" || prodDep.ExecutedBy.DisplayName != "C. Mora" {
		t.Errorf("DEP en producción: %+v", prodDep)
	}

	// Despliegue de un PKG: el paquete se inserta como dueño de la base, porque su feature llega después.
	pkgCode := fmt.Sprintf("PKG-2026-%04d", 9000+catalogSeq.Add(1))
	a.asOwner(`INSERT INTO package (id, code, title, system_id, state_id, created_by, created_at, updated_at)
SELECT gen_random_uuid(), '` + pkgCode + `', 'Paquete de prueba', s.id, st.id, 'sub-dev', now(), now()
FROM system_under_test s, workflow_state st
WHERE s.key = 'SYPAGO' AND st.entity_type = 'package' AND st.is_initial`)
	if a.call("qa", "POST", "/api/deployments", dep(env.Code, buildA.Code, pkgCode, "success", time.Minute), 422, &p); !hasField(p, "package", "invalid") {
		t.Errorf("BUILD y PKG a la vez: %+v", p)
	}
	if a.call("qa", "POST", "/api/deployments", dep(env.Code, "", "PKG-2026-9999", "success", time.Minute), 422, &p); !hasField(p, "package", "invalid") {
		t.Errorf("PKG inexistente: %+v", p)
	}
	if a.call("admin", "POST", "/api/deployments", dep(simf.Code, "", pkgCode, "success", time.Minute), 422, &p); !hasField(p, "package", "invalid") {
		t.Errorf("PKG de otro sistema: %+v", p)
	}
	var pkgDep deploymentBody
	a.call("qa", "POST", "/api/deployments", dep(env.Code, "", pkgCode, "success", time.Minute), 201, &pkgDep)
	if pkgDep.Build != nil || pkgDep.Package == nil || pkgDep.Package.Code != pkgCode || pkgDep.Package.Title != "Paquete de prueba" {
		t.Fatalf("DEP de un PKG: %+v", pkgDep)
	}
	var withPkg environmentBody // una variable nueva: lo instalado ya no trae BUILD
	a.call("ops", "GET", "/api/environments/"+env.Code, nil, 200, &withPkg)
	if withPkg.Installed == nil || withPkg.Installed.Deployment != pkgDep.Code || withPkg.Installed.Build != nil ||
		withPkg.Installed.Package == nil || withPkg.Installed.Package.Code != pkgCode {
		t.Errorf("instalado tras desplegar el PKG: %+v", withPkg.Installed)
	}

	// Lecturas de despliegues: ficha y filtros, del más reciente al más antiguo.
	var read deploymentBody
	a.call("ops", "GET", "/api/deployments/"+failed.Code, nil, 200, &read)
	if read.Code != failed.Code || read.Result != "failed" {
		t.Errorf("ficha del DEP: %+v", read)
	}
	a.call("ops", "GET", "/api/deployments/DEP-2026-9999", nil, 404, nil)
	var deps struct {
		Total int              `json:"total"`
		Items []deploymentBody `json:"items"`
	}
	a.call("ops", "GET", "/api/deployments?environment="+env.Code, nil, 200, &deps)
	if deps.Total != 5 || deps.Items[0].Code != pkgDep.Code || deps.Items[4].Code != first.Code {
		t.Errorf("DEP del ambiente: %+v", deps)
	}
	a.call("ops", "GET", "/api/deployments?build="+buildB.Code, nil, 200, &deps)
	if deps.Total != 2 || deps.Items[0].Code != prodDep.Code || deps.Items[1].Code != second.Code {
		t.Errorf("DEP del BUILD: %+v", deps)
	}
	a.call("ops", "GET", "/api/deployments?package="+pkgCode, nil, 200, &deps)
	if deps.Total != 1 || deps.Items[0].Code != pkgDep.Code {
		t.Errorf("DEP del PKG: %+v", deps)
	}
	a.call("auditor", "GET", "/api/audit?entityCode="+prodDep.Code, nil, 200, &audit)
	if audit.Total != 1 || audit.Items[0].Action != "create" || audit.Items[0].NewValue != "success" || audit.Items[0].Actor.DisplayName != "C. Mora" {
		t.Errorf("auditoría del DEP: %+v", audit)
	}

	// Un despliegue no se edita ni se borra: no hay ruta, y la base se lo niega al rol de la aplicación.
	a.call("qa", "PATCH", "/api/deployments/"+first.Code, map[string]any{"result": "failed"}, 404, nil)
	a.call("qa", "DELETE", "/api/deployments/"+first.Code, nil, 404, nil)
	a.mustFailAsApp("UPDATE deployment SET result = 'failed' WHERE code = '"+first.Code+"'", "editar un DEP")
	a.mustFailAsApp("DELETE FROM deployment WHERE code = '"+first.Code+"'", "borrar un DEP")

	// Un ambiente retirado sale de la lista y no admite despliegues; conserva su historia.
	a.call("qa", "PATCH", "/api/environments/"+env.Code, map[string]any{"retired": true}, 200, &env)
	if !env.Retired || env.Installed == nil {
		t.Errorf("ENV retirado: %+v", env)
	}
	if a.call("qa", "POST", "/api/deployments", dep(env.Code, buildA.Code, "", "success", time.Minute), 422, &p); !hasField(p, "environment", "invalid") {
		t.Errorf("despliegue en un ambiente retirado: %+v", p)
	}
	listed := func(query string) bool {
		a.call("ops", "GET", "/api/environments?system=SYPAGO&size=200"+query, nil, 200, &envs)
		for _, e := range envs.Items {
			if e.Code == env.Code {
				return true
			}
		}
		return false
	}
	if listed("") || !listed("&includeRetired=true") {
		t.Error("un ambiente retirado solo sale con includeRetired")
	}
}

// RUN y EXP se registran sobre un BUILD y un ENV del catálogo, del sistema del TC o del REQ.
func TestRunsAndExplorationsUseTheCatalog(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	executedAt := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	var req requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "RUN sobre el catálogo", "system": "SYPAGO"}, 201, &req)
	var uc, tc codeBody
	a.call("qa", "POST", "/api/requirements/"+req.Code+"/functional-cases", map[string]any{"title": "Caso"}, 201, &uc)
	a.call("qa", "POST", "/api/test-cases", map[string]any{
		"title": "TC del catálogo", "system": "SYPAGO", "steps": "1", "expectedResult": "ok", "functionalCases": []string{uc.Code},
	}, 201, &tc)

	checksum := a.uniqueChecksum()
	envName := a.uniqueName("QA")
	var build, simfBuild codeBody
	var env, prod, retired, simfEnv environmentBody
	a.call("dev", "POST", "/api/builds", map[string]any{"system": "SYPAGO", "version": "7.1.0", "checksum": checksum}, 201, &build)
	a.call("dev", "POST", "/api/builds", map[string]any{"system": "SIMF", "version": "4.2.0", "checksum": a.uniqueChecksum()}, 201, &simfBuild)
	environment := func(system, name, kind string, out *environmentBody) {
		t.Helper()
		a.call("qa", "POST", "/api/environments", map[string]any{"system": system, "name": name, "kind": kind, "os": "linux"}, 201, out)
	}
	environment("SYPAGO", envName, "clean", &env)
	environment("SYPAGO", a.uniqueName("Producción"), "production", &prod)
	environment("SYPAGO", a.uniqueName("Viejo"), "offline", &retired)
	environment("SIMF", a.uniqueName("QA SIMF"), "baseline", &simfEnv)
	a.call("qa", "PATCH", "/api/environments/"+retired.Code, map[string]any{"retired": true}, 200, nil)

	run := func(build, environment string) map[string]any {
		body := map[string]any{"testCase": tc.Code, "result": "passed", "executedAt": executedAt}
		if build != "" {
			body["build"] = build
		}
		if environment != "" {
			body["environment"] = environment
		}
		return body
	}
	a.call("qa", "POST", "/api/runs", run("", ""), 422, &p)
	if len(p.Errors) != 2 || p.Errors[0].Field != "build" || p.Errors[0].Code != "required" ||
		p.Errors[1].Field != "environment" || p.Errors[1].Code != "required" {
		t.Errorf("RUN sin BUILD ni ENV: %+v", p)
	}
	// La versión y el checksum ya no entran: se copian del BUILD.
	legacy := map[string]any{"testCase": tc.Code, "buildVersion": "7.1.0", "buildChecksum": checksum, "environmentName": envName,
		"result": "passed", "executedAt": executedAt}
	if a.call("qa", "POST", "/api/runs", legacy, 422, &p); len(p.Errors) != 2 {
		t.Errorf("RUN con los campos anteriores: %+v", p)
	}
	if a.call("qa", "POST", "/api/runs", run("BUILD-2026-9999", env.Code), 422, &p); !hasField(p, "build", "invalid") {
		t.Errorf("BUILD inexistente: %+v", p)
	}
	if a.call("qa", "POST", "/api/runs", run(build.Code, "ENV-2026-9999"), 422, &p); !hasField(p, "environment", "invalid") {
		t.Errorf("ENV inexistente: %+v", p)
	}
	if a.call("qa", "POST", "/api/runs", run(simfBuild.Code, env.Code), 422, &p); !hasField(p, "build", "invalid") {
		t.Errorf("BUILD de otro sistema: %+v", p)
	}
	if a.call("qa", "POST", "/api/runs", run(build.Code, simfEnv.Code), 422, &p); !hasField(p, "environment", "invalid") {
		t.Errorf("ENV de otro sistema: %+v", p)
	}
	if a.call("qa", "POST", "/api/runs", run(build.Code, retired.Code), 422, &p); !hasField(p, "environment", "invalid") {
		t.Errorf("ENV retirado: %+v", p)
	}
	// QA no ejecuta en producción (regla de negocio 6).
	if a.call("qa", "POST", "/api/runs", run(build.Code, prod.Code), 422, &p); !hasField(p, "environment", "invalid") {
		t.Errorf("RUN en producción: %+v", p)
	}

	var created, read runBody
	a.call("qa", "POST", "/api/runs", run(build.Code, env.Code), 201, &created)
	want := runBody{Code: created.Code, Build: build.Code, Environment: env.Code, BuildVersion: "7.1.0", BuildChecksum: checksum, EnvironmentName: envName}
	if created != want {
		t.Fatalf("RUN = %+v, quiero %+v", created, want)
	}
	// La evidencia no cambia aunque el ambiente se renombre o se retire.
	a.call("qa", "PATCH", "/api/environments/"+env.Code, map[string]any{"name": a.uniqueName("Renombrado")}, 200, nil)
	a.call("ops", "GET", "/api/runs/"+created.Code, nil, 200, &read)
	if read != want {
		t.Errorf("RUN leído = %+v, quiero %+v", read, want)
	}
	var runs struct {
		Total int       `json:"total"`
		Items []runBody `json:"items"`
	}
	a.call("ops", "GET", "/api/runs?testCase="+tc.Code+"&buildVersion=7.1.0", nil, 200, &runs)
	if runs.Total != 1 || runs.Items[0] != want {
		t.Errorf("lista de RUN: %+v", runs)
	}

	// Un RUN anterior al catálogo (sin BUILD ni ENV) se sigue leyendo, sin esos dos campos.
	a.asOwner(`INSERT INTO test_run (id, code, test_case_id, build_version, build_checksum, environment_name, result, executed_by, executed_at, created_at)
SELECT gen_random_uuid(), 'RUN-2025-` + fmt.Sprint(900000+catalogSeq.Add(1)) + `', id, '1.0.0', repeat('0', 64), 'QA antiguo', 'passed', 'sub-qa', now() - interval '400 days', now()
FROM test_case WHERE code = '` + tc.Code + `'`)
	var old struct {
		Total int              `json:"total"`
		Items []map[string]any `json:"items"`
	}
	a.call("ops", "GET", "/api/runs?testCase="+tc.Code+"&buildVersion=1.0.0", nil, 200, &old)
	if old.Total != 1 || old.Items[0]["environmentName"] != "QA antiguo" {
		t.Fatalf("RUN anterior al catálogo: %+v", old)
	}
	if _, ok := old.Items[0]["build"]; ok {
		t.Errorf("un RUN anterior al catálogo no trae BUILD: %+v", old.Items[0])
	}
	if _, ok := old.Items[0]["environment"]; ok {
		t.Errorf("un RUN anterior al catálogo no trae ENV: %+v", old.Items[0])
	}

	// EXP: mismas reglas de catálogo, pero sí se puede explorar en producción.
	exp := func(build, environment string) map[string]any {
		return map[string]any{"requirement": req.Code, "build": build, "environment": environment, "executedAt": executedAt}
	}
	a.call("qa", "POST", "/api/explorations", map[string]any{"requirement": req.Code, "executedAt": executedAt}, 422, &p)
	if len(p.Errors) != 2 || p.Errors[0].Field != "build" || p.Errors[1].Field != "environment" {
		t.Errorf("EXP sin BUILD ni ENV: %+v", p)
	}
	a.call("qa", "POST", "/api/explorations", exp(simfBuild.Code, simfEnv.Code), 422, &p)
	if len(p.Errors) != 2 || p.Errors[0].Field != "build" || p.Errors[1].Field != "environment" {
		t.Errorf("EXP sobre otro sistema: %+v", p)
	}
	if a.call("qa", "POST", "/api/explorations", exp(build.Code, retired.Code), 422, &p); !hasField(p, "environment", "invalid") {
		t.Errorf("EXP en un ambiente retirado: %+v", p)
	}
	var exploration struct {
		Code            string `json:"code"`
		Build           string `json:"build"`
		Environment     string `json:"environment"`
		BuildVersion    string `json:"buildVersion"`
		EnvironmentName string `json:"environmentName"`
	}
	a.call("qa", "POST", "/api/explorations", exp(build.Code, prod.Code), 201, &exploration)
	if exploration.Build != build.Code || exploration.Environment != prod.Code || exploration.BuildVersion != "7.1.0" ||
		!strings.HasPrefix(exploration.EnvironmentName, "Producción") {
		t.Errorf("EXP = %+v", exploration)
	}
	a.call("qa", "PATCH", "/api/explorations/"+exploration.Code, map[string]any{"findings": "Nada"}, 200, &exploration)
	if exploration.Build != build.Code || exploration.Environment != prod.Code {
		t.Errorf("EXP editado = %+v", exploration)
	}
}
