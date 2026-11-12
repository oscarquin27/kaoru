package usecase

import (
	"context"
	"errors"
	"slices"
	"time"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/feature/document/application/port"
)

// orphanMinAge es la edad mínima de un archivo para considerarlo huérfano: el archivo se
// guarda antes que su registro, así que uno reciente puede ser una subida en curso.
const orphanMinAge = 24 * time.Hour

// PruneOrphans limpia los archivos que quedaron sin registro porque la subida falló después
// de guardarlos. Es una tarea de mantenimiento: la corre quien administra el servidor, no un usuario.
type PruneOrphans struct {
	files port.FileInventory
	index port.ChecksumIndex
	clock clock.Clock
}

// NewPruneOrphans arma el caso de uso; falla si falta una dependencia.
func NewPruneOrphans(files port.FileInventory, index port.ChecksumIndex, c clock.Clock) (*PruneOrphans, error) {
	if files == nil || index == nil || c == nil {
		return nil, errors.New("document: faltan dependencias para limpiar archivos")
	}
	return &PruneOrphans{files: files, index: index, clock: c}, nil
}

// PruneOrphansOutput es el resultado de la limpieza.
type PruneOrphansOutput struct {
	// Scanned es cuántos archivos hay en el almacén.
	Scanned int
	// Orphans son los checksums sin registro con más de un día.
	Orphans []string
	// Removed dice si se borraron o solo se listaron.
	Removed bool
}

// Execute lista los huérfanos y, con apply, los borra.
func (uc *PruneOrphans) Execute(ctx context.Context, apply bool) (PruneOrphansOutput, error) {
	files, err := uc.files.List(ctx)
	if err != nil {
		return PruneOrphansOutput{}, err
	}
	out := PruneOrphansOutput{Scanned: len(files), Orphans: []string{}, Removed: apply}
	limit := uc.clock.Now().Add(-orphanMinAge)
	var old []string
	for _, f := range files {
		if f.ModifiedAt.Before(limit) {
			old = append(old, f.Checksum)
		}
	}
	if len(old) == 0 {
		return out, nil
	}
	referenced, err := uc.index.Referenced(ctx, old)
	if err != nil {
		return PruneOrphansOutput{}, err
	}
	for _, checksum := range old {
		if slices.Contains(referenced, checksum) {
			continue
		}
		if apply {
			if err := uc.files.Remove(ctx, checksum); err != nil {
				return PruneOrphansOutput{}, err
			}
		}
		out.Orphans = append(out.Orphans, checksum)
	}
	return out, nil
}
