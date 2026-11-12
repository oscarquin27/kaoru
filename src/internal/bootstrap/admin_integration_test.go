//go:build integration

package bootstrap

import (
	"bytes"
	"encoding/csv"
	"slices"
	"strconv"
	"testing"
)

type rolePermissionsBody struct {
	Permissions []struct {
		Key string `json:"key"`
	} `json:"permissions"`
	Roles []struct {
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
	} `json:"roles"`
}

func (b rolePermissionsBody) of(role string) []string {
	for _, r := range b.Roles {
		if r.Role == role {
			return r.Permissions
		}
	}
	return nil
}

// Quién puede hacer cada acción es configuración: se cambia sin tocar código y queda auditado.
func TestAdminConfiguresPermissionsAndRules(t *testing.T) {
	a := newAPI(t)
	var p problemBody

	a.call("qa", "GET", "/api/admin/role-permissions", nil, 403, nil)
	var before rolePermissionsBody
	a.call("admin", "GET", "/api/admin/role-permissions", nil, 200, &before)
	dev := before.of("dev")
	if len(before.Roles) != 6 || len(before.Permissions) < 30 || slices.Contains(dev, "run.create") || !slices.Contains(before.of("qa-admin"), "admin.configure") {
		t.Fatalf("permisos predeterminados inesperados: %+v", before.Roles)
	}
	// Las demás pruebas comparten la base: al terminar, dev vuelve a sus permisos.
	t.Cleanup(func() {
		a.call("admin", "PUT", "/api/admin/role-permissions/dev", map[string]any{"permissions": dev}, 200, nil)
	})

	// Dar a dev un permiso nuevo tiene efecto de inmediato.
	var me struct {
		Permissions []string `json:"permissions"`
	}
	var after rolePermissionsBody
	a.call("qa", "PUT", "/api/admin/role-permissions/dev", map[string]any{"permissions": dev}, 403, nil)
	a.call("admin", "PUT", "/api/admin/role-permissions/dev", map[string]any{"permissions": append(slices.Clone(dev), "run.create", "run.create")}, 200, &after)
	a.call("dev", "GET", "/api/me", nil, 200, &me)
	if !slices.Contains(after.of("dev"), "run.create") || len(after.of("dev")) != len(dev)+1 || !slices.Contains(me.Permissions, "run.create") {
		t.Errorf("dev debía ganar run.create una sola vez: %v", after.of("dev"))
	}

	a.call("admin", "PUT", "/api/admin/role-permissions/dev", map[string]any{"permissions": []string{"volar"}}, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "permissions" {
		t.Errorf("permiso desconocido: %+v", p)
	}
	a.call("admin", "PUT", "/api/admin/role-permissions/root", map[string]any{"permissions": []string{}}, 404, nil)
	a.call("admin", "PUT", "/api/admin/role-permissions/qa-admin", map[string]any{"permissions": []string{"run.create"}}, 409, &p)
	if p.Code != "INVALID_ACTION" {
		t.Errorf("qa-admin no puede quedarse sin configurar: %+v", p)
	}
	a.call("admin", "GET", "/api/me", nil, 200, &me)
	if !slices.Contains(me.Permissions, "admin.configure") {
		t.Error("un cambio rechazado no debe tocar los permisos de qa-admin")
	}

	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode=dev", nil, 200, &audit)
	if audit.Total < 1 || audit.Items[0].Field != "permissions" || audit.Items[0].Actor.DisplayName != "M. León" ||
		slices.Contains(splitComma(audit.Items[0].OldValue), "run.create") || !slices.Contains(splitComma(audit.Items[0].NewValue), "run.create") {
		t.Errorf("el cambio de permisos debe quedar auditado con el antes y el después: %+v", audit.Items)
	}

	// Reglas de bloqueo: la cobertura mínima por nivel de riesgo se ajusta desde la administración.
	var wf struct {
		Transitions []struct {
			To    string `json:"to"`
			Rules []struct {
				ID      string         `json:"id"`
				RuleKey string         `json:"ruleKey"`
				Params  map[string]any `json:"params"`
				Enabled bool           `json:"enabled"`
			} `json:"rules"`
		} `json:"transitions"`
	}
	a.call("admin", "GET", "/api/workflows/requirement", nil, 200, &wf)
	var coverageID string
	var original map[string]any
	for _, tr := range wf.Transitions {
		for _, r := range tr.Rules {
			if r.RuleKey == "min_coverage" {
				coverageID, original = r.ID, r.Params
			}
		}
	}
	if coverageID == "" {
		t.Fatal("no encontré la regla min_coverage")
	}
	t.Cleanup(func() {
		a.call("admin", "PATCH", "/api/admin/blocking-rules/"+coverageID, map[string]any{"params": original, "enabled": true}, 200, nil)
	})
	var rule struct {
		Params  map[string]any `json:"params"`
		Enabled bool           `json:"enabled"`
		Blocks  bool           `json:"blocks"`
	}
	a.call("qa", "PATCH", "/api/admin/blocking-rules/"+coverageID, map[string]any{"enabled": false}, 403, nil)
	a.call("admin", "PATCH", "/api/admin/blocking-rules/"+coverageID, map[string]any{"params": map[string]any{"high": 100, "medium": 80, "low": 50}}, 200, &rule)
	if rule.Params["low"] != float64(50) || !rule.Enabled || !rule.Blocks {
		t.Errorf("regla ajustada: %+v", rule)
	}
	a.call("admin", "PATCH", "/api/admin/blocking-rules/"+coverageID, map[string]any{"enabled": false}, 200, &rule)
	if rule.Enabled || rule.Params["low"] != float64(50) {
		t.Errorf("desactivar no debe tocar los parámetros: %+v", rule)
	}
	a.call("admin", "PATCH", "/api/admin/blocking-rules/no-es-un-id", map[string]any{"enabled": false}, 404, nil)
	a.call("auditor", "GET", "/api/audit?entityCode="+coverageID, nil, 200, &audit)
	if audit.Total != 2 || audit.Items[0].Field != "min_coverage" {
		t.Errorf("los cambios de la regla deben quedar auditados: %+v", audit.Items)
	}
}

func splitComma(s string) []string {
	var out []string
	for _, part := range bytes.Split([]byte(s), []byte(",")) {
		out = append(out, string(part))
	}
	return out
}

func TestAuditExport(t *testing.T) {
	a := newAPI(t)
	a.call("manager", "POST", "/api/changes", map[string]any{"title": "=CMD() exportación", "system": "SIMF"}, 201, nil)

	a.download("ops", "/api/audit/export", 403)
	rec := a.download("auditor", "/api/audit/export", 200)
	records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(rec.Body.Bytes(), []byte("\xEF\xBB\xBF")))).ReadAll()
	if err != nil || len(records) < 2 || records[0][0] != "N.º" || len(records[0]) != 13 {
		t.Fatalf("CSV de auditoría: %v, %d filas", err, len(records))
	}
	previous, neutralized := int64(0), false
	for i, row := range records[1:] {
		seq, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil || seq <= previous || len(row[12]) != 64 {
			t.Fatalf("fila %d fuera de orden o sin hash: %v", i+1, row)
		}
		if i > 0 && row[11] != records[i][12] {
			t.Fatalf("fila %d: su hash anterior no es el hash de la fila anterior", i+1)
		}
		previous = seq
		neutralized = neutralized || row[9] == "'=CMD() exportación"
	}
	if !neutralized {
		t.Error("un valor que empieza con = debe exportarse neutralizado")
	}
	rec = a.download("auditor", "/api/audit/export?from=2099-01-01T00:00:00Z", 200)
	if records, _ := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(rec.Body.Bytes(), []byte("\xEF\xBB\xBF")))).ReadAll(); len(records) != 1 {
		t.Errorf("un rango sin registros exporta solo el encabezado: %d filas", len(records))
	}
	a.download("auditor", "/api/audit/export?from=ayer", 422)
}

