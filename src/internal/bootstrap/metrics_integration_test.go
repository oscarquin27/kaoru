//go:build integration

package bootstrap

import (
	"slices"
	"testing"
	"time"
)

type coverageMetricsBody struct {
	Total   int `json:"total"`
	Summary struct {
		Requirements      int `json:"requirements"`
		Below             int `json:"below"`
		NoTestCases       int `json:"noTestCases"`
		Unassessed        int `json:"unassessed"`
		Meets             int `json:"meets"`
		RequiredTestCases int `json:"requiredTestCases"`
		PassedTestCases   int `json:"passedTestCases"`
	} `json:"summary"`
	Items []struct {
		Requirement    codeBody `json:"requirement"`
		RiskLevel      string   `json:"riskLevel"`
		Required       int      `json:"required"`
		Passed         int      `json:"passed"`
		Percent        int      `json:"percent"`
		MinimumPercent *float64 `json:"minimumPercent"`
		Status         string   `json:"status"`
	} `json:"items"`
}

type defectStatesBody struct {
	Period struct {
		From string  `json:"from"`
		To   *string `json:"to"`
	} `json:"period"`
	Total  int `json:"total"`
	States []struct {
		State struct {
			Key string `json:"key"`
		} `json:"state"`
		IsFinal    bool     `json:"isFinal"`
		Current    int      `json:"current"`
		Defects    int      `json:"defects"`
		Visits     int      `json:"visits"`
		MedianDays *float64 `json:"medianDays"`
	} `json:"states"`
}

// state devuelve las cuentas de un estado: actuales, que pasaron y entradas.
func (b defectStatesBody) state(key string) [3]int {
	for _, s := range b.States {
		if s.State.Key == key {
			return [3]int{s.Current, s.Defects, s.Visits}
		}
	}
	return [3]int{}
}

type regressionMetricsBody struct {
	Summary struct {
		Packages int `json:"packages"`
	} `json:"summary"`
	Items []struct {
		Package  codeBody `json:"package"`
		Proposed int      `json:"proposed"`
		Final    int      `json:"final"`
		Added    int      `json:"added"`
		Removed  int      `json:"removed"`
		ByOrigin struct {
			Impact   int `json:"impact"`
			Critical int `json:"critical"`
			Manual   int `json:"manual"`
		} `json:"byOrigin"`
		Passed          int `json:"passed"`
		SystemTestCases int `json:"systemTestCases"`
	} `json:"items"`
}

type escapedMetricsBody struct {
	Total   int `json:"total"`
	Summary struct {
		Defects     int     `json:"defects"`
		Escaped     int     `json:"escaped"`
		RatePercent float64 `json:"ratePercent"`
	} `json:"summary"`
	Months []struct {
		Month   string `json:"month"`
		Defects int    `json:"defects"`
		Escaped int    `json:"escaped"`
	} `json:"months"`
	Items []struct {
		Defect      codeBody `json:"defect"`
		Requirement string   `json:"requirement"`
		Package     string   `json:"package"`
	} `json:"items"`
}

