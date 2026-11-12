package converter_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/document/adapter/converter"
)

func TestToPDFSendsTheFileAndStreamsThePDF(t *testing.T) {
	var gotName, gotBody, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		file, header, err := r.FormFile("files")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		raw, _ := io.ReadAll(file)
		gotName, gotBody = header.Filename, string(raw)
		_, _ = w.Write([]byte("%PDF-1.7 convertido"))
	}))
	defer server.Close()

	pdf, err := converter.New(server.URL+"/").ToPDF(context.Background(), "../Informe FINAL.DOCX", strings.NewReader("contenido docx"))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(pdf)
	if err := pdf.Close(); err != nil {
		t.Fatal(err)
	}
	if string(raw) != "%PDF-1.7 convertido" || gotBody != "contenido docx" || gotPath != "/forms/libreoffice/convert" {
		t.Errorf("pdf %q, cuerpo %q, ruta %q", raw, gotBody, gotPath)
	}
	// El nombre del usuario no viaja: solo la extensión.
	if gotName != "documento.docx" {
		t.Errorf("nombre enviado = %q", gotName)
	}
}

func TestToPDFFailsAsUnavailable(t *testing.T) {
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no pude", http.StatusInternalServerError)
	}))
	defer broken.Close()
	if _, err := converter.New(broken.URL).ToPDF(context.Background(), "a.xlsx", strings.NewReader("x")); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("conversor con error: quiero SERVICE_UNAVAILABLE, obtuve %v", err)
	}
	down := httptest.NewServer(http.NotFoundHandler())
	down.Close()
	if _, err := converter.New(down.URL).ToPDF(context.Background(), "a.xlsx", strings.NewReader("x")); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("conversor apagado: quiero SERVICE_UNAVAILABLE, obtuve %v", err)
	}
}
