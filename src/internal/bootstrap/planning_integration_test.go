//go:build integration

package bootstrap

import (
	"slices"
	"strings"
	"testing"
	"time"
)

type activityBody struct {
	Code   string `json:"code"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Change *struct {
		Code   string `json:"code"`
		System string `json:"system"`
	} `json:"change"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Operators []struct {
		Id          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"operators"`
	Overlaps []struct {
		Operator struct {
			Id string `json:"id"`
		} `json:"operator"`
		Activity codeBody `json:"activity"`
		Other    codeBody `json:"other"`
		From     string   `json:"from"`
		To       string   `json:"to"`
		Days     int      `json:"days"`
	} `json:"overlaps"`
}

type activityPage struct {
	Total int            `json:"total"`
	Items []activityBody `json:"items"`
}

type scheduleBody struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Operators []struct {
		Operator struct {
			Id string `json:"id"`
		} `json:"operator"`
		Activities []activityBody `json:"activities"`
		Overlaps   []struct {
			Activity codeBody `json:"activity"`
			Other    codeBody `json:"other"`
			Days     int      `json:"days"`
		} `json:"overlaps"`
	} `json:"operators"`
	Projects []struct {
		Change struct {
			Code string `json:"code"`
		} `json:"change"`
		Certifications []activityBody `json:"certifications"`
	} `json:"projects"`
}

type overviewMissing struct {
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
}

// certificationWarning devuelve el aviso de la regla certification_planned al pasar a pruebas, o "".
func (a *api) certificationWarning(req string) string {
	a.t.Helper()
	var o overviewMissing
	a.call("qa", "GET", "/api/requirements/"+req+"/overview", nil, 200, &o)
	for _, m := range o.Missing {
		for _, c := range m.Conditions {
			if m.To.Key == "in_testing" && c.RuleKey == "certification_planned" {
				if c.Blocks {
					a.t.Errorf("la regla debe solo avisar: %+v", c)
				}
				return c.Message
			}
		}
	}
	return ""
}

// La planificación: el período de certificación de un CHG reservado con sus operadores, el aviso al
// pasar a pruebas sin él, los solapes de un operador y la planificación del rango.
func TestPlanningCertificationOperatorsAndOverlaps(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	today := time.Now().UTC()
	day := func(offset int) string { return today.AddDate(0, 0, offset).Format(time.DateOnly) }

	// Un CHG con un REQ entregado a QA. Llevarlo hasta ahí lo prueban otras pruebas; aquí se pone directo.
	var chg, other changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": a.uniqueName("Planificación"), "system": "SYPAGO"}, 201, &chg)
	a.call("manager", "POST", "/api/changes", map[string]any{"title": a.uniqueName("Sin planificar"), "system": "SYPAGO"}, 201, &other)
	var req, unplanned requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Tope diario", "system": "SYPAGO", "change": chg.Code}, 201, &req)
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Sin plan", "system": "SYPAGO", "change": other.Code}, 201, &unplanned)
	for _, code := range []string{req.Code, unplanned.Code} {
		a.asOwner(`UPDATE requirement SET state_id = (SELECT id FROM workflow_state WHERE entity_type = 'requirement' AND key = 'delivered_to_qa')
			WHERE code = '` + code + `'`)
	}
	a.call("ops", "GET", "/api/me", nil, 200, nil) // el operador existe en Kaoru desde que entra

	if msg := a.certificationWarning(req.Code); msg != chg.Code+" no tiene reservado el período de certificación. Planifícalo en Planificación." {
		t.Errorf("sin plan debe avisar: %q", msg)
	}
	// Solo avisa: el REQ del otro CHG entra en pruebas igual.
	a.call("qa", "POST", "/api/requirements/"+unplanned.Code+"/transitions", map[string]any{"to": "in_testing"}, 200, nil)

	// Planificar: solo quien tiene activity.manage, con CHG y operadores que existan.
	cert := map[string]any{"kind": "certification", "title": "Certificación " + chg.Code, "startDate": day(-2), "endDate": day(5)}
	a.call("qa", "POST", "/api/activities", cert, 403, nil)
	a.call("admin", "POST", "/api/activities", cert, 422, &p)
	if !hasField(p, "change", "required") {
		t.Errorf("la certificación exige CHG: %+v", p)
	}
	cert["change"], cert["operators"] = chg.Code, []string{"sub-nadie"}
	if a.call("admin", "POST", "/api/activities", cert, 422, &p); !hasField(p, "operators", "invalid") {
		t.Errorf("operador inexistente: %+v", p)
	}
	cert["operators"] = []string{}
	var c activityBody
	a.call("admin", "POST", "/api/activities", cert, 201, &c)
	if !strings.HasPrefix(c.Code, "ACT-") || c.Status != "active" || c.Change == nil || c.Change.Code != chg.Code || c.Change.System != "SyPago" {
		t.Fatalf("certificación inesperada: %+v", c)
	}
	if msg := a.certificationWarning(req.Code); msg != "La certificación de "+chg.Code+" no tiene operadores asignados." {
		t.Errorf("sin operadores debe avisar: %q", msg)
	}
	a.call("manager", "PATCH", "/api/activities/"+c.Code, map[string]any{"operators": []string{"sub-ops", "sub-qa"}}, 200, &c)
	if len(c.Operators) != 2 || c.Operators[0].DisplayName != "A. Rojas" {
		t.Errorf("operadores por nombre: %+v", c.Operators)
	}
	if msg := a.certificationWarning(req.Code); msg != "" {
		t.Errorf("con período y operadores no avisa: %q", msg)
	}
	a.call("qa", "POST", "/api/requirements/"+req.Code+"/transitions", map[string]any{"to": "in_testing"}, 200, nil)

	// Una regresión de Luis que pisa los dos últimos días de la certificación: se guarda y avisa.
	var r activityBody
	a.call("admin", "POST", "/api/activities", map[string]any{
		"kind": "regression", "title": "Regresión SIMF", "startDate": day(4), "endDate": day(9), "operators": []string{"sub-ops"},
	}, 201, &r)
	if len(r.Overlaps) != 1 || r.Overlaps[0].Other.Code != c.Code || r.Overlaps[0].Days != 2 || r.Overlaps[0].From != day(4) || r.Overlaps[0].To != day(5) {
		t.Errorf("solape inesperado: %+v", r.Overlaps)
	}
	if r.Status != "planned" {
		t.Errorf("empieza en cuatro días: %s", r.Status)
	}

	// La planificación del rango, filtrada por Luis.
	var s scheduleBody
	a.call("auditor", "GET", "/api/planning/schedule?operator=sub-ops&from="+day(-3)+"&to="+day(10), nil, 200, &s)
	if len(s.Operators) != 1 || s.Operators[0].Operator.Id != "sub-ops" {
		t.Fatalf("filas: %+v", s.Operators)
	}
	var codes []string
	for _, act := range s.Operators[0].Activities {
		codes = append(codes, act.Code)
	}
	overlapped := slices.ContainsFunc(s.Operators[0].Overlaps, func(o struct {
		Activity codeBody `json:"activity"`
		Other    codeBody `json:"other"`
		Days     int      `json:"days"`
	}) bool {
		return o.Activity.Code == c.Code && o.Other.Code == r.Code && o.Days == 2
	})
	if !slices.Contains(codes, c.Code) || !slices.Contains(codes, r.Code) || !overlapped {
		t.Errorf("Luis debe tener las dos y su solape: %v %+v", codes, s.Operators[0].Overlaps)
	}
	if !slices.ContainsFunc(s.Projects, func(pr struct {
		Change struct {
			Code string `json:"code"`
		} `json:"change"`
		Certifications []activityBody `json:"certifications"`
	}) bool {
		return pr.Change.Code == chg.Code && len(pr.Certifications) == 1
	}) {
		t.Errorf("el CHG debe salir entre los proyectos: %+v", s.Projects)
	}
	a.call("qa", "GET", "/api/planning/schedule?from="+day(10)+"&to="+day(0), nil, 422, &p)
	if !hasField(p, "to", "invalid") {
		t.Errorf("rango al revés: %+v", p)
	}

	// Reajustar: cancelar la regresión quita el solape.
	a.call("qa", "PATCH", "/api/activities/"+r.Code, map[string]any{"cancelled": true}, 403, nil)
	a.call("admin", "PATCH", "/api/activities/"+r.Code, map[string]any{"cancelled": true}, 200, &r)
	a.call("ops", "GET", "/api/activities/"+c.Code, nil, 200, &c)
	if r.Status != "cancelled" || len(c.Overlaps) != 0 {
		t.Errorf("cancelada %s, solapes que quedan %+v", r.Status, c.Overlaps)
	}
	a.call("ops", "GET", "/api/activities/ACT-1999-0001", nil, 404, nil)

	var page activityPage
	a.call("ops", "GET", "/api/activities?change="+chg.Code, nil, 200, &page)
	if page.Total != 1 || page.Items[0].Code != c.Code {
		t.Errorf("del CHG: %+v", page)
	}
	a.call("ops", "GET", "/api/activities?operator=sub-ops&status=cancelled", nil, 200, &page)
	if !slices.ContainsFunc(page.Items, func(x activityBody) bool { return x.Code == r.Code }) {
		t.Errorf("la cancelada debe salir con su filtro: %+v", page)
	}
	a.call("ops", "GET", "/api/activities?status=pronto", nil, 422, &p)

	// La auditoría de la certificación: alta y cambio de operadores, con su código.
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+c.Code, nil, 200, &audit)
	var fields []string
	for i := len(audit.Items) - 1; i >= 0; i-- {
		fields = append(fields, audit.Items[i].Action+":"+audit.Items[i].Field)
	}
	if !slices.Equal(fields, []string{"create:dates", "update:operators"}) {
		t.Errorf("auditoría: %v", fields)
	}
}
