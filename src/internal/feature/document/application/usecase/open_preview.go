package usecase

import (
	"context"
	"io"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/document/domain"
)

// ErrNoPreview es el error de pedir la vista previa de algo que no la tiene.
var ErrNoPreview = axis.New(axis.NotSupported, "Este archivo no tiene vista previa. Descárgalo para abrirlo.")

// OpenPreview convierte una versión de Office a PDF para verla dentro del sistema.
// Solo existe si el servidor tiene conversor; el original siempre se puede descargar.
type OpenPreview struct {
	d Deps
}

// CanPreview dice si una versión de ese tipo tiene vista previa en este servidor.
func (uc *OpenPreview) CanPreview(mediaType string) bool {
	return uc.d.Converter != nil && domain.IsOffice(mediaType)
}

// OpenPreviewOutput es el PDF de la vista previa. Quien lo recibe cierra Content.
type OpenPreviewOutput struct {
	Version domain.Version
	Content io.ReadCloser
}

// Execute devuelve NOT_FOUND_RESOURCE si la versión no existe y NOT_SUPPORTED si no tiene vista previa.
// Leer no exige permiso.
func (uc *OpenPreview) Execute(ctx context.Context, in OpenContentInput) (OpenPreviewOutput, error) {
	document, err := findDocument(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return OpenPreviewOutput{}, err
	}
	version, ok := document.Version(in.Version)
	if !ok {
		return OpenPreviewOutput{}, ErrDocumentNotFound
	}
	if !uc.CanPreview(version.MediaType) {
		return OpenPreviewOutput{}, ErrNoPreview
	}
	original, err := uc.d.FileStore.Open(ctx, version.Checksum)
	if err != nil {
		return OpenPreviewOutput{}, err
	}
	defer func() { _ = original.Close() }()
	pdf, err := uc.d.Converter.ToPDF(ctx, version.FileName, original)
	if err != nil {
		return OpenPreviewOutput{}, err
	}
	return OpenPreviewOutput{Version: version, Content: pdf}, nil
}
