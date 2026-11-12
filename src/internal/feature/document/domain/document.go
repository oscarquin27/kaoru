// Package domain modela el DOC: un documento adjunto versionado, con checksum, enlazado a cualquier entidad.
package domain

import (
	"path"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Kind es la clase de documento.
type Kind string

var kinds = []Kind{"specification", "manual", "evidence", "minutes", "snapshot", "other"}

const (
	maxTitleLength    = 200
	maxFileNameLength = 200
)

// Errores de dominio.
var (
	ErrTypeNotSupported = axis.New(axis.NotSupported, "Ese tipo de archivo no está permitido.")
	ErrTypeMismatch     = axis.New(axis.NotSupported, "El contenido del archivo no corresponde a su extensión.")
	ErrEmptyFile        = axis.Invalid(axis.Required("file", "El archivo está vacío."))
)

// fileType dice qué contenido real se espera para una extensión y con qué tipo se sirve.
type fileType struct {
	detected  string // lo que debe decir la detección por contenido
	mediaType string // con qué Content-Type se guarda y se sirve
}

// Tipos que el sistema sabe mostrar o descargar. La lista de extensiones permitidas es configuración.
var fileTypes = map[string]fileType{
	".pdf":  {"application/pdf", "application/pdf"},
	".png":  {"image/png", "image/png"},
	".jpg":  {"image/jpeg", "image/jpeg"},
	".jpeg": {"image/jpeg", "image/jpeg"},
	".md":   {"text/plain", "text/markdown"},
	".txt":  {"text/plain", "text/plain"},
	".log":  {"text/plain", "text/plain"},
	".csv":  {"text/plain", "text/csv"},
	".docx": {"application/zip", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	".xlsx": {"application/zip", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
}

// ResolveFile valida un archivo por su nombre y por el tipo real de su contenido (no solo la
// extensión). Devuelve el nombre saneado y el tipo con el que se guarda.
func ResolveFile(rawName, detected string, allowedExtensions []string) (fileName, mediaType string, err error) {
	// Solo el nombre: sin directorios, venga con / o con \.
	fileName = path.Base(strings.ReplaceAll(strings.TrimSpace(rawName), `\`, "/"))
	if fileName == "" || fileName == "." || fileName == "/" {
		return "", "", axis.Invalid(axis.Required("file", "Adjunta un archivo."))
	}
	if utf8.RuneCountInString(fileName) > maxFileNameLength {
		return "", "", axis.Invalid(axis.InvalidField("file", "El nombre del archivo no puede pasar de 200 caracteres."))
	}
	extension := strings.ToLower(path.Ext(fileName))
	expected, known := fileTypes[extension]
	if !known || !slices.Contains(allowedExtensions, extension) {
		return "", "", ErrTypeNotSupported
	}
	if detected != expected.detected {
		return "", "", ErrTypeMismatch
	}
	return fileName, expected.mediaType, nil
}

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// Version es una versión de un documento. No cambia después de subida.
type Version struct {
	Id         string
	Number     int
	FileName   string
	MediaType  string
	SizeBytes  int64
	Checksum   string
	UploadedBy UserRef
	CreatedAt  time.Time
}

// EntityRef es una entidad a la que se enlaza un documento.
type EntityRef struct {
	Type string
	Id   string
	Code string
}

// Snapshot es el contenido de un DOC.
type Snapshot struct {
	Id    string
	Code  shared.Code
	Title string
	Kind  Kind
	// Versions va de la más reciente a la más antigua.
	Versions  []Version
	LinkedTo  []EntityRef
	CreatedBy shared.UserId
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Document es un DOC.
type Document struct {
	s Snapshot
}

// New valida y crea un documento con su primera versión, enlazado a una entidad.
func New(id string, code shared.Code, title, kind string, first Version, entity EntityRef, now time.Time) (Document, error) {
	var errs axis.FieldErrors
	title = strings.TrimSpace(title)
	switch {
	case title == "":
		errs.Add(axis.Required("title", "Escribe un título."))
	case utf8.RuneCountInString(title) > maxTitleLength:
		errs.Add(axis.InvalidField("title", "El título no puede pasar de 200 caracteres."))
	}
	switch {
	case kind == "":
		errs.Add(axis.Required("kind", "Elige la clase de documento."))
	case !slices.Contains(kinds, Kind(kind)):
		errs.Add(axis.InvalidField("kind", "La clase de documento no existe."))
	}
	if err := errs.Err(); err != nil {
		return Document{}, err
	}
	first.Number = 1
	return Document{s: Snapshot{
		Id: id, Code: code, Title: title, Kind: Kind(kind), Versions: []Version{first},
		LinkedTo: []EntityRef{entity}, CreatedBy: first.UploadedBy.Id, CreatedAt: now, UpdatedAt: now,
	}}, nil
}

// Rehydrate arma un documento ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Document { return Document{s: s} }

// Snapshot devuelve el contenido del documento.
func (d Document) Snapshot() Snapshot { return d.s }

// Version busca una versión por su número.
func (d Document) Version(number int) (Version, bool) {
	for _, v := range d.s.Versions {
		if v.Number == number {
			return v, true
		}
	}
	return Version{}, false
}

// IsLinkedTo dice si el documento ya está enlazado a esa entidad.
func (d Document) IsLinkedTo(entity EntityRef) bool {
	return slices.ContainsFunc(d.s.LinkedTo, func(l EntityRef) bool { return l.Type == entity.Type && l.Id == entity.Id })
}

// officeTypes son los tipos de Office que se pueden convertir a PDF para verlos dentro del sistema.
var officeTypes = []string{
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
}

// IsOffice dice si un tipo es de Office: no se ve en el navegador sin convertirlo.
func IsOffice(mediaType string) bool { return slices.Contains(officeTypes, mediaType) }
