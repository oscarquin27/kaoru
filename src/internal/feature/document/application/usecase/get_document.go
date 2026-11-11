package usecase

import (
	"context"

	"kaoru/internal/feature/document/domain"
)

// GetDocument devuelve un documento con sus versiones y sus enlaces. Leer no exige permiso.
type GetDocument struct {
	d Deps
}

// GetDocumentInput es la entrada de GetDocument.
type GetDocumentInput struct {
	Code string
}

// Execute busca el documento.
func (uc *GetDocument) Execute(ctx context.Context, in GetDocumentInput) (domain.Snapshot, error) {
	document, err := findDocument(ctx, uc.d.Repository, in.Code)
	return document.Snapshot(), err
}
