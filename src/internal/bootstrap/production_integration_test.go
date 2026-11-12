//go:build integration

package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"
	"time"
)

type checklistBody struct {
	Code            string `json:"code"`
	TemplateVersion int    `json:"templateVersion"`
	Steps           []struct {
		Position  int `json:"position"`
		Signature *struct {
			SignedBy struct {
				DisplayName string `json:"displayName"`
			} `json:"signedBy"`
			Comment string `json:"comment"`
		} `json:"signature"`
	} `json:"steps"`
	SignedSteps int  `json:"signedSteps"`
	Complete    bool `json:"complete"`
}

type diagnosticBody struct {
	Code   string `json:"code"`
	Review *struct {
		Result     string `json:"result"`
		ReviewedBy struct {
			DisplayName string `json:"displayName"`
		} `json:"reviewedBy"`
	} `json:"review"`
}

type reportBody struct {
	Code    string `json:"code"`
	Package struct {
		Code string `json:"code"`
	} `json:"package"`
	Version      int    `json:"version"`
	Previous     string `json:"previous"`
	SupersededBy string `json:"supersededBy"`
	Checksum     string `json:"checksum"`
	IssuedBy     struct {
		DisplayName string `json:"displayName"`
	} `json:"issuedBy"`
	Content string `json:"content"`
}