type workflowBody struct {
	States []struct {
		Key      string `json:"key"`
		Name     string `json:"name"`
		Position *int   `json:"position"`
	} `json:"states"`
	Transitions []struct {
		From           string `json:"from"`
		To             string `json:"to"`
		Permission     string `json:"permission"`
		RequiresReason bool   `json:"requiresReason"`
	} `json:"transitions"`
}

func (w workflowBody) stateName(key string) string {
	for _, s := range w.States {
		if s.Key == key {
			return s.Name
		}
	}
	return ""
}

func (w workflowBody) hasTransition(from, to string) bool {
	for _, tr := range w.Transitions {
		if tr.From == from && tr.To == to {
			return true
		}
	}
	return false
}

// Los estados y las transiciones de un flujo se editan desde la administración,
// tienen efecto de inmediato sobre las entidades y quedan auditados.
func TestAdminEditsWorkflowStatesAndTransitions(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	var wf, original workflowBody
	const states, transitions = "/api/admin/workflows/change/states", "/api/admin/workflows/change/transitions"

	a.call("admin", "GET", "/api/workflows/change", nil, 200, &original)
	a.call("qa", "POST", states, map[string]any{"key": "on_hold", "name": "En espera"}, 403, nil)
	a.call("admin", "POST", "/api/admin/workflows/risk/states", map[string]any{"key": "on_hold", "name": "En espera"}, 404, nil)
	a.call("admin", "POST", states, map[string]any{"key": "En Espera", "name": ""}, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "key" {
		t.Errorf("clave inválida: %+v", p)
	}
	a.call("admin", "POST", states, map[string]any{"key": "on_hold", "name": "En espera"}, 201, &wf)
	if len(wf.States) != len(original.States)+1 || wf.stateName("on_hold") != "En espera" {
		t.Fatalf("el estado nuevo debía aparecer en el flujo: %+v", wf.States)
	}
	a.call("admin", "POST", states, map[string]any{"key": "on_hold", "name": "Otra vez"}, 409, &p)
	if p.Code != "DUPLICATE_RESOURCE" {
		t.Errorf("clave repetida: %+v", p)
	}

	// Transiciones: registrado → en espera (con motivo) y en espera → aprobado.
	hold := transitions + "/registered/on_hold"
	a.call("qa", "PUT", hold, map[string]any{"permission": "chg.approve"}, 403, nil)
	a.call("admin", "PUT", hold, map[string]any{"permission": "volar"}, 422, nil)
	a.call("admin", "PUT", transitions+"/registered/nope", map[string]any{"permission": "chg.approve"}, 404, nil)
	a.call("admin", "PUT", hold, map[string]any{"permission": "chg.approve"}, 200, nil)
	a.call("admin", "PUT", hold, map[string]any{"permission": "chg.approve", "requiresReason": true}, 200, &wf)
	a.call("admin", "PUT", transitions+"/on_hold/approved", map[string]any{"permission": "chg.approve"}, 200, &wf)
	if len(wf.Transitions) != len(original.Transitions)+2 {
		t.Fatalf("debían sumarse dos transiciones, sin duplicar la que se ajustó: %+v", wf.Transitions)
	}

	// Un CHG usa la transición nueva de inmediato, con sus condiciones.
	var chg changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": "Flujo editado", "system": "SYPAGO"}, 201, &chg)
	move := "/api/changes/" + chg.Code + "/transitions"
	a.call("manager", "POST", move, map[string]any{"to": "on_hold"}, 422, nil)
	a.call("manager", "POST", move, map[string]any{"to": "on_hold", "reason": "Falta el análisis"}, 200, &chg)
	if chg.State.Key != "on_hold" || chg.State.Name != "En espera" {
		t.Fatalf("el CHG debía quedar en espera: %+v", chg.State)
	}

	// Renombrar se ve en la entidad; la clave no cambia.
	a.call("admin", "PATCH", states+"/on_hold", map[string]any{"name": "En pausa", "position": nil}, 200, &wf)
	a.call("ops", "GET", "/api/changes/"+chg.Code, nil, 200, &chg)
	if chg.State.Name != "En pausa" || wf.stateName("on_hold") != "En pausa" {
		t.Errorf("el nombre nuevo debía verse en el CHG: %+v", chg.State)
	}
	a.call("admin", "PATCH", states+"/nope", map[string]any{"name": "X"}, 404, nil)

	// Quitar: ni el inicial, ni uno con transiciones, ni uno con registros.
	a.call("admin", "DELETE", states+"/registered", nil, 409, &p)
	a.call("admin", "DELETE", states+"/on_hold", nil, 409, &p)
	if p.Code != "INVALID_ACTION" {
		t.Errorf("estado con transiciones: %+v", p)
	}
	a.call("admin", "DELETE", hold, nil, 200, nil)
	a.call("admin", "DELETE", hold, nil, 404, nil)
	a.call("admin", "DELETE", transitions+"/on_hold/approved", nil, 200, &wf)
	a.call("admin", "DELETE", states+"/on_hold", nil, 409, &p)
	if p.Code != "INVALID_ACTION" || wf.stateName("on_hold") == "" {
		t.Errorf("un estado con un CHG dentro no se quita: %+v", p)
	}

	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode=change:on_hold", nil, 200, &audit)
	if audit.Total != 2 {
		t.Errorf("alta y cambio de nombre del estado deben quedar auditados: %+v", audit.Items)
	}
	a.call("auditor", "GET", "/api/audit?entityCode=change:registered%3Eon_hold", nil, 200, &audit)
	if audit.Total != 3 {
		t.Errorf("alta, ajuste y baja de la transición deben quedar auditados: %+v", audit.Items)
	}

	// Las demás pruebas comparten la base: el CHG sale del estado y el flujo vuelve a como estaba.
	a.call("admin", "PUT", transitions+"/on_hold/approved", map[string]any{"permission": "chg.approve"}, 200, nil)
	a.call("manager", "POST", move, map[string]any{"to": "approved"}, 200, nil)
	a.call("admin", "DELETE", transitions+"/on_hold/approved", nil, 200, nil)
	a.call("admin", "DELETE", states+"/on_hold", nil, 200, &wf)
	if len(wf.States) != len(original.States) || len(wf.Transitions) != len(original.Transitions) || wf.hasTransition("registered", "on_hold") {
		t.Errorf("el flujo debía volver a su forma original: %+v", wf)
	}
}

// La lista de usuarios sirve para elegir responsables: solo quien ya entró, sin correos.
func TestUsersAreListedForAssignment(t *testing.T) {
	a := newAPI(t)
	a.call("", "GET", "/api/users", nil, 401, nil)
	a.call("manager", "GET", "/api/me", nil, 200, nil)
	a.call("qa", "GET", "/api/me", nil, 200, nil)

	var raw bytes.Buffer
	var page struct {
		Items []map[string]string `json:"items"`
	}
	a.call("qa", "GET", "/api/users", nil, 200, &page)
	if len(page.Items) < 2 {
		t.Fatalf("deben aparecer quienes ya entraron: %+v", page.Items)
	}
	for _, u := range page.Items {
		if u["id"] == "" || u["username"] == "" || u["displayName"] == "" || len(u) != 3 {
			t.Errorf("un usuario trae id, usuario y nombre, nada más: %v", u)
		}
		raw.WriteString(u["displayName"] + "\n")
	}
	names := raw.String()
	if i, j := bytes.Index(raw.Bytes(), []byte("A. Rojas")), bytes.Index(raw.Bytes(), []byte("C. Mora")); i < 0 || j < 0 || i > j {
		t.Errorf("deben venir ordenados por nombre: %s", names)
	}
	a.call("qa", "GET", "/api/users?q=MOR", nil, 200, &page)
	if len(page.Items) != 1 || page.Items[0]["id"] != "sub-manager" {
		t.Errorf("la búsqueda es por nombre o usuario, sin distinguir mayúsculas: %+v", page.Items)
	}
	a.call("qa", "GET", "/api/users?q=%25", nil, 200, &page)
	if len(page.Items) != 0 {
		t.Errorf("un comodín se busca como texto: %+v", page.Items)
	}
}

// Las vistas guardadas son personales: cada quien ve y quita solo las suyas.
func TestSavedViewsArePersonal(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	type view struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Query string `json:"query"`
	}
	var page struct {
		Items []view `json:"items"`
	}
	a.call("", "GET", "/api/saved-views?screen=requirements", nil, 401, nil)
	a.call("qa", "GET", "/api/saved-views?screen=paquetes", nil, 422, nil)
	a.call("qa", "POST", "/api/saved-views", map[string]any{"screen": "requirements", "name": "", "query": ""}, 422, &p)
	if len(p.Errors) != 2 {
		t.Errorf("sin nombre ni filtros: %+v", p.Errors)
	}

	var saved view
	body := map[string]any{"screen": "requirements", "name": "En pruebas", "query": "?state=in_testing&system=SYPAGO"}
	a.call("qa", "POST", "/api/saved-views", body, 201, &saved)
	if saved.ID == "" || saved.Query != "state=in_testing&system=SYPAGO" {
		t.Fatalf("vista guardada inesperada: %+v", saved)
	}
	t.Cleanup(func() { a.call("qa", "DELETE", "/api/saved-views/"+saved.ID, nil, 204, nil) })
	a.call("qa", "POST", "/api/saved-views", body, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "name" {
		t.Errorf("nombre repetido: %+v", p)
	}

	a.call("qa", "GET", "/api/saved-views?screen=requirements", nil, 200, &page)
	if len(page.Items) != 1 || page.Items[0].Name != "En pruebas" {
		t.Errorf("qa debía ver su vista: %+v", page.Items)
	}
	a.call("dev", "GET", "/api/saved-views?screen=requirements", nil, 200, &page)
	a.call("qa", "GET", "/api/saved-views?screen=defects", nil, 200, &page)
	if len(page.Items) != 0 {
		t.Errorf("ni otra persona ni otra lista ven la vista: %+v", page.Items)
	}
	a.call("dev", "DELETE", "/api/saved-views/"+saved.ID, nil, 404, nil)
	a.call("qa", "DELETE", "/api/saved-views/no-es-un-id", nil, 404, nil)
}
