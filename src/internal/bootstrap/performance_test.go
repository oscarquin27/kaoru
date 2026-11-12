//go:build integration && perf

package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

// Volumen del requisito no funcional de rendimiento (requerimiento §11): 10 000 REQ y 100 000 RUN.
// Cada REQ lleva su CHG, su RISK, un UC, dos TC con cinco RUN cada uno y tres registros de auditoría;
// uno de cada cinco tiene un DEF.
const perfVolume = `
INSERT INTO app_user VALUES ('perf', 'perf', 'Usuario de volumen', NULL, now(), now());

INSERT INTO change (id, code, title, system_id, route, state_id, created_by, created_at, updated_at)
SELECT gen_random_uuid(), 'CHG-2025-' || lpad(n::text, 5, '0'), 'Cambio de volumen ' || n,
       (SELECT id FROM system_under_test ORDER BY key LIMIT 1), 'light',
       (SELECT id FROM workflow_state WHERE entity_type = 'change' AND key = 'approved'), 'perf', now(), now()
FROM generate_series(1, 500) n;

INSERT INTO requirement (id, code, title, description, system_id, change_id, state_id, owner_id, created_by, created_at, updated_at)
SELECT gen_random_uuid(), 'REQ-2025-' || lpad(n::text, 5, '0'),
       (ARRAY['Límite diario', 'Tope mensual', 'Reverso de pago', 'Conciliación', 'Notificación al comercio',
              'Comisión por canal', 'Bloqueo preventivo', 'Reporte regulatorio', 'Cierre contable', 'Alta de ordenante'])[1 + n % 10]
           || ' número ' || n,
       'Descripción de volumen del requerimiento ' || n,
       (SELECT id FROM system_under_test ORDER BY key LIMIT 1),
       (SELECT id FROM change WHERE code = 'CHG-2025-' || lpad((1 + n % 500)::text, 5, '0')),
       (SELECT id FROM workflow_state WHERE entity_type = 'requirement' AND key = 'in_testing'),
       'perf', 'perf', now() - (n || ' minutes')::interval, now()
FROM generate_series(1, 10000) n;

INSERT INTO risk_assessment (id, code, requirement_id, level, justification, assessed_by, created_at, updated_at)
SELECT gen_random_uuid(), 'RISK-2025-' || right(r.code, 5), r.id, (ARRAY['high', 'medium', 'low'])[1 + right(r.code, 5)::int % 3],
       'Volumen', 'perf', now(), now()
FROM requirement r WHERE r.code LIKE 'REQ-2025-%';
INSERT INTO risk_quality_characteristic SELECT id, 'reliability' FROM risk_assessment WHERE code LIKE 'RISK-2025-%';

INSERT INTO functional_case (id, code, requirement_id, title, created_by, created_at, updated_at)
SELECT gen_random_uuid(), 'UC-2025-' || right(r.code, 5), r.id, 'Caso de uso de ' || r.title, 'perf', now(), now()
FROM requirement r WHERE r.code LIKE 'REQ-2025-%';

INSERT INTO test_case (id, code, system_id, title, steps, expected_result, created_by, created_at, updated_at)
SELECT gen_random_uuid(), 'TC-2025-' || lpad((right(f.code, 5)::int * 2 - k)::text, 5, '0'), r.system_id,
       'Caso de prueba ' || k || ' de ' || r.title, 'Pasos', 'Resultado esperado', 'perf', now(), now()
FROM functional_case f JOIN requirement r ON r.id = f.requirement_id, generate_series(0, 1) k
WHERE f.code LIKE 'UC-2025-%';

INSERT INTO test_case_functional_case (test_case_id, functional_case_id)
SELECT t.id, f.id
FROM test_case t JOIN functional_case f ON right(f.code, 5)::int = (right(t.code, 5)::int + 1) / 2
WHERE t.code LIKE 'TC-2025-%' AND f.code LIKE 'UC-2025-%';

INSERT INTO test_run (id, code, test_case_id, build_version, build_checksum, environment_name, result, executed_by, executed_at, created_at)
SELECT gen_random_uuid(), 'RUN-2025-' || lpad(((right(t.code, 5)::int - 1) * 5 + k)::text, 6, '0'), t.id,
       '1.' || k || '.0', repeat('a', 64), 'QA limpio',
       CASE WHEN k = 5 AND right(t.code, 5)::int % 7 <> 0 THEN 'passed' WHEN k % 2 = 0 THEN 'failed' ELSE 'passed' END,
       'perf', now() - ((6 - k) || ' days')::interval, now()
FROM test_case t, generate_series(1, 5) k
WHERE t.code LIKE 'TC-2025-%';

INSERT INTO defect (id, code, title, description, requirement_id, test_run_id, severity, state_id, created_by, created_at, updated_at)
SELECT gen_random_uuid(), 'DEF-2025-' || right(r.code, 5), 'Defecto de ' || r.title, 'Volumen', r.id,
       (SELECT tr.id FROM test_run tr JOIN test_case_functional_case l ON l.test_case_id = tr.test_case_id
            JOIN functional_case f ON f.id = l.functional_case_id WHERE f.requirement_id = r.id LIMIT 1),
       (ARRAY['high', 'medium', 'low'])[1 + right(r.code, 5)::int % 3],
       (SELECT id FROM workflow_state WHERE entity_type = 'defect' AND is_initial), 'perf', now(), now()
FROM requirement r WHERE r.code LIKE 'REQ-2025-%' AND right(r.code, 5)::int % 5 = 0;

INSERT INTO audit_entry (id, occurred_at, actor_id, entity_type, entity_id, entity_code, action, field, old_value, new_value, hash)
SELECT gen_random_uuid(), now() - ((4 - k) || ' days')::interval, 'perf', 'requirement', r.id, r.code,
       CASE WHEN k = 1 THEN 'create' ELSE 'transition' END, 'state',
       (ARRAY[NULL, 'registered', 'risk_assessed'])[k], (ARRAY['registered', 'risk_assessed', 'in_testing'])[k],
       encode(sha256((r.code || k)::bytea), 'hex')
FROM requirement r, generate_series(1, 3) k WHERE r.code LIKE 'REQ-2025-%';

ANALYZE;
`

