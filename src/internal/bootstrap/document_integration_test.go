//go:build integration

package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

type documentBody struct {
	Code     string `json:"code"`
	Kind     string `json:"kind"`
	Versions []struct {
		Version    int    `json:"version"`
		FileName   string `json:"fileName"`
		MediaType  string `json:"mediaType"`
		SizeBytes  int64  `json:"sizeBytes"`
		Checksum   string `json:"checksum"`
		UploadedBy struct {
			DisplayName string `json:"displayName"`
		} `json:"uploadedBy"`
	} `json:"versions"`
	LinkedTo []struct {
		Type string `json:"type"`
		Code string `json:"code"`
	} `json:"linkedTo"`
}

// upload hace una subida multipart como el usuario dado.
func (a *api) upload(user, path string, fields map[string]string, fileName string, content []byte, wantStatus int, out any) {
	a.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = form.WriteField(k, v)
	}
	if fileName != "" {
		part, err := form.CreateFormFile("file", fileName)
		if err != nil {
			a.t.Fatal(err)
		}
		_, _ = part.Write(content)
	}
	_ = form.Close()
	req := httptest.NewRequest("POST", path, &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+user)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		a.t.Fatalf("subida a %s como %q (%s): status %d, quiero %d: %s", path, user, fileName, rec.Code, wantStatus, rec.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			a.t.Fatalf("respuesta ilegible: %v: %s", err, rec.Body.String())
		}
	}
}

