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
