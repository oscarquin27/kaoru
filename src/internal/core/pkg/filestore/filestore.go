// Package filestore guarda archivos en un directorio local, nombrados por su SHA-256.
// La interfaz es pequeña a propósito: cambiarla por un almacén S3 compatible no toca a quien la usa.
package filestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"kaoru/internal/core/pkg/axis"
)

// Stored es el resultado de guardar un archivo.
type Stored struct {
	Checksum string
	Size     int64
}

// Local guarda los archivos bajo un directorio: <dir>/<dos primeros caracteres>/<checksum>.
type Local struct {
	dir string
}

var checksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// NewLocal prepara el directorio. Falla si no se puede escribir en él.
func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(filepath.Join(dir, "tmp"), 0o750); err != nil {
		return nil, fmt.Errorf("preparar el directorio de documentos: %w", err)
	}
	return &Local{dir: dir}, nil
}

// Put guarda el contenido y devuelve su checksum. Guardar dos veces lo mismo deja un solo archivo.
func (l *Local) Put(ctx context.Context, r io.Reader) (Stored, error) {
	tmp, err := os.CreateTemp(filepath.Join(l.dir, "tmp"), "subida-*")
	if err != nil {
		return Stored{}, fmt.Errorf("crear archivo temporal: %w", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name()) // no hace nada si ya se movió a su lugar
	}()

	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, hash), &contextReader{ctx: ctx, r: r})
	if err != nil {
		return Stored{}, fmt.Errorf("guardar archivo: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return Stored{}, fmt.Errorf("guardar archivo: %w", err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))
	target := l.path(checksum)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return Stored{}, fmt.Errorf("guardar archivo: %w", err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return Stored{}, fmt.Errorf("guardar archivo: %w", err)
	}
	return Stored{Checksum: checksum, Size: size}, nil
}

// Open abre un archivo por su checksum. Devuelve NOT_FOUND_RESOURCE si no está.
func (l *Local) Open(_ context.Context, checksum string) (io.ReadCloser, error) {
	if !checksumPattern.MatchString(checksum) {
		return nil, axis.New(axis.NotFoundResource, "El archivo no existe.")
	}
	file, err := os.Open(l.path(checksum))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, axis.Wrap(axis.NotFoundResource, "El archivo no existe.", err)
	}
	if err != nil {
		return nil, fmt.Errorf("abrir archivo: %w", err)
	}
	return file, nil
}

// Entry es un archivo guardado.
type Entry struct {
	Checksum   string
	ModifiedAt time.Time
}

// List devuelve los archivos guardados. Los temporales de subidas en curso no cuentan.
func (l *Local) List(ctx context.Context) ([]Entry, error) {
	var entries []Entry
	err := filepath.WalkDir(l.dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() || !checksumPattern.MatchString(d.Name()) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		entries = append(entries, Entry{Checksum: d.Name(), ModifiedAt: info.ModTime()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listar archivos: %w", err)
	}
	return entries, nil
}

// Remove borra un archivo por su checksum. Borrar uno que no está no es un error.
func (l *Local) Remove(_ context.Context, checksum string) error {
	if !checksumPattern.MatchString(checksum) {
		return nil
	}
	if err := os.Remove(l.path(checksum)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("borrar archivo: %w", err)
	}
	return nil
}

// Detect dice el tipo real de un contenido a partir de sus primeros bytes, sin fiarse del nombre.
func (l *Local) Detect(head []byte) string {
	mediaType, _, err := mime.ParseMediaType(http.DetectContentType(head))
	if err != nil {
		return "application/octet-stream"
	}
	return mediaType
}

func (l *Local) path(checksum string) string {
	return filepath.Join(l.dir, checksum[:2], checksum)
}

// contextReader corta la copia si la petición se cancela.
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *contextReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