// Un PKG de la ruta alta en producción: el checklist con firmas lo deja desplegar, el DIAG revisado por QA
// lo deja verificar, y el informe de conformidad lo cuenta todo, inmutable y versionado.
func TestProductionChecklistVerificationAndReport(t *testing.T) {
	a := newAPI(t)
	var p problemBody

	// Un PKG de SIMF sin REQ: su ruta es la del sistema, alta. Se pone en «Aprobado para producción»
	// como dueño de la base, porque llegar hasta ahí ya lo prueba TestPackageLifecycleWithRegressionByImpact.
	var pkg packageBody
	a.call("dev", "POST", "/api/packages", map[string]any{"title": a.uniqueName("Mensajería"), "system": "SIMF"}, 201, &pkg)
	build, testEnv := a.catalog("SIMF")
	path := "/api/packages/" + pkg.Code
	a.call("dev", "PUT", path+"/builds/"+build, nil, 200, nil)
	a.asOwner(`UPDATE package SET state_id = (SELECT id FROM workflow_state WHERE entity_type = 'package' AND key = 'approved_for_production')
WHERE code = '` + pkg.Code + `'`)
	read := func() packageBody {
		t.Helper()
		var out packageBody
		a.call("ops", "GET", path, nil, 200, &out)
		return out
	}
	move := func(user, to string, wantStatus int) problemBody {
		t.Helper()
		var out problemBody
		a.call(user, "POST", path+"/transitions", map[string]any{"to": to}, wantStatus, &out)
		return out
	}
	var route struct {
		Route string `json:"route"`
	}
	a.call("ops", "GET", path, nil, 200, &route)
	if got := read().rules("deployed"); route.Route != "high" || !slices.Equal(got, []string{"checklist_signed", "deployed_to_production"}) {
		t.Errorf("ruta %q; qué falta para desplegado: %v", route.Route, got)
	}

	// --- Plantilla versionada: la publica change management. ---
	steps := []map[string]any{{"title": "Respaldar la base"}, {"title": "Desplegar el paquete", "description": "Con el instalador."}}
	a.call("qa", "POST", "/api/checklist-templates", map[string]any{"system": "SIMF", "steps": steps}, 403, nil)
	if a.call("manager", "POST", "/api/checklist-templates", map[string]any{"system": "SIMF", "steps": []any{}}, 422, &p); !hasField(p, "steps", "required") {
		t.Errorf("plantilla sin pasos: %+v", p)
	}
	var first, second struct {
		Version int `json:"version"`
	}
	a.call("manager", "POST", "/api/checklist-templates", map[string]any{"system": "SIMF", "steps": steps[:1]}, 201, &first)
	a.call("manager", "POST", "/api/checklist-templates", map[string]any{"system": "SIMF", "steps": steps}, 201, &second)
	if second.Version != first.Version+1 {
		t.Errorf("versiones de la plantilla: %d y %d", first.Version, second.Version)
	}

	// --- El checklist se abre en un ambiente de producción del sistema, con la última versión. ---
	var prod codeBody
	a.call("qa", "POST", "/api/environments", map[string]any{"system": "SIMF", "name": a.uniqueName("Producción SIMF"), "kind": "production", "os": "linux"}, 201, &prod)
	a.call("ops", "POST", "/api/checklists", map[string]any{"package": pkg.Code, "environment": prod.Code}, 403, nil)
	if a.call("manager", "POST", "/api/checklists", map[string]any{"package": pkg.Code, "environment": testEnv}, 422, &p); !hasField(p, "environment", "invalid") {
		t.Errorf("checklist en un ambiente de prueba: %+v", p)
	}
	var chk checklistBody
	a.call("manager", "POST", "/api/checklists", map[string]any{"package": pkg.Code, "environment": prod.Code}, 201, &chk)
	if len(chk.Steps) != 2 || chk.TemplateVersion != second.Version || chk.Complete {
		t.Fatalf("CHK abierto: %+v", chk)
	}

	// --- Desplegado: el despliegue solo no basta; faltan las firmas. ---
	var dep deploymentBody
	a.call("manager", "POST", "/api/deployments", map[string]any{"environment": prod.Code, "package": pkg.Code, "result": "success",
		"executedAt": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)}, 201, &dep)
	if p = move("manager", "deployed", 409); !strings.Contains(p.Detail, "Al checklist de producción le faltan 2 pasos por firmar.") {
		t.Errorf("desplegar sin firmas: %+v", p)
	}
	sign := func(user string, position, wantStatus int) {
		t.Helper()
		var out checklistBody
		a.call(user, "POST", "/api/checklists/"+chk.Code+"/steps/"+string(rune('0'+position))+"/signature", map[string]any{"comment": "Hecho por " + user}, wantStatus, &out)
		if wantStatus == 200 {
			chk = out
		}
	}
	sign("qa", 1, 403) // QA no firma pasos en producción
	sign("ops", 2, 200)
	sign("ops", 2, 409) // una firma no se cambia
	sign("ops", 3, 404)
	if p = move("manager", "deployed", 409); !strings.Contains(p.Detail, "le falta 1 paso por firmar") {
		t.Errorf("desplegar con una firma: %+v", p)
	}
	sign("manager", 1, 200)
	if !chk.Complete || chk.SignedSteps != 2 || chk.Steps[0].Signature.SignedBy.DisplayName != "C. Mora" || chk.Steps[1].Signature.Comment != "Hecho por ops" {
		t.Errorf("CHK firmado: %+v", chk)
	}
	move("manager", "deployed", 200)

	// --- Verificado en la ruta alta: el DIAG revisado y conforme; la prueba de humo es de la ruta ligera. ---
	if got := read().rules("verified"); !slices.Equal(got, []string{"diagnostic_reviewed"}) {
		t.Errorf("qué falta para verificado en la ruta alta: %v", got)
	}
	var diag diagnosticBody
	a.call("qa", "POST", "/api/diagnostics", map[string]any{"deployment": dep.Code, "notes": "Logs limpios"}, 403, nil)
	a.call("dev", "POST", "/api/diagnostics", map[string]any{"deployment": dep.Code, "notes": "Logs limpios"}, 201, &diag)
	if p = move("qa", "verified", 409); !strings.Contains(p.Detail, "Falta que QA revise el diagnóstico posterior al despliegue.") {
		t.Errorf("verificar sin revisar el DIAG: %+v", p)
	}
	review := func(user, code, result, notes string, wantStatus int) {
		t.Helper()
		var out diagnosticBody
		a.call(user, "POST", "/api/diagnostics/"+code+"/review", map[string]any{"result": result, "notes": notes}, wantStatus, &out)
		if wantStatus == 200 {
			diag = out
		}
	}
	review("dev", diag.Code, "conforming", "", 403)
	review("qa", diag.Code, "non_conforming", "", 422) // no conforme dice por qué
	review("qa", diag.Code, "non_conforming", "Errores 500 en el login", 200)
	review("qa", diag.Code, "conforming", "", 409) // una revisión no se cambia: se entrega otro DIAG
	if p = move("qa", "verified", 409); !strings.Contains(p.Detail, "El último diagnóstico no está conforme.") {
		t.Errorf("verificar con el DIAG no conforme: %+v", p)
	}
	a.call("ops", "POST", "/api/diagnostics", map[string]any{"deployment": dep.Code, "notes": "Corregido el pool"}, 201, &diag)
	review("qa", diag.Code, "conforming", "", 200)
	if diag.Review == nil || diag.Review.Result != "conforming" || diag.Review.ReviewedBy.DisplayName != "A. Rojas" {
		t.Errorf("DIAG revisado: %+v", diag)
	}
	move("qa", "verified", 200)

	// Lo firmado y lo revisado no se toca ni con la base: la aplicación no tiene UPDATE ni DELETE.
	for _, table := range []string{"checklist_signature", "diagnostic_review", "smoke_test", "conformance_report", "checklist_template_step"} {
		a.mustFailAsApp("DELETE FROM "+table, "borrar de "+table)
	}

	// --- Informe de conformidad: lo emite QA; inmutable, cada emisión es una versión nueva. ---
	a.call("manager", "POST", "/api/reports", map[string]any{"package": pkg.Code}, 403, nil)
	if a.call("qa", "POST", "/api/reports", map[string]any{"package": "PKG-2026-9999"}, 422, &p); !hasField(p, "package", "invalid") {
		t.Errorf("RPT de un PKG inexistente: %+v", p)
	}
	var v1, v2 reportBody
	a.call("qa", "POST", "/api/reports", map[string]any{"package": pkg.Code}, 201, &v1)
	sum := sha256.Sum256([]byte(v1.Content))
	if v1.Version != 1 || v1.Previous != "" || v1.IssuedBy.DisplayName != "A. Rojas" || v1.Checksum != hex.EncodeToString(sum[:]) {
		t.Errorf("RPT v1: %+v", v1)
	}
	for _, want := range []string{
		"**kaoru** · Trazabilidad a prueba de errores", "# Informe de conformidad " + v1.Code, "| Ruta | Alta |",
		"| Estado del PKG al emitir | Verificado |", build, chk.Code, "| 1 | Respaldar la base | C. Mora |",
		diag.Code, "Conforme, por A. Rojas", "No conforme, por A. Rojas", "Errores 500 en el login",
		"La regresión del PKG no se propuso.",
	} {
		if !strings.Contains(v1.Content, want) {
			t.Errorf("al RPT le falta %q", want)
		}
	}
	a.call("admin", "POST", "/api/reports", map[string]any{"package": pkg.Code}, 201, &v2)
	if v2.Version != 2 || v2.Previous != v1.Code {
		t.Errorf("RPT v2: %+v", v2)
	}
	var again reportBody
	a.call("auditor", "GET", "/api/reports/"+v1.Code, nil, 200, &again)
	if again.SupersededBy != v2.Code || again.Content != v1.Content || again.Checksum != v1.Checksum {
		t.Errorf("la v1 cambió o no sabe que la reemplazaron: %+v", again)
	}
	var reports struct {
		Total int          `json:"total"`
		Items []reportBody `json:"items"`
	}
	a.call("auditor", "GET", "/api/reports?package="+pkg.Code, nil, 200, &reports)
	if reports.Total != 2 || reports.Items[0].Code != v2.Code || reports.Items[0].Content != "" {
		t.Errorf("lista de RPT: %+v", reports)
	}
	a.call("auditor", "GET", "/api/reports/RPT-2026-9999", nil, 404, nil)

	// Exportación: el Markdown tal como se emitió, con su huella; el PDF lleva la huella en el pie.
	rec := a.download("auditor", "/api/reports/"+v1.Code+"/export?format=markdown", 200)
	if rec.Body.String() != v1.Content || !strings.HasPrefix(rec.Header().Get("Content-Disposition"), "attachment") {
		t.Errorf("Markdown exportado: %v", rec.Header())
	}
	rec = a.download("auditor", "/api/reports/"+v1.Code+"/export?format=pdf", 200)
	if body := rec.Body.String(); rec.Header().Get("Content-Type") != "application/pdf" || !strings.HasPrefix(body, "%PDF-1.4") ||
		!strings.Contains(body, v1.Checksum) || !strings.HasSuffix(body, "%%EOF\n") {
		t.Errorf("PDF exportado: %v, %d bytes", rec.Header(), rec.Body.Len())
	}
	a.download("auditor", "/api/reports/"+v1.Code+"/export?format=docx", 422)

	// La búsqueda encuentra CHK, DIAG y RPT.
	for code, kind := range map[string]string{chk.Code: "checklist", diag.Code: "diagnostic", v2.Code: "report"} {
		var hits struct {
			Items []struct {
				Type string `json:"type"`
				Code string `json:"code"`
			} `json:"items"`
		}
		a.call("ops", "GET", "/api/search?q="+code, nil, 200, &hits)
		if len(hits.Items) == 0 || hits.Items[0].Code != code || hits.Items[0].Type != kind {
			t.Errorf("búsqueda de %s: %+v", code, hits.Items)
		}
	}

	// La evidencia de un paso, de un DIAG o de un SMOKE es un DOC enlazado.
	var doc struct {
		Code string `json:"code"`
	}
	a.upload("ops", "/api/documents", map[string]string{"title": "Respaldo", "kind": "evidence", "entityType": "checklist", "entityCode": chk.Code},
		"respaldo.txt", []byte("respaldo ok"), 201, &doc)
	a.call("qa", "POST", "/api/documents/"+doc.Code+"/links", map[string]any{"entityType": "diagnostic", "entityCode": diag.Code}, 200, nil)
	a.call("qa", "POST", "/api/documents/"+doc.Code+"/links", map[string]any{"entityType": "checklist", "entityCode": "CHK-2026-9999"}, 422, nil)

	// Auditoría: cada firma y cada emisión quedan con su autor.
	for code, want := range map[string]string{chk.Code: "C. Mora", v2.Code: "M. León"} {
		var audit auditPage
		a.call("auditor", "GET", "/api/audit?entityCode="+code, nil, 200, &audit)
		if !slices.ContainsFunc(audit.Items, func(e struct {
			Seq        int64  `json:"seq"`
			Action     string `json:"action"`
			Field      string `json:"field"`
			OldValue   string `json:"oldValue"`
			NewValue   string `json:"newValue"`
			Reason     string `json:"reason"`
			EntityCode string `json:"entityCode"`
			Actor      struct {
				DisplayName string `json:"displayName"`
			} `json:"actor"`
		}) bool {
			return e.Actor.DisplayName == want
		}) {
			t.Errorf("auditoría de %s: %+v", code, audit.Items)
		}
	}
}

