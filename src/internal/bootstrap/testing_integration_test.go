//go:build integration

package bootstrap

import (
	"strings"
	"testing"
	"time"
)

type codeBody struct {
	Code string `json:"code"`
}

type defectBody struct {
	Code  string `json:"code"`
	State struct {
		Key string `json:"key"`
	} `json:"state"`
	Severity    string  `json:"severity"`
	DaysInState float64 `json:"daysInState"`
	Run         string  `json:"run"`
	Assignee    *struct {
		DisplayName string `json:"displayName"`
	} `json:"assignee"`
}

type totalBody struct {
	Total int `json:"total"`
}

// De «En pruebas» a «Aprobado QA»: hacen falta TC con RUN aprobado y ningún DEF alto abierto.
func TestRequirementApprovalNeedsCoverageAndNoHighDefects(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	build, environment := a.catalog("SYPAGO")
	executedAt := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	// Un REQ de SyPago (ruta ligera) llevado hasta «En pruebas».
	var req requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Límite diario", "system": "SYPAGO"}, 201, &req)
	path := "/api/requirements/" + req.Code
	move := func(user, to string, wantStatus int) problemBody {
		t.Helper()
		var out problemBody
		a.call(user, "POST", path+"/transitions", map[string]any{"to": to}, wantStatus, &out)
		return out
	}
	a.call("qa", "PUT", path+"/risk", map[string]any{"level": "low", "justification": "Módulo estable"}, 200, nil)
	move("qa", "risk_assessed", 200) // ruta ligera: el RISK no necesita características
	move("qa", "test_design", 200)
	move("dev", "in_development", 200)
	var contract entryContract
	a.call("dev", "GET", path+"/entry-contract", nil, 200, &contract)
	for _, item := range contract.Items {
		a.call("dev", "PUT", path+"/entry-contract/"+item.Key, map[string]any{}, 200, nil)
	}
	move("dev", "delivered_to_qa", 200)
	move("qa", "in_testing", 200)

	// Diseño de pruebas: UC y TC.
	a.call("dev", "POST", path+"/functional-cases", map[string]any{"title": "x"}, 403, nil)
	a.call("qa", "POST", path+"/functional-cases", map[string]any{}, 422, nil)
	var uc codeBody
	a.call("qa", "POST", path+"/functional-cases", map[string]any{"title": "Pago sobre el límite"}, 201, &uc)
	if !strings.HasPrefix(uc.Code, "UC-") {
		t.Fatalf("UC inesperado: %+v", uc)
	}
	tcBody := func(title string, extra map[string]any) map[string]any {
		body := map[string]any{"title": title, "system": "SYPAGO", "steps": "1. Pagar", "expectedResult": "Rechazado", "functionalCases": []string{uc.Code}}
		for k, v := range extra {
			body[k] = v
		}
		return body
	}
	a.call("qa", "POST", "/api/test-cases", tcBody("x", map[string]any{"functionalCases": []string{}}), 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "functionalCases" || p.Errors[0].Code != "required" {
		t.Errorf("TC sin UC: %+v", p)
	}
	a.call("qa", "POST", "/api/test-cases", tcBody("x", map[string]any{"functionalCases": []string{"UC-2026-9999"}}), 422, nil)
	a.call("qa", "POST", "/api/test-cases", tcBody("x", map[string]any{"system": "SIMF"}), 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "functionalCases" || p.Errors[0].Code != "invalid" {
		t.Errorf("TC de un sistema con UC de otro: %+v", p)
	}
	var tc1, tc2, tc3 codeBody
	a.call("qa", "POST", "/api/test-cases", tcBody("Rechaza sobre el límite", map[string]any{"isCritical": true}), 201, &tc1)
	a.call("qa", "POST", "/api/test-cases", tcBody("Acepta bajo el límite", nil), 201, &tc2)
	a.call("qa", "POST", "/api/test-cases", tcBody("Caso obsoleto", nil), 201, &tc3)
	a.call("qa", "PATCH", "/api/test-cases/"+tc3.Code, map[string]any{"retired": true}, 200, nil)

	var total totalBody
	a.call("ops", "GET", "/api/test-cases?requirement="+req.Code, nil, 200, &total)
	if total.Total != 3 {
		t.Errorf("TC del REQ = %d, quiero 3", total.Total)
	}
	a.call("ops", "GET", "/api/test-cases?critical=true&requirement="+req.Code, nil, 200, &total)
	if total.Total != 1 {
		t.Errorf("TC críticos del REQ = %d, quiero 1", total.Total)
	}
	var ucRead struct {
		TestCases []string `json:"testCases"`
	}
	a.call("ops", "GET", "/api/functional-cases/"+uc.Code, nil, 200, &ucRead)
	if len(ucRead.TestCases) != 3 {
		t.Errorf("el UC debe listar sus TC: %+v", ucRead)
	}

	// Sin RUN no hay cobertura. El TC retirado no se exige.
	if p = move("qa", "approved_qa", 409); !strings.Contains(p.Detail, "Faltan 2 TC sin RUN aprobado") ||
		!strings.Contains(p.Detail, tc1.Code) || strings.Contains(p.Detail, tc3.Code) {
		t.Errorf("sin RUN: %+v", p)
	}

	// RUN: solo QA, sobre un BUILD y un ENV del catálogo, y nunca con fecha futura ni sobre un TC retirado.
	run := func(tc, result string) map[string]any {
		return map[string]any{"testCase": tc, "build": build, "environment": environment, "result": result, "executedAt": executedAt}
	}
	a.call("dev", "POST", "/api/runs", run(tc1.Code, "passed"), 403, nil)
	bad := run(tc1.Code, "passed")
	bad["result"], bad["executedAt"] = "aprobado", time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	a.call("qa", "POST", "/api/runs", bad, 422, &p)
	if len(p.Errors) != 2 {
		t.Errorf("RUN inválido: %+v", p.Errors)
	}
	a.call("qa", "POST", "/api/runs", run("TC-2026-9999", "passed"), 422, nil)
	a.call("qa", "POST", "/api/runs", run(tc3.Code, "passed"), 409, &p)
	if p.Code != "INVALID_ACTION" {
		t.Errorf("RUN sobre un TC retirado: %+v", p)
	}
	var run1, run2 codeBody
	a.call("qa", "POST", "/api/runs", run(tc1.Code, "passed"), 201, &run1)
	a.call("qa", "POST", "/api/runs", run(tc2.Code, "failed"), 201, &run2)
	a.call("qa", "PATCH", "/api/runs/"+run1.Code, map[string]any{"result": "failed"}, 404, nil) // un RUN no se edita
	a.call("qa", "DELETE", "/api/runs/"+run1.Code, nil, 404, nil)                               // ni se borra

	if p = move("qa", "approved_qa", 409); !strings.Contains(p.Detail, "Falta 1 TC sin RUN aprobado: "+tc2.Code) {
		t.Errorf("con un TC fallido: %+v", p)
	}

	// DEF: necesita el RUN que lo reprodujo.
	defect := func(runCode, severity string) map[string]any {
		return map[string]any{"title": "Acepta sobre el límite", "description": "Pasos…", "requirement": req.Code, "run": runCode, "severity": severity}
	}
	a.call("dev", "POST", "/api/defects", defect(run2.Code, "high"), 403, nil)
	a.call("qa", "POST", "/api/defects", defect(run1.Code, "high"), 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "run" {
		t.Errorf("un RUN aprobado no sirve para abrir un DEF: %+v", p)
	}
	// El RUN debe ser de un TC de ese REQ: no vale la evidencia de otro requerimiento.
	var other requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Otro REQ", "system": "SYPAGO"}, 201, &other)
	wrong := defect(run2.Code, "high")
	wrong["requirement"] = other.Code
	a.call("qa", "POST", "/api/defects", wrong, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "run" {
		t.Errorf("DEF con el RUN de otro REQ: %+v", p)
	}
	var high, low defectBody
	a.call("qa", "POST", "/api/defects", defect(run2.Code, "high"), 201, &high)
	a.call("qa", "POST", "/api/defects", defect(run2.Code, "low"), 201, &low)
	if !strings.HasPrefix(high.Code, "DEF-") || high.State.Key != "open" || high.Run != run2.Code {
		t.Fatalf("DEF inesperado: %+v", high)
	}
	move("qa", "with_defects", 200)
	move("qa", "in_testing", 200)

	transition := func(user, code, to, reason string, wantStatus int) problemBody {
		t.Helper()
		var out problemBody
		a.call(user, "POST", "/api/defects/"+code+"/transitions", map[string]any{"to": to, "reason": reason}, wantStatus, &out)
		return out
	}
	// Diferir: solo change management, con motivo, y nunca un DEF de severidad alta.
	transition("qa", high.Code, "triaged", "", 200)
	transition("qa", low.Code, "triaged", "", 200)
	transition("qa", low.Code, "deferred", "Cosmético", 403)
	if p = transition("manager", low.Code, "deferred", "", 422); len(p.Errors) != 1 || p.Errors[0].Field != "reason" {
		t.Errorf("diferir exige motivo: %+v", p)
	}
	if p = transition("manager", high.Code, "deferred", "Lo pide negocio", 409); !strings.Contains(p.Detail, "severidad alta") {
		t.Errorf("un DEF alto no se difiere: %+v", p)
	}
	transition("manager", low.Code, "deferred", "Cosmético; va al siguiente paquete", 200)

	// Dev corrige, QA reprueba con un RUN nuevo.
	a.call("dev", "PATCH", "/api/defects/"+high.Code, map[string]any{"assignee": "sub-dev"}, 403, nil)
	a.call("qa", "PATCH", "/api/defects/"+high.Code, map[string]any{"assignee": "sub-dev"}, 200, &high)
	if high.Assignee == nil || high.Assignee.DisplayName != "J. Díaz" {
		t.Errorf("asignación: %+v", high)
	}
	transition("qa", high.Code, "in_fix", "", 403)
	transition("dev", high.Code, "in_fix", "", 200)
	transition("dev", high.Code, "ready_for_retest", "", 200)
	a.call("qa", "POST", "/api/runs", run(tc2.Code, "passed"), 201, nil)

	// La cobertura ya está, pero el DEF alto sigue abierto. El diferido bajo no bloquea.
	if p = move("qa", "approved_qa", 409); !strings.Contains(p.Detail, "Hay 1 DEF abierto de severidad alta: "+high.Code) ||
		strings.Contains(p.Detail, "TC") {
		t.Errorf("con un DEF alto abierto: %+v", p)
	}
	transition("dev", high.Code, "closed", "", 403)
	transition("qa", high.Code, "closed", "", 200)
	move("qa", "approved_qa", 200)

	a.call("ops", "GET", path, nil, 200, &req)
	if req.State.Key != "approved_qa" {
		t.Fatalf("el REQ debía quedar aprobado: %+v", req.State)
	}

	// Lecturas: RUN por REQ y por TC; DEF con su tiempo en estado; el motivo del diferido en la auditoría.
	a.call("ops", "GET", "/api/runs?requirement="+req.Code, nil, 200, &total)
	if total.Total != 3 {
		t.Errorf("RUN del REQ = %d, quiero 3", total.Total)
	}
	a.call("ops", "GET", "/api/runs?testCase="+tc2.Code+"&requirement="+req.Code, nil, 200, &total)
	if total.Total != 2 {
		t.Errorf("RUN del TC = %d, quiero 2", total.Total)
	}
	a.call("ops", "GET", "/api/runs?requirement=REQ-2026-9999", nil, 200, &total)
	if total.Total != 0 {
		t.Errorf("RUN de un REQ que no existe = %d", total.Total)
	}
	var defects struct {
		Total int          `json:"total"`
		Items []defectBody `json:"items"`
	}
	a.call("ops", "GET", "/api/defects?state=deferred&requirement="+req.Code, nil, 200, &defects)
	if defects.Total != 1 || defects.Items[0].Code != low.Code || defects.Items[0].DaysInState < 0 || defects.Items[0].DaysInState > 1 {
		t.Errorf("DEF diferidos: %+v", defects)
	}
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+low.Code, nil, 200, &audit)
	if audit.Items[0].Action != "transition" || audit.Items[0].NewValue != "deferred" ||
		audit.Items[0].Reason != "Cosmético; va al siguiente paquete" || audit.Items[0].Actor.DisplayName != "C. Mora" {
		t.Errorf("quién aceptó diferir y por qué debe quedar en la auditoría: %+v", audit.Items[0])
	}

	// EXP: se registra, pero no cuenta como evidencia.
	var exp codeBody
	a.call("qa", "POST", "/api/explorations", map[string]any{
		"requirement": req.Code, "build": build, "environment": environment, "manualChanges": "Reloj adelantado", "executedAt": executedAt,
	}, 201, &exp)
	a.call("qa", "PATCH", "/api/explorations/"+exp.Code, map[string]any{"findings": "El corte de día falla"}, 200, nil)
	a.call("dev", "PATCH", "/api/explorations/"+exp.Code, map[string]any{"findings": "x"}, 403, nil)
	a.call("ops", "GET", "/api/explorations?requirement="+req.Code, nil, 200, &total)
	if total.Total != 1 || !strings.HasPrefix(exp.Code, "EXP-") {
		t.Errorf("EXP del REQ: %+v %+v", total, exp)
	}
}
