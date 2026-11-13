//go:build integration

package bootstrap

import (
	"fmt"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type stateBody struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type missingBody struct {
	To         stateBody `json:"to"`
	Conditions []struct {
		RuleKey string   `json:"ruleKey"`
		Message string   `json:"message"`
		Blocks  bool     `json:"blocks"`
		Related []string `json:"related"`
	} `json:"conditions"`
}

type packageBody struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	System      struct {
		Key string `json:"key"`
	} `json:"system"`
	State  stateBody `json:"state"`
	Builds []struct {
		Code     string `json:"code"`
		Version  string `json:"version"`
		Checksum string `json:"checksum"`
	} `json:"builds"`
	Requirements []struct {
		Code  string    `json:"code"`
		Title string    `json:"title"`
		State stateBody `json:"state"`
	} `json:"requirements"`
	Missing   []missingBody `json:"missing"`
	CreatedBy struct {
		DisplayName string `json:"displayName"`
	} `json:"createdBy"`
}

// rules devuelve, para la transición a ese estado, las reglas que no se cumplen, por clave;
// nil si la transición no sale del estado actual.
func (p packageBody) rules(to string) []string {
	for _, m := range p.Missing {
		if m.To.Key == to {
			keys := []string{}
			for _, c := range m.Conditions {
				keys = append(keys, c.RuleKey)
			}
			slices.Sort(keys)
			return keys
		}
	}
	return nil
}

