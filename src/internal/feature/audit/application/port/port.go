// Package port declara lo que necesitan los casos de uso de audit.
package port

import (
	"context"
	"time"

	"kaoru/internal/feature/audit/domain"
)

// Filter acota la consulta de la auditoría. Un campo vacío no filtra.
type Filter struct {
	EntityCode string
	ActorId    string
	From       *time.Time
	To         *time.Time
	Page       int
	Size       int
}

// EntryView es un registro con el nombre de quien lo produjo, para mostrarlo.
type EntryView struct {
	Entry     domain.Entry
	ActorName string
}

// Repository guarda y lee la cadena de auditoría. Solo inserta: nunca edita ni borra.
type Repository interface {
	// LockChain serializa las inserciones hasta el fin de la transacción.
	LockChain(ctx context.Context) error
	// LastHash devuelve el hash del último registro, o vacío si no hay ninguno.
	LastHash(ctx context.Context) (string, error)
	Insert(ctx context.Context, entry domain.Entry) error
	List(ctx context.Context, filter Filter) (entries []EntryView, total int, err error)
	// ChainAfter devuelve hasta limit registros con seq mayor que afterSeq, en orden.
	ChainAfter(ctx context.Context, afterSeq int64, limit int) ([]domain.Entry, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}
