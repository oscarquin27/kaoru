package domain_test

import (
	"errors"
	"strings"
	"testing"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/document/domain"
)

var allowed = []string{".pdf", ".png", ".md", ".txt", ".log", ".xlsx"}

func TestResolveFile(t *testing.T) {
	cases := []struct {
		name      string
		fileName  string
		detected  string
		wantName  string
		wantType  string
		wantError error
		wantCode  axis.Code
	}{
		{"PDF", "Especificación.pdf", "application/pdf", "Especificación.pdf", "application/pdf", nil, ""},
		{"extensión en mayúsculas", "CAPTURA.PNG", "image/png", "CAPTURA.PNG", "image/png", nil, ""},
		{"Markdown se guarda como tal", "notas.md", "text/plain", "notas.md", "text/markdown", nil, ""},
		{"log", "salida.log", "text/plain", "salida.log", "text/plain", nil, ""},
		{"hoja de cálculo", "evidencia.xlsx", "application/zip", "evidencia.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil, ""},
		{"quita la ruta de Windows", `C:\Users\ana\spec.pdf`, "application/pdf", "spec.pdf", "application/pdf", nil, ""},
		{"quita la ruta que intenta salir", "../../etc/spec.pdf", "application/pdf", "spec.pdf", "application/pdf", nil, ""},
		{"ejecutable disfrazado de PDF", "factura.pdf", "application/octet-stream", "", "", domain.ErrTypeMismatch, ""},
		{"HTML disfrazado de texto", "notas.txt", "text/html", "", "", domain.ErrTypeMismatch, ""},
		{"zip cualquiera con nombre de PDF", "datos.pdf", "application/zip", "", "", domain.ErrTypeMismatch, ""},
		{"tipo que el sistema no conoce", "virus.exe", "application/octet-stream", "", "", domain.ErrTypeNotSupported, ""},
		{"tipo conocido pero no permitido aquí", "foto.jpg", "image/jpeg", "", "", domain.ErrTypeNotSupported, ""},
		{"sin extensión", "LEEME", "text/plain", "", "", domain.ErrTypeNotSupported, ""},
		{"doble extensión: manda la última", "informe.pdf.exe", "application/octet-stream", "", "", domain.ErrTypeNotSupported, ""},
		{"sin nombre", "  ", "application/pdf", "", "", nil, axis.InvalidRequest},
		{"nombre larguísimo", strings.Repeat("a", 201) + ".pdf", "application/pdf", "", "", nil, axis.InvalidRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, mediaType, err := domain.ResolveFile(tc.fileName, tc.detected, allowed)
			switch {
			case tc.wantError != nil:
				if !errors.Is(err, tc.wantError) || !axis.IsCode(err, axis.NotSupported) {
					t.Fatalf("esperaba %v (415), obtuve %v", tc.wantError, err)
				}
			case tc.wantCode != "":
				if !axis.IsCode(err, tc.wantCode) {
					t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
				}
			default:
				if err != nil || name != tc.wantName || mediaType != tc.wantType {
					t.Fatalf("ResolveFile = %q, %q, %v", name, mediaType, err)
				}
			}
		})
	}
}