type regressionItem struct {
	TestCase struct {
		Code  string `json:"code"`
		Title string `json:"title"`
	} `json:"testCase"`
	IsCritical bool   `json:"isCritical"`
	Origin     string `json:"origin"`
	Proposed   bool   `json:"proposed"`
	Included   bool   `json:"included"`
	Reason     string `json:"reason"`
	DecidedBy  *struct {
		DisplayName string `json:"displayName"`
	} `json:"decidedBy"`
	DecidedAt  *time.Time `json:"decidedAt"`
	Components []struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"components"`
	LastRun *struct {
		Code   string `json:"code"`
		Result string `json:"result"`
		Build  string `json:"build"`
	} `json:"lastRun"`
}

type regressionBody struct {
	Package    string `json:"package"`
	ProposedBy struct {
		DisplayName string `json:"displayName"`
	} `json:"proposedBy"`
	ConfirmedBy *struct {
		DisplayName string `json:"displayName"`
	} `json:"confirmedBy"`
	ConfirmedAt *time.Time `json:"confirmedAt"`
	Summary     struct {
		Proposed int `json:"proposed"`
		Included int `json:"included"`
		Added    int `json:"added"`
		Removed  int `json:"removed"`
		Passed   int `json:"passed"`
		Failed   int `json:"failed"`
		Pending  int `json:"pending"`
	} `json:"summary"`
	Items []regressionItem `json:"items"`
}

// item devuelve el TC de la selección con ese código, o nil si no está.
func (r regressionBody) item(code string) *regressionItem {
	for i := range r.Items {
		if r.Items[i].TestCase.Code == code {
			return &r.Items[i]
		}
	}
	return nil
}

// El recorrido completo de un PKG: se arma con BUILD y REQ, se propone y se ajusta su regresión por impacto,
// y avanza hasta «Verificado» solo cuando cada regla de bloqueo se cumple.
func TestPackageLifecycleWithRegressionByImpact(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	tag := fmt.Sprintf("pkg%d", time.Now().UnixNano())
	oldBuild, environment := a.catalog("SYPAGO")
	ago := func(d time.Duration) string { return time.Now().Add(-d).UTC().Format(time.RFC3339) }

	// Componentes: uno que el REQ toca y otro que no.
	component := func(system, name string) string {
		t.Helper()
		var out codeBody
		a.call("dev", "POST", "/api/components", map[string]any{"system": system, "kind": "microservice", "name": tag + "-" + name}, 201, &out)
		return out.Code
	}
	touched, untouched := component("SYPAGO", "api-pagos"), component("SYPAGO", "api-reportes")

	// Un REQ de SyPago que toca api-pagos, llevado hasta «En pruebas».
	var req requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Límite " + tag, "system": "SYPAGO"}, 201, &req)
	reqPath := "/api/requirements/" + req.Code
	moveReq := func(user, to string) {
		t.Helper()
		a.call(user, "POST", reqPath+"/transitions", map[string]any{"to": to}, 200, nil)
	}
	a.call("dev", "PUT", reqPath+"/components", map[string]any{"components": []string{touched}}, 200, nil)
	a.call("qa", "PUT", reqPath+"/risk", map[string]any{"level": "low", "justification": "Módulo estable"}, 200, nil)
	moveReq("qa", "risk_assessed")
	moveReq("qa", "test_design")
	moveReq("dev", "in_development")
	var contract entryContract
	a.call("dev", "GET", reqPath+"/entry-contract", nil, 200, &contract)
	for _, item := range contract.Items {
		a.call("dev", "PUT", reqPath+"/entry-contract/"+item.Key, map[string]any{}, 200, nil)
	}
	moveReq("dev", "delivered_to_qa")
	moveReq("qa", "in_testing")

	// Sus TC: por impacto, por impacto y crítico, solo crítico, sin impacto y uno retirado que sí toca el componente.
	var uc codeBody
	a.call("qa", "POST", reqPath+"/functional-cases", map[string]any{"title": "Pagar con límite"}, 201, &uc)
	testCase := func(title string, critical bool, components ...string) string {
		t.Helper()
		var out codeBody
		a.call("qa", "POST", "/api/test-cases", map[string]any{
			"title": title + " " + tag, "system": "SYPAGO", "steps": "1. Pagar", "expectedResult": "ok",
			"functionalCases": []string{uc.Code}, "isCritical": critical,
		}, 201, &out)
		if len(components) > 0 {
			a.call("qa", "PUT", "/api/test-cases/"+out.Code+"/components", map[string]any{"components": components}, 200, nil)
		}
		return out.Code
	}
	tcImpact := testCase("Rechaza sobre el límite", false, touched, untouched)
	tcBoth := testCase("Acepta bajo el límite", true, touched)
	tcCritical := testCase("Inicia sesión", true)
	tcOther := testCase("Exporta el reporte", false, untouched)
	tcRetired := testCase("Caso obsoleto", false, touched)
	a.call("qa", "PATCH", "/api/test-cases/"+tcRetired, map[string]any{"retired": true}, 200, nil)
	var simfReq requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Mensajería " + tag, "system": "SIMF"}, 201, &simfReq)
	var simfUC, simfTC codeBody
	a.call("qa", "POST", "/api/requirements/"+simfReq.Code+"/functional-cases", map[string]any{"title": "Enviar"}, 201, &simfUC)
	a.call("qa", "POST", "/api/test-cases", map[string]any{
		"title": "MT103 " + tag, "system": "SIMF", "steps": "1", "expectedResult": "ok", "functionalCases": []string{simfUC.Code},
	}, 201, &simfTC)

	// El REQ se aprueba con sus RUN sobre un BUILD anterior, que no será del PKG.
	run := func(tc, build, result string, executedAgo time.Duration) string {
		t.Helper()
		var out codeBody
		a.call("qa", "POST", "/api/runs", map[string]any{
			"testCase": tc, "build": build, "environment": environment, "result": result, "executedAt": ago(executedAgo),
		}, 201, &out)
		return out.Code
	}
	for _, tc := range []string{tcImpact, tcBoth, tcCritical, tcOther} {
		run(tc, oldBuild, "passed", 5*time.Hour)
	}
	moveReq("qa", "approved_qa")

	// Un segundo REQ que no está aprobado, y los BUILD del catálogo.
	var notApproved requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Sin aprobar " + tag, "system": "SYPAGO"}, 201, &notApproved)
	registerBuild := func(system, version string) string {
		t.Helper()
		var out codeBody
		a.call("dev", "POST", "/api/builds", map[string]any{"system": system, "version": version, "checksum": a.uniqueChecksum()}, 201, &out)
		return out.Code
	}
	build, secondBuild, simfBuild := registerBuild("SYPAGO", "3.0.0"), registerBuild("SYPAGO", "3.0.1"), registerBuild("SIMF", "4.2.0")

	// --- Crear el PKG: pkg.build (dev). ---
	title := "Paquete " + tag
	a.call("qa", "POST", "/api/packages", map[string]any{"title": title, "system": "SYPAGO"}, 403, &p)
	if p.Code != "PERMISSION_DENIED" {
		t.Errorf("QA no arma paquetes: %+v", p)
	}
	a.call("", "POST", "/api/packages", map[string]any{"title": title, "system": "SYPAGO"}, 401, nil)
	if a.call("dev", "POST", "/api/packages", map[string]any{"system": "SYPAGO"}, 422, &p); !hasField(p, "title", "required") {
		t.Errorf("PKG sin título: %+v", p)
	}
	if a.call("dev", "POST", "/api/packages", map[string]any{"title": title, "system": "OTRO"}, 422, &p); !hasField(p, "system", "invalid") {
		t.Errorf("PKG de un sistema desconocido: %+v", p)
	}
	var pkg packageBody
	pkg = packageBody{}
	a.call("dev", "POST", "/api/packages", map[string]any{"title": "  " + title + "  ", "description": "Primera entrega", "system": "SYPAGO"}, 201, &pkg)
	if !strings.HasPrefix(pkg.Code, "PKG-") || pkg.Title != title || pkg.System.Key != "SYPAGO" || pkg.State.Key != "in_assembly" ||
		pkg.State.Name != "En armado" || pkg.Builds == nil || len(pkg.Builds) != 0 || pkg.Requirements == nil || len(pkg.Requirements) != 0 ||
		pkg.CreatedBy.DisplayName != "J. Díaz" {
		t.Fatalf("PKG creado: %+v", pkg)
	}
	if got := pkg.rules("in_testing"); !slices.Equal(got, []string{"package_has_content", "regression_confirmed"}) || len(pkg.Missing) != 1 {
		t.Errorf("qué falta en un PKG vacío: %+v", pkg.Missing)
	}
	path := "/api/packages/" + pkg.Code
	move := func(user, to, reason string, wantStatus int) problemBody {
		t.Helper()
		var out problemBody
		a.call(user, "POST", path+"/transitions", map[string]any{"to": to, "reason": reason}, wantStatus, &out)
		return out
	}
	read := func() packageBody {
		t.Helper()
		var out packageBody
		a.call("ops", "GET", path, nil, 200, &out)
		return out
	}
	a.call("ops", "GET", "/api/packages/PKG-2026-9999", nil, 404, nil)
	a.call("ops", "GET", "/api/packages/basura", nil, 404, nil)

	// Editar: título y descripción; el sistema no cambia.
	a.call("qa", "PATCH", path, map[string]any{"title": "x"}, 403, nil)
	if a.call("dev", "PATCH", path, map[string]any{"title": " "}, 422, &p); !hasField(p, "title", "required") {
		t.Errorf("título vacío: %+v", p)
	}
	pkg = packageBody{}
	a.call("dev", "PATCH", path, map[string]any{"description": "Límites de noviembre", "system": "SIMF"}, 200, &pkg)
	if pkg.Description != "Límites de noviembre" || pkg.Title != title || pkg.System.Key != "SYPAGO" {
		t.Errorf("PKG editado: %+v", pkg)
	}

	// --- Contenido: BUILD. ---
	a.call("qa", "PUT", path+"/builds/"+build, nil, 403, nil)
	a.call("dev", "PUT", path+"/builds/BUILD-2026-9999", nil, 404, nil)
	a.call("dev", "PUT", "/api/packages/PKG-2026-9999/builds/"+build, nil, 404, nil)
	if a.call("dev", "PUT", path+"/builds/"+simfBuild, nil, 409, &p); p.Code != "INVALID_ACTION" || !strings.Contains(p.Detail, "otro sistema") {
		t.Errorf("BUILD de otro sistema: %+v", p)
	}
	pkg = packageBody{}
	a.call("dev", "PUT", path+"/builds/"+build, nil, 200, &pkg)
	pkg = packageBody{}
	a.call("dev", "PUT", path+"/builds/"+build, nil, 200, &pkg) // repetir no cambia nada
	if len(pkg.Builds) != 1 || pkg.Builds[0].Code != build || pkg.Builds[0].Version != "3.0.0" || len(pkg.Builds[0].Checksum) != 64 {
		t.Fatalf("BUILD del PKG: %+v", pkg.Builds)
	}
	if got := pkg.rules("in_testing"); !slices.Equal(got, []string{"package_has_content", "regression_confirmed"}) {
		t.Errorf("con BUILD pero sin REQ todavía falta contenido: %v", got)
	}

	// --- Contenido: REQ. Entra «Aprobado QA» y pasa a «En paquete»; exige también ese permiso. ---
	a.call("qa", "PUT", path+"/requirements/"+req.Code, nil, 403, nil)
	a.call("dev", "PUT", path+"/requirements/REQ-2026-9999", nil, 404, nil)
	if a.call("dev", "PUT", path+"/requirements/"+notApproved.Code, nil, 409, &p); p.Code != "INVALID_ACTION" || !strings.Contains(p.Detail, "Aprobado QA") {
		t.Errorf("REQ sin aprobar: %+v", p)
	}
	if a.call("dev", "PUT", path+"/requirements/"+simfReq.Code, nil, 409, &p); !strings.Contains(p.Detail, "otro sistema") {
		t.Errorf("REQ de otro sistema: %+v", p)
	}
	// ops recibe pkg.build solo durante esta prueba: arma el PKG pero no puede pasar el REQ a «En paquete».
	a.asOwner(`INSERT INTO role_permission (role, permission_key) VALUES ('ops', 'pkg.build')`)
	t.Cleanup(func() { a.asOwner(`DELETE FROM role_permission WHERE role = 'ops' AND permission_key = 'pkg.build'`) })
	a.call("ops", "PUT", path+"/requirements/"+req.Code, nil, 403, &p)
	if pkg = read(); len(pkg.Requirements) != 0 {
		t.Fatalf("sin el permiso de la transición el REQ no entra: %+v", pkg.Requirements)
	}
	a.call("ops", "GET", reqPath, nil, 200, &req)
	if req.State.Key != "approved_qa" {
		t.Fatalf("y su estado no cambia: %+v", req.State)
	}
	pkg = packageBody{}
	a.call("dev", "PUT", path+"/requirements/"+req.Code, nil, 200, &pkg)
	if len(pkg.Requirements) != 1 || pkg.Requirements[0].Code != req.Code || pkg.Requirements[0].Title != "Límite "+tag ||
		pkg.Requirements[0].State.Key != "in_package" || pkg.Requirements[0].State.Name != "En paquete" {
		t.Fatalf("REQ del PKG: %+v", pkg.Requirements)
	}
	a.call("ops", "GET", reqPath, nil, 200, &req)
	if req.State.Key != "in_package" {
		t.Errorf("el REQ debía pasar a «En paquete»: %+v", req.State)
	}
	pkg = packageBody{}
	a.call("ops", "PUT", path+"/requirements/"+req.Code, nil, 200, &pkg) // ya está, y ya «En paquete»: no cambia nada
	if len(pkg.Requirements) != 1 {
		t.Errorf("agregar dos veces no duplica: %+v", pkg.Requirements)
	}
	if got := pkg.rules("in_testing"); !slices.Equal(got, []string{"regression_confirmed"}) {
		t.Errorf("con contenido solo falta la regresión: %v", got)
	}

	// --- A pruebas: no sin regresión confirmada. ---
	move("dev", "in_testing", "", 403)
	if p = move("qa", "in_testing", "", 409); p.Code != "INVALID_ACTION" ||
		!strings.Contains(p.Detail, "No se puede pasar a «En pruebas». Falta proponer y confirmar la regresión.") {
		t.Errorf("a pruebas sin regresión: %+v", p)
	}

	// --- Regresión: proponer por impacto y críticos. ---
	regression := path + "/regression"
	a.call("ops", "GET", regression, nil, 404, &p)
	if p.Code != "NOT_FOUND_RESOURCE" {
		t.Errorf("regresión sin proponer: %+v", p)
	}
	a.call("ops", "GET", "/api/packages/PKG-2026-9999/regression", nil, 404, nil)
	a.call("dev", "POST", regression+"/proposal", nil, 403, nil)
	a.call("qa", "POST", regression+"/confirmation", nil, 404, nil)
	a.call("qa", "PUT", regression+"/items/"+tcImpact, map[string]any{"included": false, "reason": "x"}, 404, nil)
	var reg regressionBody
	reg = regressionBody{}
	a.call("qa", "POST", regression+"/proposal", nil, 200, &reg)
	if reg.Package != pkg.Code || reg.ProposedBy.DisplayName != "A. Rojas" || reg.ConfirmedBy != nil || reg.ConfirmedAt != nil {
		t.Errorf("cabecera de la regresión: %+v", reg)
	}
	for code, want := range map[string]string{tcImpact: "impact", tcBoth: "impact", tcCritical: "critical"} {
		item := reg.item(code)
		if item == nil || item.Origin != want || !item.Proposed || !item.Included || item.Reason != "" || item.DecidedBy != nil || item.LastRun != nil {
			t.Errorf("TC %s propuesto por %s: %+v", code, want, item)
		}
	}
	if reg.item(tcOther) != nil || reg.item(tcRetired) != nil || reg.item(simfTC.Code) != nil {
		t.Errorf("no se propone lo que no cruza, lo retirado ni lo de otro sistema: %+v", reg.Items)
	}
	if item := reg.item(tcImpact); len(item.Components) != 1 || item.Components[0].Code != touched || item.Components[0].Name != tag+"-api-pagos" ||
		item.IsCritical || !strings.HasPrefix(item.TestCase.Title, "Rechaza sobre el límite") {
		t.Errorf("el TC muestra solo los componentes que toca el PKG: %+v", item)
	}
	if item := reg.item(tcCritical); !item.IsCritical || item.Components == nil || len(item.Components) != 0 {
		t.Errorf("TC crítico sin componentes: %+v", item)
	}
	if !slices.IsSortedFunc(reg.Items, func(x, y regressionItem) int { return strings.Compare(x.TestCase.Code, y.TestCase.Code) }) {
		t.Error("los TC van por código")
	}
	// Los críticos que dejaron otras pruebas en el sistema también se proponen: QA los quita con su criterio.
	foreign := 0
	for _, item := range reg.Items {
		if !slices.Contains([]string{tcImpact, tcBoth, tcCritical}, item.TestCase.Code) {
			if item.Origin != "critical" || !item.IsCritical {
				t.Errorf("TC ajeno propuesto sin ser crítico: %+v", item)
			}
			a.call("qa", "PUT", regression+"/items/"+item.TestCase.Code, map[string]any{"included": false, "reason": "De otro cambio"}, 200, nil)
			foreign++
		}
	}
	if reg.Summary.Proposed != 3+foreign || reg.Summary.Included != 3+foreign || reg.Summary.Pending != 3+foreign || reg.Summary.Added != 0 {
		t.Errorf("resumen de la propuesta: %+v", reg.Summary)
	}

	// --- Ajustar: el criterio es obligatorio cuando la decisión difiere de lo propuesto. ---
	item := func(code string) string { return regression + "/items/" + code }
	a.call("dev", "PUT", item(tcCritical), map[string]any{"included": false, "reason": "x"}, 403, nil)
	if a.call("qa", "PUT", item(tcCritical), map[string]any{"reason": "x"}, 422, &p); !hasField(p, "included", "required") {
		t.Errorf("ajuste sin decisión: %+v", p)
	}
	if a.call("qa", "PUT", item(tcCritical), map[string]any{"included": false}, 422, &p); !hasField(p, "reason", "required") {
		t.Errorf("quitar un TC propuesto sin criterio: %+v", p)
	}
	if a.call("qa", "PUT", item(tcOther), map[string]any{"included": true, "reason": " "}, 422, &p); !hasField(p, "reason", "required") {
		t.Errorf("agregar un TC no propuesto sin criterio: %+v", p)
	}
	a.call("qa", "PUT", item("TC-2026-9999"), map[string]any{"included": true, "reason": "x"}, 404, nil)
	if a.call("qa", "PUT", item(simfTC.Code), map[string]any{"included": true, "reason": "x"}, 409, &p); p.Code != "INVALID_ACTION" {
		t.Errorf("TC de otro sistema: %+v", p)
	}
	if a.call("qa", "PUT", item(tcRetired), map[string]any{"included": true, "reason": "x"}, 409, &p); !strings.Contains(p.Detail, "retirado") {
		t.Errorf("TC retirado: %+v", p)
	}
	reg = regressionBody{}
	a.call("qa", "PUT", item(tcCritical), map[string]any{"included": false, "reason": "  El inicio de sesión no cambia  "}, 200, &reg)
	if got := reg.item(tcCritical); got.Included || !got.Proposed || got.Reason != "El inicio de sesión no cambia" ||
		got.DecidedBy == nil || got.DecidedBy.DisplayName != "A. Rojas" || got.DecidedAt == nil {
		t.Errorf("TC quitado con su criterio: %+v", got)
	}
	reg = regressionBody{}
	a.call("admin", "PUT", item(tcOther), map[string]any{"included": true, "reason": "Comparte la base con pagos"}, 200, &reg)
	if got := reg.item(tcOther); got == nil || got.Origin != "manual" || got.Proposed || !got.Included || got.DecidedBy.DisplayName != "M. León" ||
		len(got.Components) != 0 {
		t.Errorf("TC agregado por QA: %+v", got)
	}
	if s := reg.Summary; s.Proposed != 3+foreign || s.Included != 3 || s.Added != 1 || s.Removed != 1+foreign || s.Pending != 3 || s.Passed != 0 {
		t.Errorf("resumen tras los ajustes: %+v", s)
	}

	// --- Confirmar: desde entonces no cambia. ---
	if p = move("qa", "in_testing", "", 409); !strings.Contains(p.Detail, "Falta confirmar la selección de regresión.") {
		t.Errorf("a pruebas sin confirmar: %+v", p)
	}
	a.call("dev", "POST", regression+"/confirmation", nil, 403, nil)
	reg = regressionBody{}
	a.call("qa", "POST", regression+"/confirmation", nil, 200, &reg)
	if reg.ConfirmedBy == nil || reg.ConfirmedBy.DisplayName != "A. Rojas" || reg.ConfirmedAt == nil {
		t.Errorf("regresión confirmada: %+v", reg)
	}
	if a.call("qa", "POST", regression+"/confirmation", nil, 409, &p); p.Code != "INVALID_ACTION" {
		t.Errorf("confirmar dos veces: %+v", p)
	}
	if a.call("qa", "POST", regression+"/proposal", nil, 409, &p); p.Code != "INVALID_ACTION" {
		t.Errorf("proponer con la selección confirmada: %+v", p)
	}
	if a.call("qa", "PUT", item(tcImpact), map[string]any{"included": false, "reason": "x"}, 409, &p); p.Code != "INVALID_ACTION" {
		t.Errorf("ajustar con la selección confirmada: %+v", p)
	}

	// Cambiar el contenido del PKG quita la confirmación: la selección ya no corresponde.
	pkg = packageBody{}
	a.call("dev", "PUT", path+"/builds/"+secondBuild, nil, 200, &pkg)
	if len(pkg.Builds) != 2 || !slices.Equal(pkg.rules("in_testing"), []string{"regression_confirmed"}) {
		t.Errorf("PKG con dos BUILD y la regresión por confirmar otra vez: %+v", pkg)
	}
	reg = regressionBody{}
	a.call("ops", "GET", regression, nil, 200, &reg)
	if reg.ConfirmedBy != nil || reg.ConfirmedAt != nil || len(reg.Items) != 4+foreign {
		t.Errorf("cambiar el contenido reabre la regresión sin tocar sus TC: %+v", reg)
	}
	pkg = packageBody{}
	a.call("dev", "DELETE", path+"/builds/"+secondBuild, nil, 200, &pkg)
	pkg = packageBody{}
	a.call("dev", "DELETE", path+"/builds/"+secondBuild, nil, 200, &pkg) // quitar lo que no está no cambia nada
	a.call("dev", "DELETE", path+"/builds/BUILD-2026-9999", nil, 404, nil)
	a.call("qa", "DELETE", path+"/builds/"+build, nil, 403, nil)
	if len(pkg.Builds) != 1 || pkg.Builds[0].Code != build {
		t.Errorf("BUILD del PKG tras quitar: %+v", pkg.Builds)
	}
	a.call("qa", "POST", regression+"/confirmation", nil, 200, nil)
	pkg = packageBody{}
	a.call("qa", "POST", path+"/transitions", map[string]any{"to": "in_testing"}, 200, &pkg)
	if pkg.State.Key != "in_testing" || len(pkg.Missing) != 2 || len(pkg.rules("in_assembly")) != 0 ||
		!slices.Equal(pkg.rules("ready_to_release"), []string{"regression_passed"}) {
		t.Fatalf("PKG en pruebas: %+v", pkg)
	}

	// --- El contenido solo cambia en armado. ---
	for _, c := range []struct{ method, path string }{
		{"PUT", path + "/builds/" + secondBuild}, {"DELETE", path + "/builds/" + build},
		{"PUT", path + "/requirements/" + req.Code}, {"DELETE", path + "/requirements/" + req.Code},
	} {
		if a.call("dev", c.method, c.path, nil, 409, &p); p.Code != "INVALID_ACTION" || !strings.Contains(p.Detail, "en armado") {
			t.Errorf("%s %s fuera de armado: %+v", c.method, c.path, p)
		}
	}

	// --- Volver a armado: con motivo, y quita la confirmación. Volver a proponer conserva los ajustes. ---
	move("dev", "in_assembly", "Falta un BUILD", 403)
	if p = move("qa", "in_assembly", " ", 422); !hasField(p, "reason", "required") {
		t.Errorf("volver a armado sin motivo: %+v", p)
	}
	move("qa", "in_assembly", "Falta revisar la selección", 200)
	reg = regressionBody{}
	a.call("ops", "GET", regression, nil, 200, &reg)
	if reg.ConfirmedBy != nil {
		t.Errorf("volver a armado quita la confirmación: %+v", reg.ConfirmedBy)
	}
	reg = regressionBody{}
	a.call("admin", "POST", regression+"/proposal", nil, 200, &reg)
	if got := reg.item(tcCritical); got == nil || got.Included || got.Reason != "El inicio de sesión no cambia" || got.DecidedBy.DisplayName != "A. Rojas" {
		t.Errorf("volver a proponer conserva lo que QA quitó: %+v", got)
	}
	if got := reg.item(tcOther); got == nil || got.Origin != "manual" || !got.Included || got.Reason != "Comparte la base con pagos" {
		t.Errorf("volver a proponer conserva lo que QA agregó: %+v", got)
	}
	if reg.ProposedBy.DisplayName != "M. León" || reg.Summary.Included != 3 || reg.Summary.Removed != 1+foreign {
		t.Errorf("segunda propuesta: %+v", reg.Summary)
	}
	// Quitar un REQ no cambia su estado; al volver a agregarlo ya está «En paquete».
	pkg = packageBody{}
	a.call("dev", "DELETE", path+"/requirements/"+req.Code, nil, 200, &pkg)
	a.call("ops", "GET", reqPath, nil, 200, &req)
	if len(pkg.Requirements) != 0 || req.State.Key != "in_package" {
		t.Errorf("quitar el REQ: %+v %+v", pkg.Requirements, req.State)
	}
	pkg = packageBody{}
	a.call("dev", "PUT", path+"/requirements/"+req.Code, nil, 200, &pkg)
	a.call("qa", "POST", regression+"/confirmation", nil, 200, nil)
	move("qa", "in_testing", "", 200)

	// --- A «Listo para liberar»: la regresión final con RUN aprobado sobre un BUILD del PKG y ningún DEF alto abierto. ---
	move("dev", "ready_to_release", "", 403)
	if p = move("qa", "ready_to_release", "", 409); !strings.Contains(p.Detail, "No se puede pasar a «Listo para liberar».") ||
		!strings.Contains(p.Detail, "Faltan 3 TC sin RUN aprobado sobre un BUILD del PKG") || !strings.Contains(p.Detail, tcImpact) ||
		!strings.Contains(p.Detail, tcOther) || strings.Contains(p.Detail, tcCritical) || strings.Contains(p.Detail, "DEF") {
		t.Errorf("los RUN sobre otro BUILD no cuentan, y lo que QA quitó no se exige: %+v", p)
	}
	failed := run(tcImpact, build, "failed", 40*time.Minute)
	var def defectBody
	a.call("qa", "POST", "/api/defects", map[string]any{
		"title": "Acepta sobre el límite " + tag, "description": "Pasos…", "requirement": req.Code, "run": failed, "severity": "high",
	}, 201, &def)
	reg = regressionBody{}
	a.call("ops", "GET", regression, nil, 200, &reg)
	if got := reg.item(tcImpact); got.LastRun == nil || got.LastRun.Code != failed || got.LastRun.Result != "failed" || got.LastRun.Build != build ||
		reg.Summary.Failed != 1 || reg.Summary.Pending != 2 {
		t.Errorf("avance con un RUN fallido: %+v %+v", got.LastRun, reg.Summary)
	}
	if p = move("qa", "ready_to_release", "", 409); !strings.Contains(p.Detail, "Faltan 3 TC sin RUN aprobado") ||
		!strings.Contains(p.Detail, "Hay 1 DEF abierto de severidad alta: "+def.Code) {
		t.Errorf("con un TC fallido y un DEF alto: %+v", p)
	}
	pkg = read()
	if got := pkg.rules("ready_to_release"); !slices.Equal(got, []string{"no_open_defect_with_severity", "regression_passed"}) {
		t.Errorf("qué falta para liberar: %v", got)
	}
	for _, m := range pkg.Missing {
		for _, c := range m.Conditions {
			switch c.RuleKey {
			case "regression_passed":
				if !c.Blocks || !slices.Equal(c.Related, []string{tcImpact, tcBoth, tcOther}) {
					t.Errorf("TC pendientes: %+v", c)
				}
			case "no_open_defect_with_severity":
				if !c.Blocks || !slices.Equal(c.Related, []string{def.Code}) {
					t.Errorf("DEF que bloquea: %+v", c)
				}
			}
		}
	}

	// Se registran los RUN sobre el BUILD del PKG: cuenta el último de cada TC.
	passed := run(tcImpact, build, "passed", 30*time.Minute)
	run(tcBoth, build, "passed", 30*time.Minute)
	run(tcOther, build, "passed", 30*time.Minute)
	run(tcOther, oldBuild, "failed", 10*time.Minute) // sobre un BUILD que no es del PKG: no cuenta
	reg = regressionBody{}
	a.call("ops", "GET", regression, nil, 200, &reg)
	if got := reg.item(tcImpact); got.LastRun == nil || got.LastRun.Code != passed || reg.Summary.Passed != 3 || reg.Summary.Failed != 0 || reg.Summary.Pending != 0 {
		t.Errorf("avance con todo aprobado: %+v %+v", got.LastRun, reg.Summary)
	}
	if p = move("qa", "ready_to_release", "", 409); strings.Contains(p.Detail, "TC") || !strings.Contains(p.Detail, "Hay 1 DEF abierto de severidad alta: "+def.Code) {
		t.Errorf("con la regresión aprobada y el DEF alto abierto: %+v", p)
	}
	defect := func(user, to string) {
		t.Helper()
		a.call(user, "POST", "/api/defects/"+def.Code+"/transitions", map[string]any{"to": to}, 200, nil)
	}
	defect("qa", "triaged")
	defect("dev", "in_fix")
	defect("dev", "ready_for_retest")
	defect("qa", "closed")
	pkg = packageBody{}
	a.call("qa", "POST", path+"/transitions", map[string]any{"to": "ready_to_release"}, 200, &pkg)
	if pkg.State.Key != "ready_to_release" || len(pkg.Missing) != 1 || len(pkg.rules("approved_for_production")) != 0 {
		t.Fatalf("PKG listo para liberar: %+v", pkg)
	}

	// --- Aprobar para producción: change management. ---
	move("qa", "approved_for_production", "", 403)
	move("admin", "approved_for_production", "", 403)
	move("manager", "approved_for_production", "", 200)

	// --- A «Desplegado»: hace falta un despliegue exitoso del PKG en un ambiente de producción. ---
	var prod codeBody
	a.call("qa", "POST", "/api/environments", map[string]any{"system": "SYPAGO", "name": a.uniqueName("Producción"), "kind": "production", "os": "linux"}, 201, &prod)
	deploy := func(user, env, result string, wantStatus int) deploymentBody {
		t.Helper()
		var out deploymentBody
		a.call(user, "POST", "/api/deployments", map[string]any{"environment": env, "package": pkg.Code, "result": result, "executedAt": ago(time.Minute)}, wantStatus, &out)
		return out
	}
	move("qa", "deployed", "", 403)
	if p = move("manager", "deployed", "", 409); !strings.Contains(p.Detail, "Falta un despliegue exitoso del PKG en un ambiente de producción.") {
		t.Errorf("a desplegado sin despliegue: %+v", p)
	}
	deploy("qa", environment, "success", 201) // en un ambiente de prueba no cuenta
	deploy("qa", prod.Code, "success", 403)   // en producción despliega change management
	deploy("manager", prod.Code, "failed", 201)
	if p = move("manager", "deployed", "", 409); p.Code != "INVALID_ACTION" {
		t.Errorf("un despliegue de prueba o uno fallido no cuentan: %+v", p)
	}
	if got := read().rules("deployed"); !slices.Equal(got, []string{"deployed_to_production"}) {
		t.Errorf("qué falta para desplegado: %v", got)
	}
	dep := deploy("manager", prod.Code, "success", 201)
	if dep.Package == nil || dep.Package.Code != pkg.Code || dep.Package.Title != title {
		t.Errorf("DEP del PKG: %+v", dep)
	}
	move("manager", "deployed", "", 200)
	move("manager", "verified", "", 403)

	// --- A «Verificado» por la ruta ligera: hace falta una prueba de humo aprobada, la última que registró QA.
	// El DIAG solo lo exige la ruta alta. ---
	if got := read().rules("verified"); !slices.Equal(got, []string{"smoke_passed"}) {
		t.Errorf("qué falta para verificado en la ruta ligera: %v", got)
	}
	if p = move("qa", "verified", "", 409); !strings.Contains(p.Detail, "Falta la prueba de humo posterior al despliegue.") {
		t.Errorf("a verificado sin SMOKE: %+v", p)
	}
	smoke := func(user, result, notes string, wantStatus int) {
		t.Helper()
		a.call(user, "POST", "/api/smoke-tests", map[string]any{"deployment": dep.Code, "result": result, "notes": notes}, wantStatus, nil)
	}
	smoke("manager", "passed", "", 403)
	smoke("qa", "failed", "", 422) // una prueba fallida dice qué falló
	smoke("qa", "failed", "El login tarda 9 s", 201)
	if p = move("qa", "verified", "", 409); !strings.Contains(p.Detail, "La última prueba de humo falló.") {
		t.Errorf("a verificado con el SMOKE fallido: %+v", p)
	}
	smoke("qa", "passed", "", 201)
	pkg = packageBody{}
	a.call("qa", "POST", path+"/transitions", map[string]any{"to": "verified"}, 200, &pkg)
	if pkg.State.Key != "verified" || pkg.Missing == nil || len(pkg.Missing) != 0 {
		t.Fatalf("PKG verificado: %+v", pkg)
	}
	move("qa", "in_assembly", "x", 409) // el flujo no vuelve atrás desde el final

	// --- Lista: filtros y contenido, sin «qué falta». ---
	var page struct {
		Total int           `json:"total"`
		Items []packageBody `json:"items"`
	}
	a.call("ops", "GET", "/api/packages?system=SYPAGO&state=verified&q="+tag, nil, 200, &page)
	if page.Total != 1 || page.Items[0].Code != pkg.Code || len(page.Items[0].Builds) != 1 || len(page.Items[0].Requirements) != 1 ||
		page.Items[0].Missing == nil || len(page.Items[0].Missing) != 0 {
		t.Errorf("lista de PKG: %+v", page)
	}
	for _, query := range []string{"system=SIMF&q=" + tag, "state=in_assembly&q=" + tag} {
		if a.call("ops", "GET", "/api/packages?"+query, nil, 200, &page); page.Total != 0 || page.Items == nil {
			t.Errorf("lista con %s: %+v", query, page)
		}
	}
	a.call("ops", "GET", "/api/packages?q="+pkg.Code, nil, 200, &page)
	if page.Total != 1 {
		t.Errorf("buscar por código: %+v", page)
	}

	// --- Documentos: se enlazan a un PKG y a un DEP. ---
	pdf := []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n% " + tag)
	var doc documentBody
	a.upload("qa", "/api/documents", map[string]string{"title": "Acta de liberación " + tag, "kind": "minutes", "entityType": "package", "entityCode": pkg.Code},
		"acta.pdf", pdf, 201, &doc)
	a.upload("qa", "/api/documents", map[string]string{"title": "x", "kind": "minutes", "entityType": "package", "entityCode": "PKG-2026-9999"},
		"acta.pdf", pdf, 422, nil)
	a.call("qa", "POST", "/api/documents/"+doc.Code+"/links", map[string]any{"entityType": "deployment", "entityCode": dep.Code}, 200, &doc)
	a.call("qa", "POST", "/api/documents/"+doc.Code+"/links", map[string]any{"entityType": "deployment", "entityCode": "DEP-2026-9999"}, 422, nil)
	if len(doc.LinkedTo) != 2 {
		t.Errorf("documento enlazado al PKG y al DEP: %+v", doc.LinkedTo)
	}
	var docs struct {
		Items []documentBody `json:"items"`
	}
	for _, entity := range []string{"entityType=package&entityCode=" + pkg.Code, "entityType=deployment&entityCode=" + dep.Code} {
		if a.call("ops", "GET", "/api/documents?"+entity, nil, 200, &docs); len(docs.Items) != 1 || docs.Items[0].Code != doc.Code {
			t.Errorf("documentos con %s: %+v", entity, docs.Items)
		}
	}

	// --- Comentarios: se comenta un PKG. ---
	a.call("auditor", "POST", "/api/comments", map[string]any{"entityType": "package", "entityCode": pkg.Code, "body": "x"}, 403, nil)
	a.call("dev", "POST", "/api/comments", map[string]any{"entityType": "package", "entityCode": "PKG-2026-9999", "body": "x"}, 422, nil)
	a.call("dev", "POST", "/api/comments", map[string]any{"entityType": "package", "entityCode": pkg.Code, "body": "  Listo para la ventana del jueves  "}, 201, nil)
	var comments struct {
		Items []struct {
			Body   string `json:"body"`
			Author struct {
				DisplayName string `json:"displayName"`
			} `json:"author"`
		} `json:"items"`
	}
	a.call("ops", "GET", "/api/comments?entityType=package&entityCode="+pkg.Code, nil, 200, &comments)
	if len(comments.Items) != 1 || comments.Items[0].Body != "Listo para la ventana del jueves" || comments.Items[0].Author.DisplayName != "J. Díaz" {
		t.Errorf("comentarios del PKG: %+v", comments.Items)
	}

	// --- Matriz filtrada por paquete: solo los REQ del PKG. ---
	var matrix matrixBody
	a.call("ops", "GET", "/api/traceability/matrix?package="+pkg.Code, nil, 200, &matrix)
	if matrix.Total == 0 {
		t.Fatalf("la matriz del PKG no puede estar vacía: %+v", matrix)
	}
	for _, row := range matrix.Items {
		if row.Requirement.Code != req.Code {
			t.Errorf("la matriz del PKG solo trae sus REQ: %+v", row.Requirement)
		}
	}
	a.call("ops", "GET", "/api/traceability/matrix?package=PKG-2026-9999", nil, 200, &matrix)
	if matrix.Total != 0 {
		t.Errorf("matriz de un PKG que no existe: %d filas", matrix.Total)
	}

	// --- Auditoría: el PKG y su regresión quedan con quién, qué y por qué. ---
	a.call("dev", "GET", "/api/audit?entityCode="+pkg.Code, nil, 403, nil)
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?size=200&entityCode="+pkg.Code, nil, 200, &audit)
	has := func(action, field, oldValue, newValue, reason, actor string) bool {
		for _, e := range audit.Items {
			if e.Action == action && e.Field == field && e.OldValue == oldValue && e.NewValue == newValue && e.Reason == reason &&
				e.Actor.DisplayName == actor && e.EntityCode == pkg.Code {
				return true
			}
		}
		return false
	}
	for name, found := range map[string]bool{
		"alta":                     has("create", "", "", title, "", "J. Díaz"),
		"edición":                  has("update", "description", "Primera entrega", "Límites de noviembre", "", "J. Díaz"),
		"BUILD agregado":           has("update", "builds", "", build, "", "J. Díaz"),
		"BUILD quitado":            has("update", "builds", secondBuild, "", "", "J. Díaz"),
		"REQ agregado":             has("update", "requirements", "", req.Code, "", "J. Díaz"),
		"REQ quitado":              has("update", "requirements", req.Code, "", "", "J. Díaz"),
		"a pruebas":                has("transition", "state", "in_assembly", "in_testing", "", "A. Rojas"),
		"vuelta a armado":          has("transition", "state", "in_testing", "in_assembly", "Falta revisar la selección", "A. Rojas"),
		"aprobado para producción": has("transition", "state", "ready_to_release", "approved_for_production", "", "C. Mora"),
		"verificado":               has("transition", "state", "deployed", "verified", "", "A. Rojas"),
		"propuesta":                has("create", "proposed", "", fmt.Sprint(3+foreign), "", "A. Rojas"),
		"segunda propuesta":        has("update", "proposed", fmt.Sprint(3+foreign), fmt.Sprint(3+foreign), "", "M. León"),
		"TC quitado":               has("decision", "item", tcCritical+": included", tcCritical+": excluded", "El inicio de sesión no cambia", "A. Rojas"),
		"TC agregado":              has("decision", "item", tcOther+": absent", tcOther+": included", "Comparte la base con pagos", "M. León"),
		"confirmación":             has("decision", "confirmed", "false", "true", "", "A. Rojas"),
		"reabierta por contenido":  has("update", "confirmed", "true", "false", "Cambió el contenido del PKG.", "J. Díaz"),
		"reabierta por armado":     has("update", "confirmed", "true", "false", "El PKG volvió a armado.", "A. Rojas"),
	} {
		if !found {
			t.Errorf("falta en la auditoría del PKG: %s", name)
		}
	}
	if audit.Total > 200 {
		t.Logf("la auditoría del PKG tiene %d registros: se revisaron los 200 más recientes", audit.Total)
	}
	var v verification
	a.call("auditor", "GET", "/api/audit/verification", nil, 200, &v)
	if !v.Intact {
		t.Errorf("la cadena de auditoría debía seguir íntegra: %+v", v)
	}
}

// Dos escrituras simultáneas sobre la misma selección van una tras otra: de varias confirmaciones
// a la vez vale una sola, y las demás ven la selección ya confirmada.
func TestConcurrentRegressionConfirmationsDoNotOverlap(t *testing.T) {
	a := newAPI(t)
	var pkg packageBody
	pkg = packageBody{}
	a.call("dev", "POST", "/api/packages", map[string]any{"title": a.uniqueName("Paquete concurrente"), "system": "SIMF"}, 201, &pkg)
	regression := "/api/packages/" + pkg.Code + "/regression"
	a.call("qa", "POST", regression+"/proposal", nil, 200, nil)

	const parallel = 12
	var ok, rejected, other atomic.Int32
	var wg sync.WaitGroup
	for range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", regression+"/confirmation", nil)
			req.Header.Set("Authorization", "Bearer qa")
			req.Header.Set("Origin", testOrigin)
			rec := httptest.NewRecorder()
			a.handler.ServeHTTP(rec, req)
			switch rec.Code {
			case 200:
				ok.Add(1)
			case 409:
				rejected.Add(1)
			default:
				other.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 || rejected.Load() != parallel-1 || other.Load() != 0 {
		t.Fatalf("de %d confirmaciones simultáneas debe valer una sola: %d ok, %d rechazadas, %d con otro código",
			parallel, ok.Load(), rejected.Load(), other.Load())
	}
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+pkg.Code, nil, 200, &audit)
	confirmations := 0
	for _, item := range audit.Items {
		if item.Field == "confirmed" {
			confirmations++
		}
	}
	if confirmations != 1 {
		t.Errorf("la auditoría debe tener una sola confirmación, tiene %d", confirmations)
	}
}
