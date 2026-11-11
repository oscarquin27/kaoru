//go:build integration

package bootstrap

import (
	"slices"
	"strings"
	"testing"
)

type requirementBody struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	System struct {
		Key string `json:"key"`
	} `json:"system"`
	Change *struct {
		Type  string `json:"type"`
		Code  string `json:"code"`
		Title string `json:"title"`
	} `json:"change"`
	State struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"state"`
	Owner *struct {
		DisplayName string `json:"displayName"`
	} `json:"owner"`
	CreatedBy struct {
		DisplayName string `json:"displayName"`
	} `json:"createdBy"`
}

type riskBody struct {
	Code            string   `json:"code"`
	Level           string   `json:"level"`
	Characteristics []string `json:"characteristics"`
	AssessedBy      struct {
		DisplayName string `json:"displayName"`
	} `json:"assessedBy"`
}

type linkBody struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Sheet *struct {
		ID    string `json:"id"`
		Gid   string `json:"gid"`
		Range string `json:"range"`
	} `json:"sheet"`
	IsOfficial bool `json:"isOfficial"`
	AddedBy    struct {
		DisplayName string `json:"displayName"`
	} `json:"addedBy"`
}

type entryContract struct {
	Items []struct {
		Key       string `json:"key"`
		Done      bool   `json:"done"`
		CheckedBy *struct {
			DisplayName string `json:"displayName"`
		} `json:"checkedBy"`
	} `json:"items"`
}

// El recorrido de un REQ de ruta alta: cada paso exige su rol y sus reglas de bloqueo.
func TestRequirementLifecycleWithBlockingRules(t *testing.T) {
	a := newAPI(t)
	var p problemBody

	var chg changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": "Liberación SIMF", "system": "SIMF"}, 201, &chg)
	if chg.Route != "high" {
		t.Fatalf("un CHG de SIMF nace en ruta alta: %+v", chg)
	}

	// Registrar: solo quien tiene req.create; el CHG debe ser del mismo sistema.
	a.call("qa", "POST", "/api/requirements", map[string]any{"title": "x", "system": "SIMF"}, 403, nil)
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "x", "system": "SYPAGO", "change": chg.Code}, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "change" {
		t.Errorf("CHG de otro sistema: %+v", p)
	}
	var req requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{
		"title": "Validar monto máximo", "acceptanceCriteria": "Rechaza sobre el máximo", "system": "SIMF", "change": chg.Code,
	}, 201, &req)
	if !strings.HasPrefix(req.Code, "REQ-") || req.State.Key != "registered" || req.Change == nil ||
		req.Change.Code != chg.Code || req.Change.Type != "change" || req.CreatedBy.DisplayName != "C. Mora" || req.Owner != nil {
		t.Fatalf("REQ creado inesperado: %+v", req)
	}
	path := "/api/requirements/" + req.Code
	move := func(user, to string, wantStatus int) problemBody {
		t.Helper()
		var out problemBody
		a.call(user, "POST", path+"/transitions", map[string]any{"to": to}, wantStatus, &out)
		return out
	}

	// Riesgo: en ruta alta hace falta el RISK y sus características ISO 25010.
	a.call("qa", "GET", path+"/risk", nil, 404, nil)
	if p = move("qa", "risk_assessed", 409); p.Code != "INVALID_ACTION" || !strings.Contains(p.Detail, "Falta evaluar el riesgo.") ||
		!strings.Contains(p.Detail, "ISO 25010") {
		t.Errorf("sin RISK debe decir qué falta: %+v", p)
	}
	a.call("dev", "PUT", path+"/risk", map[string]any{"level": "high", "justification": "Toca saldos"}, 403, nil)
	a.call("qa", "PUT", path+"/risk", map[string]any{"level": "alto"}, 422, &p)
	if len(p.Errors) != 2 {
		t.Errorf("RISK inválido: %+v", p.Errors)
	}
	var risk riskBody
	a.call("qa", "PUT", path+"/risk", map[string]any{"level": "high", "justification": "Toca saldos"}, 200, &risk)
	if !strings.HasPrefix(risk.Code, "RISK-") || risk.Level != "high" || risk.Characteristics == nil || risk.AssessedBy.DisplayName != "A. Rojas" {
		t.Errorf("RISK inesperado: %+v", risk)
	}
	if p = move("qa", "risk_assessed", 409); strings.Contains(p.Detail, "Falta evaluar el riesgo.") || !strings.Contains(p.Detail, "ISO 25010") {
		t.Errorf("con RISK sin características solo falta eso: %+v", p)
	}
	first := risk.Code
	a.call("qa", "PUT", path+"/risk", map[string]any{
		"level": "high", "justification": "Toca saldos", "characteristics": []string{"security", "reliability"},
	}, 200, &risk)
	if risk.Code != first || !slices.Equal(risk.Characteristics, []string{"reliability", "security"}) {
		t.Errorf("reevaluar conserva el RISK y actualiza sus características: %+v", risk)
	}
	a.call("ops", "GET", path+"/risk", nil, 200, &risk)
	move("dev", "risk_assessed", 403)
	move("qa", "risk_assessed", 200)
	move("qa", "test_design", 200)
	move("qa", "in_development", 403)
	move("dev", "in_development", 200)

	// Contrato de entrada: sin todos los ítems, dev no entrega a QA.
	if p = move("dev", "delivered_to_qa", 409); !strings.Contains(p.Detail, "Faltan 9 ítems del contrato de entrada") {
		t.Errorf("sin contrato de entrada: %+v", p)
	}
	var contract entryContract
	a.call("dev", "GET", path+"/entry-contract", nil, 200, &contract)
	if len(contract.Items) != 9 {
		t.Fatalf("el contrato trae %d ítems", len(contract.Items))
	}
	a.call("qa", "PUT", path+"/entry-contract/ids", map[string]any{}, 403, nil)
	a.call("dev", "PUT", path+"/entry-contract/inventado", map[string]any{}, 404, nil)
	for _, item := range contract.Items[1:] {
		a.call("dev", "PUT", path+"/entry-contract/"+item.Key, map[string]any{"note": "ok"}, 200, nil)
	}
	if p = move("dev", "delivered_to_qa", 409); !strings.Contains(p.Detail, "Falta 1 ítem del contrato de entrada: IDs de REQ y CHG.") {
		t.Errorf("con un ítem pendiente debe nombrarlo: %+v", p)
	}
	a.call("dev", "PUT", path+"/entry-contract/ids", map[string]any{}, 200, nil)
	a.call("dev", "DELETE", path+"/entry-contract/rollback", nil, 204, nil)
	a.call("dev", "PUT", path+"/entry-contract/rollback", map[string]any{}, 200, nil)
	a.call("dev", "GET", path+"/entry-contract", nil, 200, &contract)
	if !contract.Items[0].Done || contract.Items[0].CheckedBy == nil || contract.Items[0].CheckedBy.DisplayName != "J. Díaz" {
		t.Errorf("el ítem debe decir quién lo marcó: %+v", contract.Items[0])
	}
	move("dev", "delivered_to_qa", 200)
	move("qa", "in_testing", 200)

	// Aprobar: todavía no hay TC ni RUN, así que la cobertura bloquea.
	if p = move("qa", "approved_qa", 409); !strings.Contains(p.Detail, "El REQ no tiene TC definidos.") {
		t.Errorf("sin TC no se aprueba: %+v", p)
	}
	a.call("qa", "GET", path, nil, 200, &req)
	if req.State.Key != "in_testing" || req.State.Name != "En pruebas" {
		t.Errorf("estado final inesperado: %+v", req.State)
	}

	// La auditoría del REQ trae cada paso, y la cadena sigue íntegra.
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?size=100&entityCode="+req.Code, nil, 200, &audit)
	var transitions []string
	for i := len(audit.Items) - 1; i >= 0; i-- {
		if audit.Items[i].Action == "transition" {
			transitions = append(transitions, audit.Items[i].NewValue)
		}
	}
	if !slices.Equal(transitions, []string{"risk_assessed", "test_design", "in_development", "delivered_to_qa", "in_testing"}) {
		t.Errorf("transiciones auditadas = %v", transitions)
	}
	a.call("auditor", "GET", "/api/audit?entityCode="+risk.Code, nil, 200, &audit)
	if audit.Total != 2 || audit.Items[0].Field != "characteristics" || audit.Items[0].NewValue != "reliability,security" {
		t.Errorf("auditoría del RISK: %+v", audit.Items)
	}
	var v verification
	a.call("auditor", "GET", "/api/audit/verification", nil, 200, &v)
	if !v.Intact {
		t.Errorf("la cadena de auditoría debe seguir íntegra: %+v", v)
	}
}

func TestRequirementLinksAndEdits(t *testing.T) {
	a := newAPI(t)
	var p problemBody

	var chg changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": "Pagos", "system": "SYPAGO"}, 201, &chg)
	var req requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Límite diario", "system": "SYPAGO", "change": chg.Code}, 201, &req)
	path := "/api/requirements/" + req.Code

	// Enlace de Google Sheets: se guarda estructurado y con etiqueta legible. El servidor no abre la URL.
	const sheet = "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789/edit#gid=1842&range=A1:F40"
	a.call("auditor", "POST", path+"/links", map[string]any{"url": sheet}, 403, nil)
	a.call("qa", "POST", path+"/links", map[string]any{"url": "javascript:alert(1)"}, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "url" {
		t.Errorf("URL peligrosa: %+v", p)
	}
	var link linkBody
	a.call("qa", "POST", path+"/links", map[string]any{"url": sheet}, 201, &link)
	if link.Kind != "google_sheets" || link.Label != "Google Sheets · pestaña 1842 · A1:F40" || link.Sheet == nil ||
		link.Sheet.ID != "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789" || link.Sheet.Gid != "1842" || link.Sheet.Range != "A1:F40" ||
		link.AddedBy.DisplayName != "A. Rojas" {
		t.Fatalf("enlace inesperado: %+v", link)
	}
	var other linkBody
	a.call("dev", "POST", path+"/links", map[string]any{"url": "https://youtrack.lab.invalid/issue/SY-12", "label": "Incidencia"}, 201, &other)
	if other.Kind != "youtrack" || other.Sheet != nil || other.Label != "Incidencia" {
		t.Errorf("enlace de YouTrack inesperado: %+v", other)
	}

	a.call("qa", "PATCH", path+"/links/"+link.ID, map[string]any{"isOfficial": true, "label": "Especificación SyPago"}, 200, &link)
	if !link.IsOfficial || link.Label != "Especificación SyPago" {
		t.Errorf("edición del enlace: %+v", link)
	}
	a.call("qa", "PATCH", path+"/links/"+link.ID, map[string]any{"snapshotDocument": "DOC-2026-9999"}, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "snapshotDocument" {
		t.Errorf("instantánea inexistente: %+v", p)
	}
	a.call("qa", "PATCH", path+"/links/no-es-un-id", map[string]any{"isOfficial": true}, 404, nil)

	var links struct {
		Items []linkBody `json:"items"`
	}
	a.call("ops", "GET", path+"/links", nil, 200, &links)
	if len(links.Items) != 2 || links.Items[0].ID != link.ID {
		t.Errorf("lista de enlaces: %+v", links.Items)
	}
	a.call("qa", "DELETE", path+"/links/"+other.ID, nil, 204, nil)
	a.call("qa", "DELETE", path+"/links/"+other.ID, nil, 404, nil)

	// Editar: responsable y CHG se pueden asignar y quitar.
	a.call("auditor", "PATCH", path, map[string]any{"title": "x"}, 403, nil)
	a.call("qa", "PATCH", path, map[string]any{"owner": "sub-qa", "title": "Límite diario por ordenante"}, 200, &req)
	if req.Owner == nil || req.Owner.DisplayName != "A. Rojas" || req.Title != "Límite diario por ordenante" {
		t.Errorf("edición del REQ: %+v", req)
	}
	a.call("qa", "PATCH", path, map[string]any{"owner": "nadie"}, 422, nil)

	var page struct {
		Total int               `json:"total"`
		Items []requirementBody `json:"items"`
	}
	a.call("ops", "GET", "/api/requirements?owner=sub-qa&change="+chg.Code, nil, 200, &page)
	if page.Total != 1 || page.Items[0].Code != req.Code {
		t.Errorf("lista filtrada por CHG y responsable: %+v", page)
	}
	req = requirementBody{}
	a.call("qa", "PATCH", path, map[string]any{"change": nil, "owner": nil}, 200, &req)
	if req.Change != nil || req.Owner != nil {
		t.Errorf("null debe quitar el CHG y el responsable: %+v", req)
	}
	a.call("ops", "GET", "/api/requirements?change="+chg.Code, nil, 200, &page)
	if page.Total != 0 {
		t.Errorf("el REQ ya no pertenece al CHG: %+v", page)
	}
	a.call("ops", "GET", "/api/requirements/REQ-2026-9999", nil, 404, nil)

	// Quitar un enlace deja su URL en la auditoría.
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?size=100&entityCode="+req.Code, nil, 200, &audit)
	found := false
	for _, item := range audit.Items {
		found = found || (item.Action == "delete" && item.Field == "link" && item.OldValue == "https://youtrack.lab.invalid/issue/SY-12")
	}
	if !found {
		t.Errorf("falta el enlace quitado en la auditoría: %+v", audit.Items)
	}
}
