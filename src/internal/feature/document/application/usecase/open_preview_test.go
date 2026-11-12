package usecase_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/document/application/usecase"
	"kaoru/internal/feature/document/domain"
)

const docxType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

// fakeConverter «convierte» anteponiendo una cabecera de PDF a lo que recibe.
type fakeConverter struct {
	err      error
	fileName string
}

func (c *fakeConverter) ToPDF(_ context.Context, fileName string, content io.Reader) (io.ReadCloser, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.fileName = fileName
	raw, _ := io.ReadAll(content)
	return io.NopCloser(strings.NewReader("%PDF-1.7 de: " + string(raw))), nil
}

// withOffice arma los casos de uso con un documento cuya versión 1 es un PDF y la 2 un .docx.
func withOffice(t *testing.T, conv *fakeConverter) (*usecase.UseCases, *world, string) {
	t.Helper()
	w := &world{documents: map[string]domain.Document{}, files: map[string][]byte{}}
	d := deps(w)
	if conv != nil {
		d.Converter = conv
	}
	uc, err := usecase.New(d)
	if err != nil {
		t.Fatal(err)
	}
	code := attach(t, uc)
	// La segunda versión se registra como .docx directamente: el doble de Detect no conoce zip.
	snapshot := w.documents[code].Snapshot()
	stored, _ := w.Put(ctx, strings.NewReader("contenido docx"))
	version := snapshot.Versions[0]
	version.Number, version.FileName, version.MediaType, version.Checksum = 2, "informe.docx", docxType, stored.Checksum
	snapshot.Versions = append(snapshot.Versions, version)
	w.documents[code] = domain.Rehydrate(snapshot)
	return uc, w, code
}

func TestOpenPreviewConvertsOfficeFiles(t *testing.T) {
	conv := &fakeConverter{}
	uc, _, code := withOffice(t, conv)

	if !uc.OpenPreview.CanPreview(docxType) || uc.OpenPreview.CanPreview("application/pdf") {
		t.Error("solo los archivos de Office tienen vista previa convertida")
	}
	out, err := uc.OpenPreview.Execute(ctx, usecase.OpenContentInput{Code: code, Version: 2})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(out.Content)
	_ = out.Content.Close()
	if string(raw) != "%PDF-1.7 de: contenido docx" || conv.fileName != "informe.docx" || out.Version.Number != 2 {
		t.Errorf("vista previa inesperada: %q de %q", raw, conv.fileName)
	}
}

func TestOpenPreviewErrors(t *testing.T) {
	uc, w, code := withOffice(t, &fakeConverter{})
	if _, err := uc.OpenPreview.Execute(ctx, usecase.OpenContentInput{Code: code, Version: 1}); !errors.Is(err, usecase.ErrNoPreview) {
		t.Errorf("un PDF no se convierte: %v", err)
	}
	if _, err := uc.OpenPreview.Execute(ctx, usecase.OpenContentInput{Code: code, Version: 9}); !errors.Is(err, usecase.ErrDocumentNotFound) {
		t.Errorf("versión inexistente: %v", err)
	}
	if _, err := uc.OpenPreview.Execute(ctx, usecase.OpenContentInput{Code: "DOC-2026-0099", Version: 1}); !errors.Is(err, usecase.ErrDocumentNotFound) {
		t.Errorf("documento inexistente: %v", err)
	}
	for checksum := range w.files {
		delete(w.files, checksum)
	}
	if _, err := uc.OpenPreview.Execute(ctx, usecase.OpenContentInput{Code: code, Version: 2}); !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("archivo perdido: %v", err)
	}

	// Sin conversor en el servidor no hay vista previa de nada.
	uc, _, code = withOffice(t, nil)
	if uc.OpenPreview.CanPreview(docxType) {
		t.Error("sin conversor nada es previsualizable")
	}
	if _, err := uc.OpenPreview.Execute(ctx, usecase.OpenContentInput{Code: code, Version: 2}); !errors.Is(err, usecase.ErrNoPreview) {
		t.Errorf("sin conversor: %v", err)
	}

	boom := axis.New(axis.ServiceUnavailable, "conversor caído")
	uc, _, code = withOffice(t, &fakeConverter{err: boom})
	if _, err := uc.OpenPreview.Execute(ctx, usecase.OpenContentInput{Code: code, Version: 2}); !errors.Is(err, boom) {
		t.Errorf("quiero el error del conversor: %v", err)
	}
}