// measure llama a la ruta varias veces y devuelve la mediana y el peor tiempo.
func (a *api) measure(path string) (median, worst time.Duration) {
	a.t.Helper()
	const rounds = 15
	times := make([]time.Duration, 0, rounds)
	for i := range rounds + 2 {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer qa")
		rec := httptest.NewRecorder()
		start := time.Now()
		a.handler.ServeHTTP(rec, req)
		elapsed := time.Since(start)
		if rec.Code != http.StatusOK {
			a.t.Fatalf("GET %s: status %d: %s", path, rec.Code, rec.Body.String())
		}
		if i >= 2 { // las dos primeras calientan cachés y conexiones
			times = append(times, elapsed)
		}
	}
	slices.Sort(times)
	return times[len(times)/2], times[len(times)-1]
}

// La ficha del REQ y la búsqueda responden en menos de 1 s con 10 000 REQ y 100 000 RUN.
// Mide el tiempo del servidor (API y base), sin red ni navegador. Corre sola, sobre una base limpia:
// make check-perf.
func TestPerformanceWithVolume(t *testing.T) {
	a := newAPI(t)
	start := time.Now()
	a.asOwner(perfVolume)
	t.Logf("volumen cargado en %s", time.Since(start).Round(time.Millisecond))

	const limit = time.Second
	routes := []struct {
		name, path string
		required   bool
	}{
		{"ficha del REQ (overview)", "/api/requirements/REQ-2025-05000/overview", true},
		{"REQ", "/api/requirements/REQ-2025-05000", true},
		{"búsqueda por texto", "/api/search?q=conciliaci%C3%B3n", true},
		{"búsqueda por ID", "/api/search?q=REQ-2025-07321", true},
		{"búsqueda sin resultados", "/api/search?q=zzzzzz", true},
		{"lista de REQ", "/api/requirements", false},
		{"lista de REQ filtrada por texto", "/api/requirements?q=reverso", false},
		{"lista de REQ, página 150", "/api/requirements?page=150", false},
		{"matriz", "/api/traceability/matrix", false},
		{"matriz filtrada por riesgo", "/api/traceability/matrix?risk=high", false},
		{"RUN de un REQ", "/api/runs?requirement=REQ-2025-05000", false},
		{"lista de RUN", "/api/runs", false},
		{"lista de TC", "/api/test-cases", false},
		{"lista de DEF", "/api/defects", false},
		{"auditoría de un REQ", "/api/audit?entityCode=REQ-2025-05000", false},
	}
	a.call("auditor", "GET", "/api/me", nil, 200, nil)
	for _, route := range routes {
		user := "qa"
		_ = user
		median, worst := a.measure(route.path)
		t.Logf("%-34s mediana %7s  peor %7s", route.name, median.Round(time.Millisecond), worst.Round(time.Millisecond))
		if worst > limit {
			if route.required {
				t.Errorf("%s tardó %s: el requisito es menos de %s", route.name, worst, limit)
			} else {
				t.Logf("  AVISO: %s supera %s (no es parte del requisito)", route.name, limit)
			}
		}
	}
}
