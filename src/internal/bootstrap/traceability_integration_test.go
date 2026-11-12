//go:build integration

package bootstrap

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"io"
	"strings"
	"testing"
	"time"
)

type overviewBody struct {
	Requirement struct {
		Code   string `json:"code"`
		Route  string `json:"route"`
		Change *struct {
			Code string `json:"code"`
		} `json:"change"`
		Risk *struct {
			Code  string `json:"code"`
			Level string `json:"level"`
		} `json:"risk"`
		State struct {
			Key string `json:"key"`
		} `json:"state"`
	} `json:"requirement"`
	Lifecycle []struct {
		State struct {
			Key string `json:"key"`
		} `json:"state"`
		Status    string  `json:"status"`
		EnteredAt *string `json:"enteredAt"`
		Days      float64 `json:"days"`
	} `json:"lifecycle"`
	Missing []struct {
		To struct {
			Key string `json:"key"`
		} `json:"to"`
		Conditions []struct {
			RuleKey string   `json:"ruleKey"`
			Message string   `json:"message"`
			Blocks  bool     `json:"blocks"`
			Related []string `json:"related"`
		} `json:"conditions"`
	} `json:"missing"`
	Coverage struct {
		Required int `json:"required"`
		Passed   int `json:"passed"`
	} `json:"coverage"`
	FunctionalCases []codeBody `json:"functionalCases"`
	TestCases       []struct {
		Code    string `json:"code"`
		LastRun *struct {
			Result string `json:"result"`
		} `json:"lastRun"`
	} `json:"testCases"`
	Defects []codeBody `json:"defects"`
}

type matrixBody struct {
	Indicators struct {
		CoveragePercent     int            `json:"coveragePercent"`
		BlockedRequirements int            `json:"blockedRequirements"`
		OpenDefects         map[string]int `json:"openDefects"`
	} `json:"indicators"`
	Total int `json:"total"`
	Items []struct {
		Requirement     codeBody `json:"requirement"`
		Risk            string   `json:"risk"`
		RiskLevel       string   `json:"riskLevel"`
		FunctionalCases []string `json:"functionalCases"`
		TestCase        *struct {
			Code    string `json:"code"`
			LastRun *struct {
				Code   string `json:"code"`
				Result string `json:"result"`
			} `json:"lastRun"`
		} `json:"testCase"`
		Defects []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
		} `json:"defects"`
	} `json:"items"`
}