// El calendario muestra las ventanas de los CHG vigentes y avisa de lo atrasado: un hito pendiente vencido
// o una ventana que terminó sin que el CHG esté liberado.
func TestCalendarWindowsMilestonesAndAlerts(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	at := func(d time.Duration) string { return time.Now().Add(d).UTC().Format(time.RFC3339) }

	var late, open changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": a.uniqueName("Ventana vencida"), "system": "SIMF",
		"windowStart": at(-72 * time.Hour), "windowEnd": at(-25 * time.Hour)}, 201, &late)
	a.call("manager", "POST", "/api/changes", map[string]any{"title": a.uniqueName("Ventana próxima"), "system": "SYPAGO",
		"windowStart": at(24 * time.Hour), "windowEnd": at(48 * time.Hour)}, 201, &open)

	type milestoneBody struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		DueDate string `json:"dueDate"`
		Owner   *struct {
			DisplayName string `json:"displayName"`
		} `json:"owner"`
		Completed bool `json:"completed"`
		DaysLate  int  `json:"daysLate"`
	}
	a.call("qa", "POST", "/api/milestones", map[string]any{"title": "Congelamiento", "dueDate": "2026-11-30"}, 403, nil)
	if a.call("manager", "POST", "/api/milestones", map[string]any{"title": "Congelamiento", "dueDate": "30/11"}, 422, &p); !hasField(p, "dueDate", "invalid") {
		t.Errorf("fecha mal escrita: %+v", p)
	}
	var overdue, upcoming milestoneBody
	title := a.uniqueName("Congelamiento")
	a.call("manager", "POST", "/api/milestones", map[string]any{"title": title, "owner": "sub-ops",
		"dueDate": time.Now().UTC().AddDate(0, 0, -3).Format("2006-01-02")}, 201, &overdue)
	a.call("manager", "POST", "/api/milestones", map[string]any{"title": a.uniqueName("Salida"),
		"dueDate": time.Now().UTC().AddDate(0, 0, 10).Format("2006-01-02")}, 201, &upcoming)
	if overdue.DaysLate != 3 || overdue.Owner == nil || overdue.Owner.DisplayName != "L. Paz" || upcoming.DaysLate != 0 {
		t.Errorf("hitos: %+v %+v", overdue, upcoming)
	}

	type calendarBody struct {
		Windows []struct {
			Change struct {
				Code string `json:"code"`
			} `json:"change"`
			DaysLate int `json:"daysLate"`
		} `json:"windows"`
		Alerts []struct {
			Kind      string `json:"kind"`
			Change    string `json:"change"`
			Milestone string `json:"milestone"`
			DaysLate  int    `json:"daysLate"`
		} `json:"alerts"`
	}
	calendar := func() (windows map[string]int, alerts map[string]int) {
		t.Helper()
		var c calendarBody
		a.call("dev", "GET", "/api/calendar", nil, 200, &c)
		windows, alerts = map[string]int{}, map[string]int{}
		for _, w := range c.Windows {
			windows[w.Change.Code] = w.DaysLate
		}
		for _, al := range c.Alerts {
			alerts[al.Kind+":"+al.Change+al.Milestone] = al.DaysLate
		}
		return windows, alerts
	}
	windows, alerts := calendar()
	if windows[late.Code] != 2 || windows[open.Code] != 0 {
		t.Errorf("ventanas: %v", windows)
	}
	if alerts["change_window:"+late.Code] != 2 || alerts["milestone:"+overdue.ID] != 3 {
		t.Errorf("alertas: %v", alerts)
	}
	if _, ok := alerts["milestone:"+upcoming.ID]; ok {
		t.Error("un hito que no venció no es una alerta")
	}

	// Cumplido, el hito deja de ser una alerta; rechazado, el CHG deja el calendario.
	var done milestoneBody
	a.call("ops", "PATCH", "/api/milestones/"+overdue.ID, map[string]any{"completed": true}, 403, nil)
	a.call("manager", "PATCH", "/api/milestones/"+overdue.ID, map[string]any{"completed": true, "owner": ""}, 200, &done)
	if !done.Completed || done.DaysLate != 0 || done.Owner != nil {
		t.Errorf("hito cumplido: %+v", done)
	}
	a.call("manager", "PATCH", "/api/milestones/00000000-0000-0000-0000-000000000000", map[string]any{"completed": true}, 404, nil)
	a.call("manager", "PATCH", "/api/milestones/nada", map[string]any{"completed": true}, 404, nil)
	a.call("manager", "POST", "/api/changes/"+late.Code+"/transitions", map[string]any{"to": "rejected", "reason": "Se replanifica"}, 200, nil)
	windows, alerts = calendar()
	if _, ok := windows[late.Code]; ok {
		t.Error("un CHG rechazado sigue en el calendario")
	}
	if _, ok := alerts["milestone:"+overdue.ID]; ok {
		t.Error("un hito cumplido sigue como alerta")
	}

	var page struct {
		Total int             `json:"total"`
		Items []milestoneBody `json:"items"`
	}
	a.call("auditor", "GET", "/api/milestones?status=completed&size=100", nil, 200, &page)
	if !slices.ContainsFunc(page.Items, func(m milestoneBody) bool { return m.ID == overdue.ID }) {
		t.Errorf("el hito cumplido no está entre los cumplidos: %+v", page)
	}
	a.call("auditor", "GET", "/api/milestones?status=otro", nil, 422, nil)

}
