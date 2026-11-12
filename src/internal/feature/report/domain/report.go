// Package domain modela el informe de conformidad (RPT-) de un PKG: su contenido en Markdown,
// armado con lo que había al emitirlo, su huella SHA-256 y la cadena de versiones (regla de negocio 11).
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	shared "kaoru/internal/shared/domain"
)

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// PackageRef es el PKG al que pertenece un informe.
type PackageRef struct {
	Id    string
	Code  string
	Title string
}

// Snapshot es un RPT tal como se emitió. PreviousCode está vacío en la primera versión;
// SupersededBy, mientras ninguna versión posterior lo reemplace. Content solo viene en la ficha.
type Snapshot struct {
	Id           string
	Code         shared.Code
	Package      PackageRef
	Version      int
	PreviousId   string
	PreviousCode string
	SupersededBy string
	Content      string
	Checksum     string
	IssuedBy     UserRef
	CreatedAt    time.Time
}

// Report es un RPT-. Inmutable: no tiene métodos para cambiar su contenido.
type Report struct {
	s Snapshot
}

// NewParams son los datos para emitir un RPT. Previous es la versión vigente que reemplaza; nil si es la primera.
type NewParams struct {
	Id       string
	Code     shared.Code
	Package  PackageRef
	Previous *Snapshot
	Content  string
	IssuedBy shared.UserId
	Now      time.Time
}

// New emite un RPT: su versión sigue a la que reemplaza y su huella es el SHA-256 del Markdown.
func New(p NewParams) Report {
	s := Snapshot{
		Id: p.Id, Code: p.Code, Package: p.Package, Version: 1, Content: p.Content, Checksum: Checksum(p.Content),
		IssuedBy: UserRef{Id: p.IssuedBy}, CreatedAt: p.Now,
	}
	if p.Previous != nil {
		s.Version, s.PreviousId, s.PreviousCode = p.Previous.Version+1, p.Previous.Id, p.Previous.Code.String()
	}
	return Report{s: s}
}

// Rehydrate arma un RPT ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Report { return Report{s: s} }

// Snapshot devuelve el contenido del RPT.
func (r Report) Snapshot() Snapshot { return r.s }

// Checksum es el SHA-256 en hexadecimal de un contenido: la huella del informe.
func Checksum(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
