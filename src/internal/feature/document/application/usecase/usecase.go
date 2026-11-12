// Package usecase contiene los casos de uso de document.
package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/document/application/port"
	"kaoru/internal/feature/document/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionUpload es el permiso para adjuntar documentos, subir versiones y enlazarlos.
const PermissionUpload = "doc.upload"

// sniffLength son los bytes que bastan para detectar el tipo real de un contenido.
const sniffLength = 512

// ErrDocumentNotFound es el error de un documento o una versión que no existe.
var ErrDocumentNotFound = axis.New(axis.NotFoundResource, "El documento no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	FileStore   port.FileStore
	Entities    port.Entities
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
	// Converter es opcional: nil cuando el servidor no tiene conversor de Office.
	Converter port.Converter
	// AllowedExtensions es la lista configurada de tipos permitidos, como extensiones con punto.
	AllowedExtensions []string
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateDocument *CreateDocument
	AddVersion     *AddVersion
	LinkDocument   *LinkDocument
	GetDocument    *GetDocument
	ListDocuments  *ListDocuments
	OpenContent    *OpenContent
	OpenPreview    *OpenPreview
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.FileStore == nil || d.Entities == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil || len(d.AllowedExtensions) == 0 {
		return nil, errors.New("document: faltan dependencias")
	}
	return &UseCases{
		CreateDocument: &CreateDocument{d: d},
		AddVersion:     &AddVersion{d: d},
		LinkDocument:   &LinkDocument{d: d},
		GetDocument:    &GetDocument{d: d},
		ListDocuments:  &ListDocuments{d: d},
		OpenContent:    &OpenContent{d: d},
		OpenPreview:    &OpenPreview{d: d},
	}, nil
}

// FileInput es un archivo que llega en una subida.
type FileInput struct {
	Name    string
	Content io.Reader
}

// storeFile valida el tipo real del archivo y lo guarda. Va fuera de toda transacción:
// primero el archivo, después su registro. Un archivo sin registro se puede limpiar;
// un registro sin archivo no puede existir.
func storeFile(ctx context.Context, d Deps, file FileInput, by shared.UserId) (domain.Version, error) {
	if file.Content == nil {
		return domain.Version{}, axis.Invalid(axis.Required("file", "Adjunta un archivo."))
	}
	head := make([]byte, sniffLength)
	n, err := io.ReadFull(file.Content, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return domain.Version{}, err
	}
	if n == 0 {
		return domain.Version{}, domain.ErrEmptyFile
	}
	head = head[:n]
	// El tipo se valida antes de escribir nada en disco.
	fileName, mediaType, err := domain.ResolveFile(file.Name, d.FileStore.Detect(head), d.AllowedExtensions)
	if err != nil {
		return domain.Version{}, err
	}
	stored, err := d.FileStore.Put(ctx, io.MultiReader(bytes.NewReader(head), file.Content))
	if err != nil {
		return domain.Version{}, err
	}
	id, err := d.IdGenerator.NewId()
	if err != nil {
		return domain.Version{}, err
	}
	return domain.Version{
		Id: id, FileName: fileName, MediaType: mediaType, SizeBytes: stored.Size, Checksum: stored.Checksum,
		UploadedBy: domain.UserRef{Id: by}, CreatedAt: d.Clock.Now(),
	}, nil
}

// resolveEntity busca la entidad a enlazar; si no existe, lo dice en el campo.
func resolveEntity(ctx context.Context, d Deps, entityType, code string) (domain.EntityRef, error) {
	if entityType == "" || code == "" {
		return domain.EntityRef{}, axis.Invalid(axis.Required("entityCode", "Indica a qué se adjunta el documento."))
	}
	entity, err := d.Entities.Resolve(ctx, entityType, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.EntityRef{}, axis.Invalid(axis.InvalidField("entityCode", "No existe "+code+" o no admite documentos."))
	}
	return entity, err
}

func findDocument(ctx context.Context, repo port.Repository, raw string) (domain.Document, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Document{}, ErrDocumentNotFound
	}
	document, err := repo.FindByCode(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Document{}, ErrDocumentNotFound
	}
	return document, err
}
