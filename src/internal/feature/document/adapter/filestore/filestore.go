// Package filestore implementa el puerto FileStore con el almacén local del core.
package filestore

import (
	"context"
	"io"

	corefs "kaoru/internal/core/pkg/filestore"
	"kaoru/internal/feature/document/application/port"
)

// Store adapta el almacén del core al puerto del feature.
type Store struct {
	local *corefs.Local
}

var _ port.FileStore = (*Store)(nil)

// New crea el adaptador.
func New(local *corefs.Local) *Store { return &Store{local: local} }

// Put guarda el contenido y devuelve su checksum.
func (s *Store) Put(ctx context.Context, r io.Reader) (port.Stored, error) {
	stored, err := s.local.Put(ctx, r)
	return port.Stored{Checksum: stored.Checksum, Size: stored.Size}, err
}

// Open abre un archivo por su checksum.
func (s *Store) Open(ctx context.Context, checksum string) (io.ReadCloser, error) {
	return s.local.Open(ctx, checksum)
}

// Detect dice el tipo real de un contenido a partir de sus primeros bytes.
func (s *Store) Detect(head []byte) string { return s.local.Detect(head) }
