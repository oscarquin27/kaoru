// Package port declara lo que necesitan los casos de uso de document.
package port

import (
	"context"
	"io"
	"time"

	"kaoru/internal/feature/document/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los metadatos de los documentos. El contenido vive en el FileStore.
// No tiene cómo editar ni borrar una versión.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año.
	NextCode(ctx context.Context, year int) (shared.Code, error)
	// Insert guarda un documento nuevo con su primera versión y su primer enlace.
	Insert(ctx context.Context, document domain.Document, linkedBy shared.UserId) error
	// AddVersion agrega una versión con el número siguiente y lo devuelve.
	// Devuelve NOT_FOUND_RESOURCE si el documento no existe.
	AddVersion(ctx context.Context, code shared.Code, version domain.Version) (number int, err error)
	// AddLink enlaza el documento a una entidad; enlazar dos veces no cambia nada.
	AddLink(ctx context.Context, documentId string, entity domain.EntityRef, by shared.UserId, at time.Time) error
	FindByCode(ctx context.Context, code shared.Code) (domain.Document, error)
	ListByEntity(ctx context.Context, entity domain.EntityRef) ([]domain.Document, error)
}

// FileStore guarda el contenido de los archivos, nombrado por su SHA-256.
type FileStore interface {
	Put(ctx context.Context, r io.Reader) (Stored, error)
	// Open devuelve NOT_FOUND_RESOURCE si el archivo no está.
	Open(ctx context.Context, checksum string) (io.ReadCloser, error)
	// Detect dice el tipo real de un contenido a partir de sus primeros bytes.
	Detect(head []byte) string
}

// Stored es el resultado de guardar un archivo.
type Stored struct {
	Checksum string
	Size     int64
}

// Entities resuelve, en los demás features, la entidad a la que se enlaza un documento.
type Entities interface {
	// Resolve devuelve NOT_FOUND_RESOURCE si la entidad no existe o el tipo no admite documentos.
	Resolve(ctx context.Context, entityType, code string) (domain.EntityRef, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
)

// AuditRecord es un hecho sobre un documento.
type AuditRecord struct {
	Actor      shared.UserId
	DocumentId string
	Code       shared.Code
	Action     AuditAction
	Field      string
	NewValue   string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
