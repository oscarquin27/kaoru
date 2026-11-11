package usecase

import (
	"context"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/audit/application/port"
	shared "kaoru/internal/shared/domain"
)

const (
	exportBatch = 1000
	// MaxExportEntries es el tope de registros de una exportación.
	MaxExportEntries = 200000
)

// ErrExportTooLarge es el error de una exportación que pasa el tope de registros.
var ErrExportTooLarge = axis.New(axis.InvalidAction, "El rango pasa de 200 000 registros. Acota las fechas para exportarlo.")

// ExportEntries devuelve los registros de auditoría de un rango de fechas, del más antiguo al
// más reciente, para exportarlos.
type ExportEntries struct {
	d Deps
}

// ExportEntriesInput es la entrada de ExportEntries. From y To nil no acotan.
type ExportEntriesInput struct {
	Actor shared.Actor
	From  *time.Time
	To    *time.Time
}

// Execute exige el permiso audit.read.
func (uc *ExportEntries) Execute(ctx context.Context, in ExportEntriesInput) ([]port.EntryView, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionRead); err != nil {
		return nil, err
	}
	var all []port.EntryView
	for page := 1; ; page++ {
		batch, total, err := uc.d.Repository.List(ctx, port.Filter{From: in.From, To: in.To, Page: page, Size: exportBatch})
		if err != nil {
			return nil, err
		}
		if total > MaxExportEntries {
			return nil, ErrExportTooLarge
		}
		all = append(all, batch...)
		if len(batch) < exportBatch {
			break
		}
	}
	// La lista viene del más reciente al más antiguo; la exportación va en el orden de la cadena.
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	return all, nil
}
