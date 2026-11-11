// Package postgres implementa los puertos de audit con sqlc.
package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/audit/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/audit/application/port"
	"kaoru/internal/feature/audit/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee la cadena de auditoría.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// LockChain serializa las inserciones hasta el fin de la transacción.
func (r *Repository) LockChain(ctx context.Context) error {
	return corepg.TranslateError(r.queries(ctx).LockChain(ctx))
}

// LastHash devuelve el hash del último registro, o vacío si no hay ninguno.
func (r *Repository) LastHash(ctx context.Context) (string, error) {
	hash, err := r.queries(ctx).LastHash(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return hash, corepg.TranslateError(err)
}

// Insert guarda un registro ya encadenado.
func (r *Repository) Insert(ctx context.Context, e domain.Entry) error {
	id, err := uuid.Parse(e.Id)
	if err != nil {
		return axis.Wrap(axis.InternalServerError, "id de auditoría no válido", err)
	}
	entityID, err := uuid.Parse(e.EntityId)
	if err != nil {
		return axis.Wrap(axis.InternalServerError, "id de entidad no válido", err)
	}
	return corepg.TranslateError(r.queries(ctx).InsertEntry(ctx, sqlcgen.InsertEntryParams{
		ID:         id,
		OccurredAt: e.OccurredAt,
		ActorID:    string(e.ActorId),
		EntityType: e.EntityType,
		EntityID:   entityID,
		EntityCode: nullable(e.EntityCode),
		Action:     string(e.Action),
		Field:      nullable(e.Field),
		OldValue:   nullable(e.OldValue),
		NewValue:   nullable(e.NewValue),
		Reason:     nullable(e.Reason),
		PrevHash:   nullable(e.PrevHash),
		Hash:       e.Hash,
	}))
}

// List devuelve una página de registros, del más reciente al más antiguo.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]port.EntryView, int, error) {
	q := r.queries(ctx)
	total, err := q.CountEntries(ctx, sqlcgen.CountEntriesParams{
		EntityCode: nullable(f.EntityCode), ActorID: nullable(f.ActorId), FromTime: f.From, ToTime: f.To,
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListEntries(ctx, sqlcgen.ListEntriesParams{
		EntityCode: nullable(f.EntityCode), ActorID: nullable(f.ActorId), FromTime: f.From, ToTime: f.To,
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	views := make([]port.EntryView, len(rows))
	for i, row := range rows {
		views[i] = port.EntryView{
			ActorName: row.ActorName,
			Entry: toEntry(sqlcgen.AuditEntry{
				ID: row.ID, Seq: row.Seq, OccurredAt: row.OccurredAt, ActorID: row.ActorID,
				EntityType: row.EntityType, EntityID: row.EntityID, EntityCode: row.EntityCode,
				Action: row.Action, Field: row.Field, OldValue: row.OldValue, NewValue: row.NewValue,
				Reason: row.Reason, PrevHash: row.PrevHash, Hash: row.Hash,
			}),
		}
	}
	return views, int(total), nil
}

// ChainAfter devuelve hasta limit registros con seq mayor que afterSeq, en orden.
func (r *Repository) ChainAfter(ctx context.Context, afterSeq int64, limit int) ([]domain.Entry, error) {
	rows, err := r.queries(ctx).ListChainAfter(ctx, sqlcgen.ListChainAfterParams{
		AfterSeq: &afterSeq, BatchSize: int32(limit), //nolint:gosec // constante pequeña
	})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	entries := make([]domain.Entry, len(rows))
	for i, row := range rows {
		entries[i] = toEntry(row)
	}
	return entries, nil
}

func toEntry(row sqlcgen.AuditEntry) domain.Entry {
	e := domain.Entry{
		Id:         row.ID.String(),
		OccurredAt: row.OccurredAt.UTC(),
		ActorId:    shared.UserId(row.ActorID),
		EntityType: row.EntityType,
		EntityId:   row.EntityID.String(),
		EntityCode: value(row.EntityCode),
		Action:     domain.Action(row.Action),
		Field:      value(row.Field),
		OldValue:   value(row.OldValue),
		NewValue:   value(row.NewValue),
		Reason:     value(row.Reason),
		PrevHash:   value(row.PrevHash),
		Hash:       row.Hash,
	}
	if row.Seq != nil {
		e.Seq = *row.Seq
	}
	return e
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func value(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