// La matriz y la ficha devuelven la cadena completa REQ → RISK → UC → TC → RUN → DEF.
func TestTraceabilityReturnsTheWholeChain(t *testing.T) {
	a := newAPI(t)
	build, environment := a.catalog("SGLPAR")
	executedAt := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	var chg changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": "Trazabilidad", "system": "SGLPAR"}, 201, &chg)
	var req, bare requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Conciliación =SUMA(A1)", "system": "SGLPAR", "change": chg.Code}, 201, &req)
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "REQ sin pruebas", "system": "SGLPAR", "change": chg.Code}, 201, &bare)
	path := "/api/requirements/" + req.Code

	// Recién registrado: la primera etapa es la actual y la siguiente está bloqueada por el RISK.
	var ov overviewBody
	a.call("ops", "GET", path+"/overview", nil, 200, &ov)
	if ov.Requirement.Route != "high" || ov.Requirement.Risk != nil || len(ov.Lifecycle) != 11 ||
		ov.Lifecycle[0].Status != "current" || ov.Lifecycle[1].Status != "blocked" || ov.Lifecycle[2].Status != "pending" {
		t.Fatalf("ficha recién registrada: %+v", ov.Lifecycle)
	}
	var toRisk []string
	for _, m := range ov.Missing {
		if m.To.Key == "risk_assessed" {
			for _, c := range m.Conditions {
				toRisk = append(toRisk, c.RuleKey)
			}
		}
	}
	if len(toRisk) != 2 {
		t.Errorf("en ruta alta faltan el RISK y sus características: %v", toRisk)
	}

	// Se arma la cadena completa.
	a.call("qa", "PUT", path+"/risk", map[string]any{"level": "high", "justification": "Dinero", "characteristics": []string{"reliability"}}, 200, nil)
	a.call("qa", "POST", path+"/transitions", map[string]any{"to": "risk_assessed"}, 200, nil)
	var uc, tc1, tc2, run1, run2, def codeBody
	a.call("qa", "POST", path+"/functional-cases", map[string]any{"title": "Concilia el día"}, 201, &uc)
	tc := func(title string) map[string]any {
		return map[string]any{"title": title, "system": "SGLPAR", "steps": "1", "expectedResult": "ok", "functionalCases": []string{uc.Code}}
	}
	a.call("qa", "POST", "/api/test-cases", tc("Cuadra"), 201, &tc1)
	a.call("qa", "POST", "/api/test-cases", tc("No cuadra"), 201, &tc2)
	run := func(code, result string) map[string]any {
		return map[string]any{"testCase": code, "build": build, "environment": environment, "result": result, "executedAt": executedAt}
	}
	a.call("qa", "POST", "/api/runs", run(tc1.Code, "passed"), 201, &run1)
	a.call("qa", "POST", "/api/runs", run(tc2.Code, "failed"), 201, &run2)
	a.call("qa", "POST", "/api/defects", map[string]any{
		"title": "Descuadre", "description": "…", "requirement": req.Code, "run": run2.Code, "severity": "high",
	}, 201, &def)

	// Ficha consolidada.
	a.call("auditor", "GET", path+"/overview", nil, 200, &ov)
	if ov.Requirement.Risk == nil || ov.Requirement.Risk.Level != "high" || ov.Requirement.Change == nil ||
		ov.Requirement.Change.Code != chg.Code || ov.Coverage.Required != 2 || ov.Coverage.Passed != 1 ||
		len(ov.FunctionalCases) != 1 || len(ov.TestCases) != 2 || len(ov.Defects) != 1 || ov.Defects[0].Code != def.Code {
		t.Errorf("ficha consolidada: %+v", ov)
	}
	if ov.TestCases[0].LastRun == nil || ov.TestCases[0].LastRun.Result != "passed" || ov.TestCases[1].LastRun.Result != "failed" {
		t.Errorf("último RUN de cada TC: %+v", ov.TestCases)
	}
	if ov.Lifecycle[0].Status != "done" || ov.Lifecycle[0].EnteredAt == nil || ov.Lifecycle[1].Status != "current" || ov.Lifecycle[2].Status != "pending" {
		t.Errorf("línea de vida tras evaluar el riesgo: %+v", ov.Lifecycle[:3])
	}
	a.call("auditor", "GET", "/api/requirements/REQ-2026-9999/overview", nil, 404, nil)

	// Matriz: una fila por REQ y TC; el REQ sin TC aparece una vez.
	var m matrixBody
	a.call("auditor", "GET", "/api/traceability/matrix?change="+chg.Code, nil, 200, &m)
	if m.Total != 3 || len(m.Items) != 3 || m.Indicators.CoveragePercent != 50 || m.Indicators.BlockedRequirements != 1 ||
		m.Indicators.OpenDefects["high"] != 1 || m.Indicators.OpenDefects["low"] != 0 {
		t.Fatalf("matriz del CHG: total %d, indicadores %+v", m.Total, m.Indicators)
	}
	byTC := map[string]int{}
	for i, item := range m.Items {
		if item.TestCase != nil {
			byTC[item.TestCase.Code] = i
		} else if item.Requirement.Code != bare.Code || len(item.FunctionalCases) != 0 || len(item.Defects) != 0 {
			t.Errorf("la fila sin TC debe ser la del REQ sin pruebas: %+v", item)
		}
	}
	failed := m.Items[byTC[tc2.Code]]
	if failed.Requirement.Code != req.Code || failed.RiskLevel != "high" || !strings.HasPrefix(failed.Risk, "RISK-") ||
		len(failed.FunctionalCases) != 1 || failed.FunctionalCases[0] != uc.Code || failed.TestCase.LastRun.Code != run2.Code ||
		len(failed.Defects) != 1 || failed.Defects[0].Code != def.Code {
		t.Errorf("la fila debe traer la cadena completa: %+v", failed)
	}
	if passed := m.Items[byTC[tc1.Code]]; passed.TestCase.LastRun.Result != "passed" || len(passed.Defects) != 0 {
		t.Errorf("el DEF solo va en la fila de su TC: %+v", passed)
	}
	a.call("auditor", "GET", "/api/traceability/matrix?risk=high&change="+chg.Code, nil, 200, &m)
	if m.Total != 2 {
		t.Errorf("filtrada por riesgo alto: %d filas", m.Total)
	}
	a.call("auditor", "GET", "/api/traceability/matrix?from=2020-01-01&to=2020-12-31", nil, 200, &m)
	if m.Total != 0 || m.Items == nil {
		t.Errorf("fuera del rango de fechas no hay filas: %+v", m)
	}
	a.call("auditor", "GET", "/api/traceability/matrix?from=ayer", nil, 422, nil)

	// Exportación: CSV y XLSX con las mismas filas, sin fórmulas ejecutables.
	rec := a.download("auditor", "/api/traceability/matrix/export?format=csv&change="+chg.Code, 200)
	records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(rec.Body.Bytes(), []byte("\xEF\xBB\xBF")))).ReadAll()
	if err != nil || len(records) != 4 || records[0][0] != "REQ" || len(records[1]) != 19 {
		t.Fatalf("CSV: %v, %d filas", err, len(records))
	}
	var csvRow []string
	for _, r := range records[1:] {
		if r[10] == tc2.Code {
			csvRow = r
		}
	}
	if csvRow == nil || csvRow[0] != req.Code || csvRow[1] != "Conciliación =SUMA(A1)" || csvRow[6] != "Alto" ||
		csvRow[14] != "Fallido" || !strings.Contains(csvRow[18], def.Code+" (alta, Abierto)") {
		t.Errorf("fila del CSV: %v", csvRow)
	}
	a.call("manager", "PATCH", path, map[string]any{"title": "=SUMA(A1) conciliación"}, 200, nil)
	rec = a.download("auditor", "/api/traceability/matrix/export?format=csv&change="+chg.Code, 200)
	if !strings.Contains(rec.Body.String(), "'=SUMA(A1) conciliación") {
		t.Error("un texto que empieza con = debe exportarse neutralizado")
	}
	rec = a.download("auditor", "/api/traceability/matrix/export?format=xlsx&change="+chg.Code, 200)
	book, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("el XLSX no es un zip válido: %v", err)
	}
	var sheet string
	for _, part := range book.File {
		if part.Name == "xl/worksheets/sheet1.xml" {
			r, _ := part.Open()
			raw, _ := io.ReadAll(r)
			sheet = string(raw)
		}
	}
	if len(book.File) != 5 || strings.Count(sheet, "<row>") != 4 || !strings.Contains(sheet, tc2.Code) || !strings.Contains(sheet, "Título del TC") {
		t.Errorf("XLSX: %d partes, %d filas", len(book.File), strings.Count(sheet, "<row>"))
	}
	a.download("auditor", "/api/traceability/matrix/export?format=pdf", 422)

	// Búsqueda global: por ID exacto, por prefijo y por texto; los comodines se buscan como texto.
	var hits struct {
		Items []struct {
			Type  string `json:"type"`
			Code  string `json:"code"`
			State *struct {
				Key string `json:"key"`
			} `json:"state"`
		} `json:"items"`
	}
	a.call("ops", "GET", "/api/search?q="+strings.ToLower(def.Code), nil, 200, &hits)
	if len(hits.Items) == 0 || hits.Items[0].Code != def.Code || hits.Items[0].Type != "defect" || hits.Items[0].State == nil {
		t.Errorf("búsqueda por ID: %+v", hits.Items)
	}
	a.call("ops", "GET", "/api/search?q=descuadre", nil, 200, &hits)
	if len(hits.Items) != 1 || hits.Items[0].Code != def.Code {
		t.Errorf("búsqueda por texto: %+v", hits.Items)
	}
	a.call("ops", "GET", "/api/search?q=No+cuadra", nil, 200, &hits)
	types := map[string]bool{}
	for _, h := range hits.Items {
		types[h.Type] = true
	}
	if !types["test_case"] || !types["test_run"] {
		t.Errorf("la búsqueda cruza entidades: %+v", hits.Items)
	}
	a.call("ops", "GET", "/api/search?q=%25%25", nil, 200, &hits)
	if len(hits.Items) != 0 {
		t.Errorf("%%%% se busca como texto, no como comodín: %d resultados", len(hits.Items))
	}
	a.call("ops", "GET", "/api/search?q=a", nil, 422, nil)

	// Sistemas y comentarios.
	var systems struct {
		Items []struct {
			Key          string `json:"key"`
			DefaultRoute string `json:"defaultRoute"`
		} `json:"items"`
	}
	a.call("ops", "GET", "/api/systems", nil, 200, &systems)
	if len(systems.Items) != 3 || systems.Items[0].Key != "SGLPAR" || systems.Items[0].DefaultRoute != "high" {
		t.Errorf("sistemas: %+v", systems.Items)
	}
	a.call("auditor", "POST", "/api/comments", map[string]any{"entityType": "defect", "entityCode": def.Code, "body": "x"}, 403, nil)
	a.call("dev", "POST", "/api/comments", map[string]any{"entityType": "defect", "entityCode": def.Code, "body": " "}, 422, nil)
	a.call("dev", "POST", "/api/comments", map[string]any{"entityType": "test_case", "entityCode": tc1.Code, "body": "x"}, 422, nil)
	a.call("dev", "POST", "/api/comments", map[string]any{"entityType": "defect", "entityCode": def.Code, "body": "  Reproducido en QA-2  "}, 201, nil)
	var comments struct {
		Items []struct {
			Body   string `json:"body"`
			Author struct {
				DisplayName string `json:"displayName"`
			} `json:"author"`
		} `json:"items"`
	}
	a.call("ops", "GET", "/api/comments?entityType=defect&entityCode="+def.Code, nil, 200, &comments)
	if len(comments.Items) != 1 || comments.Items[0].Body != "Reproducido en QA-2" || comments.Items[0].Author.DisplayName != "J. Díaz" {
		t.Errorf("comentarios: %+v", comments.Items)
	}
}