func (a *api) download(user, path string, wantStatus int) *httptest.ResponseRecorder {
	a.t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Authorization", "Bearer "+user)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		a.t.Fatalf("GET %s: status %d, quiero %d: %s", path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}

func TestDocumentsAreVersionedAndTypeChecked(t *testing.T) {
	a := newAPI(t)
	var p problemBody

	var chg changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": "Pagos", "system": "SYPAGO"}, 201, &chg)
	var req requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Límite diario", "system": "SYPAGO"}, 201, &req)

	pdf := []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n")
	sum := sha256.Sum256(pdf)
	fields := map[string]string{"title": "Especificación del límite", "kind": "specification", "entityType": "requirement", "entityCode": req.Code}

	// Quién puede y qué se rechaza: el tipo se valida por el contenido, no por el nombre.
	a.upload("auditor", "/api/documents", fields, "spec.pdf", pdf, 403, nil)
	a.upload("qa", "/api/documents", fields, "spec.pdf", []byte("MZ\x90\x00 esto es un ejecutable"), 415, &p)
	if p.Code != "NOT_SUPPORTED" {
		t.Errorf("ejecutable con nombre de PDF: %+v", p)
	}
	a.upload("qa", "/api/documents", fields, "notas.txt", []byte("<html><script>alert(1)</script></html>"), 415, nil)
	a.upload("qa", "/api/documents", fields, "herramienta.exe", pdf, 415, nil)
	a.upload("qa", "/api/documents", fields, "vacio.pdf", nil, 422, nil)
	a.upload("qa", "/api/documents", fields, "", nil, 422, nil)
	a.upload("qa", "/api/documents", map[string]string{"title": "x", "kind": "specification", "entityType": "requirement", "entityCode": "REQ-2026-9999"}, "spec.pdf", pdf, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "entityCode" {
		t.Errorf("entidad inexistente: %+v", p)
	}
	a.upload("qa", "/api/documents", map[string]string{"entityType": "requirement", "entityCode": req.Code}, "spec.pdf", pdf, 422, &p)
	if len(p.Errors) != 2 {
		t.Errorf("sin título ni clase: %+v", p.Errors)
	}
	big := append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("x"), 3<<20)...)
	a.upload("qa", "/api/documents", fields, "enorme.pdf", big, 413, &p)
	if p.Code != "RESOURCE_EXHAUSTED" {
		t.Errorf("archivo que pasa el límite: %+v", p)
	}

	// Primera versión: checksum calculado por el servidor.
	var doc documentBody
	a.upload("qa", "/api/documents", fields, `C:\docs\spec.pdf`, pdf, 201, &doc)
	if !strings.HasPrefix(doc.Code, "DOC-") || len(doc.Versions) != 1 || doc.Versions[0].Version != 1 ||
		doc.Versions[0].Checksum != hex.EncodeToString(sum[:]) || doc.Versions[0].SizeBytes != int64(len(pdf)) ||
		doc.Versions[0].FileName != "spec.pdf" || doc.Versions[0].MediaType != "application/pdf" ||
		doc.Versions[0].UploadedBy.DisplayName != "A. Rojas" || len(doc.LinkedTo) != 1 || doc.LinkedTo[0].Code != req.Code {
		t.Fatalf("documento inesperado: %+v", doc)
	}
	path := "/api/documents/" + doc.Code

	// Segunda versión: la anterior se conserva.
	markdown := []byte("# Límite diario\n\nRegla nueva.\n")
	a.upload("ops", path+"/versions", nil, "spec.md", markdown, 201, &doc)
	if len(doc.Versions) != 2 || doc.Versions[0].Version != 2 || doc.Versions[0].MediaType != "text/markdown" || doc.Versions[1].Version != 1 {
		t.Fatalf("las versiones van de la más reciente a la más antigua: %+v", doc.Versions)
	}
	a.upload("qa", "/api/documents/DOC-2026-9999/versions", nil, "spec.pdf", pdf, 404, nil)

	// Contenido: el de cada versión, con cabeceras que impiden que el navegador lo ejecute.
	rec := a.download("auditor", path+"/versions/1/content", 200)
	if !bytes.Equal(rec.Body.Bytes(), pdf) || rec.Header().Get("Content-Type") != "application/pdf" ||
		!strings.HasPrefix(rec.Header().Get("Content-Disposition"), "inline") ||
		rec.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Errorf("contenido de la v1: %v", rec.Header())
	}
	rec = a.download("auditor", path+"/versions/2/content", 200)
	if !bytes.Equal(rec.Body.Bytes(), markdown) || rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("el Markdown se sirve como texto plano: %v", rec.Header())
	}
	rec = a.download("auditor", path+"/versions/1/content?download=true", 200)
	if !strings.HasPrefix(rec.Header().Get("Content-Disposition"), "attachment") {
		t.Errorf("download=true debe descargar: %v", rec.Header())
	}
	a.download("auditor", path+"/versions/9/content", 404)
	a.download("auditor", path+"/versions/x/content", 404)

	// Enlazar a otra entidad y listar por entidad.
	a.call("qa", "POST", path+"/links", map[string]any{"entityType": "change", "entityCode": chg.Code}, 200, &doc)
	a.call("qa", "POST", path+"/links", map[string]any{"entityType": "change", "entityCode": chg.Code}, 200, &doc)
	if len(doc.LinkedTo) != 2 {
		t.Errorf("enlazar dos veces a lo mismo no duplica: %+v", doc.LinkedTo)
	}
	a.call("qa", "POST", path+"/links", map[string]any{"entityType": "paquete", "entityCode": chg.Code}, 422, nil)
	var list struct {
		Items []documentBody `json:"items"`
	}
	a.call("ops", "GET", "/api/documents?entityType=change&entityCode="+chg.Code, nil, 200, &list)
	if len(list.Items) != 1 || list.Items[0].Code != doc.Code {
		t.Errorf("documentos del CHG: %+v", list.Items)
	}
	a.call("ops", "GET", "/api/documents?entityType=requirement&entityCode=REQ-2026-9999", nil, 200, &list)
	if len(list.Items) != 0 {
		t.Errorf("una entidad que no existe no tiene documentos: %+v", list.Items)
	}

	// La instantánea de un enlace de Google Sheets es un DOC: con ella deja de avisar.
	var link linkBody
	a.call("qa", "POST", "/api/requirements/"+req.Code+"/links", map[string]any{
		"url": "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789/edit#gid=0",
	}, 201, &link)
	var patched struct {
		SnapshotDocument string `json:"snapshotDocument"`
	}
	a.call("qa", "PATCH", "/api/requirements/"+req.Code+"/links/"+link.ID, map[string]any{"snapshotDocument": doc.Code}, 200, &patched)
	if patched.SnapshotDocument != doc.Code {
		t.Errorf("instantánea del enlace: %+v", patched)
	}

	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+doc.Code, nil, 200, &audit)
	if audit.Total != 3 || audit.Items[2].Action != "create" || audit.Items[2].NewValue != hex.EncodeToString(sum[:]) ||
		audit.Items[1].Field != "version.2" || audit.Items[0].Field != "link" || audit.Items[0].NewValue != chg.Code {
		t.Errorf("auditoría del documento: %+v", audit.Items)
	}
}
