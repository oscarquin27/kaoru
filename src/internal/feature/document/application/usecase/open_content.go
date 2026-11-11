package usecase

import (
	"context"
	"io"

	"kaoru/internal/feature/document/domain"
)

// OpenContent abre el contenido de una versión, para el visor o para descargar.
type OpenContent struct {
	d Deps
}

// OpenContentInput es la entrada de OpenContent.
type OpenContentInput struct {
	Code    string
	Version int
}

// OpenContentOutput es el contenido con sus metadatos. Quien lo recibe cierra Content.
type OpenContentOutput struct {
	Version domain.Version
	Content io.ReadCloser
}

// Execute devuelve NOT_FOUND_RESOURCE si el documento o la versión no existen. Leer no exige permiso.
func (uc *OpenContent) Execute(ctx context.Context, in OpenContentInput) (OpenContentOutput, error) {
	document, err := findDocument(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return OpenContentOutput{}, err
	}
	version, ok := document.Version(in.Version)
	if !ok {
		return OpenContentOutput{}, ErrDocumentNotFound
	}
	content, err := uc.d.FileStore.Open(ctx, version.Checksum)
	if err != nil {
		return OpenContentOutput{}, err
	}
	return OpenContentOutput{Version: version, Content: content}, nil
}