// Las métricas leen lo que dejaron los demás features: cobertura por REQ frente al mínimo de su riesgo,
// tiempo en estado de los DEF desde la auditoría, tamaño de la regresión confirmada y defectos que se
// abrieron después de que su REQ llegó a producción. Otras pruebas comparten la base: se miden diferencias.
func TestMetricsReadWhatOtherFeaturesStored(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	build, environment := a.catalog("SYPAGO")
	executedAt := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	// --- Cobertura: un CHG con un REQ a medias y otro sin riesgo. ---
	var chg changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": a.uniqueName("Métricas"), "system": "SYPAGO"}, 201, &chg)
	var req, unassessed requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Tope por comercio", "system": "SYPAGO", "change": chg.Code}, 201, &req)
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Sin riesgo", "system": "SYPAGO", "change": chg.Code}, 201, &unassessed)
	a.call("qa", "PUT", "/api/requirements/"+req.Code+"/risk", map[string]any{"level": "low", "justification": "Estable"}, 200, nil)
	var uc, tc1, tc2 codeBody
	a.call("qa", "POST", "/api/requirements/"+req.Code+"/functional-cases", map[string]any{"title": "Tope"}, 201, &uc)
	tc := func(title string) map[string]any {
		return map[string]any{"title": title, "system": "SYPAGO", "steps": "1. Pagar", "expectedResult": "Rechazado", "functionalCases": []string{uc.Code}}
	}
	a.call("qa", "POST", "/api/test-cases", tc("Rechaza sobre el tope"), 201, &tc1)
	a.call("qa", "POST", "/api/test-cases", tc("Acepta bajo el tope"), 201, &tc2)
	run := func(tc, result string) codeBody {
		var out codeBody
		a.call("qa", "POST", "/api/runs", map[string]any{"testCase": tc, "build": build, "environment": environment, "result": result, "executedAt": executedAt}, 201, &out)
		return out
	}
	run(tc1.Code, "passed")
	failed := run(tc2.Code, "failed")

	var cov coverageMetricsBody
	a.call("auditor", "GET", "/api/metrics/coverage?change="+chg.Code, nil, 200, &cov)
	if s := cov.Summary; s.Requirements != 2 || s.Below != 1 || s.Unassessed != 1 || s.RequiredTestCases != 2 || s.PassedTestCases != 1 || cov.Total != 2 {
		t.Errorf("resumen de cobertura: %+v", cov.Summary)
	}
	// Primero lo que está por debajo del mínimo; la regla exige el 100 % en riesgo bajo.
	if len(cov.Items) != 2 || cov.Items[0].Requirement.Code != req.Code || cov.Items[0].Status != "below" || cov.Items[0].Percent != 50 ||
		cov.Items[0].MinimumPercent == nil || *cov.Items[0].MinimumPercent != 100 || cov.Items[0].RiskLevel != "low" ||
		cov.Items[1].Requirement.Code != unassessed.Code || cov.Items[1].Status != "unassessed" || cov.Items[1].MinimumPercent != nil {
		t.Errorf("cobertura por REQ: %+v", cov.Items)
	}
	a.call("ops", "GET", "/api/metrics/coverage?change="+chg.Code+"&status=unassessed", nil, 200, &cov)
	if cov.Total != 1 || cov.Summary.Requirements != 2 {
		t.Errorf("filtrar por estado acota la lista, no el resumen: %+v", cov)
	}
	if a.call("ops", "GET", "/api/metrics/coverage?status=bajo", nil, 422, &p); !hasField(p, "status", "invalid") {
		t.Errorf("estado de cobertura desconocido: %+v", p)
	}

	// --- Tiempo en estado: un DEF que pasa por triado y queda en corrección. ---
	states := func() defectStatesBody {
		t.Helper()
		var out defectStatesBody
		a.call("ops", "GET", "/api/metrics/defect-states?system=SYPAGO&severity=medium", nil, 200, &out)
		return out
	}
	before := states()
	defect := func() defectBody {
		var out defectBody
		a.call("qa", "POST", "/api/defects", map[string]any{
			"title": "Acepta sobre el tope", "description": "Pasos…", "requirement": req.Code, "run": failed.Code, "severity": "medium",
		}, 201, &out)
		return out
	}
	inTesting := defect()
	a.call("qa", "POST", "/api/defects/"+inTesting.Code+"/transitions", map[string]any{"to": "triaged"}, 200, nil)
	a.call("dev", "POST", "/api/defects/"+inTesting.Code+"/transitions", map[string]any{"to": "in_fix"}, 200, nil)
	after := states()
	if after.Total != before.Total+1 || after.Period.To != nil {
		t.Errorf("DEF medidos: %d antes, %d después; período %+v", before.Total, after.Total, after.Period)
	}
	for key, delta := range map[string][3]int{"open": {0, 1, 1}, "triaged": {0, 1, 1}, "in_fix": {1, 1, 1}, "closed": {0, 0, 0}} {
		b, a := before.state(key), after.state(key)
		if got := [3]int{a[0] - b[0], a[1] - b[1], a[2] - b[2]}; got != delta {
			t.Errorf("%s: cambió %v, quiero %v", key, got, delta)
		}
	}
	if keys := func() (out []string) {
		for _, s := range after.States {
			out = append(out, s.State.Key)
		}
		return out
	}(); !slices.Equal(keys[:5], []string{"open", "triaged", "in_fix", "ready_for_retest", "closed"}) {
		t.Errorf("estados en el orden de la línea de vida: %v", keys)
	}
	a.call("ops", "GET", "/api/metrics/defect-states?from=2026-11-14&to=2026-11-01&severity=x", nil, 422, &p)
	if fields := func() (out []string) {
		for _, e := range p.Errors {
			out = append(out, e.Field+":"+e.Code)
		}
		return out
	}(); !slices.Equal(fields, []string{"severity:invalid", "to:invalid"}) {
		t.Errorf("período al revés y severidad desconocida: %+v", p)
	}
	if a.call("ops", "GET", "/api/metrics/defect-states?from=14-11-2026", nil, 422, &p); !hasField(p, "from", "invalid") {
		t.Errorf("fecha mal escrita: %+v", p)
	}
	// Un período que no incluye hoy no cuenta el DEF recién abierto.
	var past defectStatesBody
	a.call("ops", "GET", "/api/metrics/defect-states?system=SYPAGO&from=2020-01-01&to=2020-12-31", nil, 200, &past)
	if past.Total != 0 || past.Period.From != "2020-01-01" || past.Period.To == nil || *past.Period.To != "2020-12-31" {
		t.Errorf("período pasado: %+v", past)
	}

	// --- Regresión: un PKG con el REQ, su BUILD y la selección confirmada. ---
	var pkg packageBody
	a.call("dev", "POST", "/api/packages", map[string]any{"title": a.uniqueName("Topes"), "system": "SYPAGO"}, 201, &pkg)
	path := "/api/packages/" + pkg.Code
	a.call("dev", "PUT", path+"/builds/"+build, nil, 200, nil)
	a.asOwner(`INSERT INTO package_requirement (package_id, requirement_id, added_by, added_at)
SELECT p.id, r.id, 'sub-dev', now() FROM package p, requirement r WHERE p.code = '` + pkg.Code + `' AND r.code = '` + req.Code + `'`)
	var reg regressionBody
	a.call("qa", "POST", path+"/regression/proposal", nil, 200, &reg)
	a.call("qa", "PUT", path+"/regression/items/"+tc1.Code, map[string]any{"included": true, "reason": "Toca el tope"}, 200, &reg)
	a.call("qa", "POST", path+"/regression/confirmation", nil, 200, &reg)
	var sizes regressionMetricsBody
	a.call("ops", "GET", "/api/metrics/regression?system=SYPAGO&size=200", nil, 200, &sizes)
	found := false
	for _, row := range sizes.Items {
		if row.Package.Code != pkg.Code {
			continue
		}
		found = true
		s := reg.Summary
		if row.Proposed != s.Proposed || row.Final != s.Included || row.Added != s.Added || row.Removed != s.Removed ||
			row.Passed != s.Passed || row.Passed != 1 || row.ByOrigin.Manual != 1 ||
			row.ByOrigin.Impact+row.ByOrigin.Critical+row.ByOrigin.Manual != row.Final || row.SystemTestCases < 2 {
			t.Errorf("tamaño de la regresión %+v; la del PKG: %+v", row, s)
		}
	}
	if !found || sizes.Summary.Packages < 1 {
		t.Errorf("el PKG %s no está en la métrica de regresión: %+v", pkg.Code, sizes)
	}

	// --- Escapados: un DEF abierto después de que el PKG del REQ llegó a producción. ---
	escaped := func() escapedMetricsBody {
		t.Helper()
		var out escapedMetricsBody
		a.call("ops", "GET", "/api/metrics/escaped-defects?system=SYPAGO&size=200", nil, 200, &out)
		return out
	}
	beforeRelease := escaped()
	a.asOwner(`UPDATE package SET state_id = (SELECT id FROM workflow_state WHERE entity_type = 'package' AND key = 'approved_for_production')
WHERE code = '` + pkg.Code + `'`)
	var prod codeBody
	a.call("qa", "POST", "/api/environments", map[string]any{"system": "SYPAGO", "name": a.uniqueName("Producción SyPago"), "kind": "production", "os": "linux"}, 201, &prod)
	// Un despliegue fallido no es una llegada a producción.
	a.call("manager", "POST", "/api/deployments", map[string]any{"environment": prod.Code, "package": pkg.Code, "result": "failed",
		"executedAt": time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)}, 201, nil)
	if got := escaped(); got.Summary.Escaped != beforeRelease.Summary.Escaped {
		t.Errorf("un despliegue fallido no hace escapar nada: %+v", got.Summary)
	}
	// El DEF de pruebas se abrió antes del despliegue exitoso: no escapó.
	a.asOwner(`UPDATE defect SET created_at = created_at - interval '1 hour' WHERE code = '` + inTesting.Code + `'`)
	a.call("manager", "POST", "/api/deployments", map[string]any{"environment": prod.Code, "package": pkg.Code, "result": "success",
		"executedAt": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)}, 201, nil)
	inProduction := defect()
	got := escaped()
	if got.Summary.Defects != beforeRelease.Summary.Defects+1 || got.Summary.Escaped != beforeRelease.Summary.Escaped+1 || got.Total != got.Summary.Escaped {
		t.Errorf("escapados: antes %+v, después %+v", beforeRelease.Summary, got.Summary)
	}
	if len(got.Items) == 0 || got.Items[0].Defect.Code != inProduction.Code || got.Items[0].Requirement != req.Code || got.Items[0].Package != pkg.Code {
		t.Errorf("el DEF escapado va primero, con su REQ y su PKG: %+v", got.Items)
	}
	for _, item := range got.Items {
		if item.Defect.Code == inTesting.Code {
			t.Errorf("el DEF abierto antes de producción no escapó: %+v", item)
		}
	}
	if n := len(got.Months); n != 12 || got.Months[n-1].Month != time.Now().UTC().Format("2006-01") || got.Months[n-1].Escaped < 1 {
		t.Errorf("meses del período: %+v", got.Months)
	}
}
