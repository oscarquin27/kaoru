// Package postgres implementa los puertos de planning con sqlc.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/planning/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los hitos.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// Insert guarda un hito nuevo.
func (r *Repository) Insert(ctx context.Context, m domain.Milestone) error {
	s := m.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return corepg.TranslateError(r.queries(ctx).InsertMilestone(ctx, sqlcgen.InsertMilestoneParams{
		ID: id, Title: s.Title, DueDate: date(s.DueDate), Notes: s.Notes, OwnerID: ownerId(s.Owner),
		CompletedAt: s.CompletedAt, CreatedBy: string(s.CreatedBy.Id), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// Update guarda los datos editables de un hito.
func (r *Repository) Update(ctx context.Context, m domain.Milestone) error {
	s := m.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return corepg.TranslateError(r.queries(ctx).UpdateMilestone(ctx, sqlcgen.UpdateMilestoneParams{
		ID: id, Title: s.Title, DueDate: date(s.DueDate), Notes: s.Notes, OwnerID: ownerId(s.Owner),
		CompletedAt: s.CompletedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// Lock bloquea el hito; NOT_FOUND_RESOURCE si no existe o el id está mal formado.
func (r *Repository) Lock(ctx context.Context, raw string) error {
	id, err := parseID(raw)
	if err != nil {
		return err
	}
	_, err = r.queries(ctx).LockMilestone(ctx, id)
	return corepg.TranslateError(err)
}

// Find devuelve el hito con los nombres de su responsable y de quien lo registró.
func (r *Repository) Find(ctx context.Context, raw string) (domain.Milestone, error) {
	id, err := parseID(raw)
	if err != nil {
		return domain.Milestone{}, err
	}
	row, err := r.queries(ctx).GetMilestone(ctx, id)
	if err != nil {
		return domain.Milestone{}, corepg.TranslateError(err)
	}
	return toMilestone(row), nil
}

// List devuelve una página de hitos por fecha.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]domain.Milestone, int, error) {
	q := r.queries(ctx)
	var status *string
	if f.Status != "" {
		status = &f.Status
	}
	total, err := q.CountMilestones(ctx, status)
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListMilestones(ctx, sqlcgen.ListMilestonesParams{
		Status: status, PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	out := make([]domain.Milestone, len(rows))
	for i, row := range rows {
		out[i] = toMilestone(sqlcgen.GetMilestoneRow(row))
	}
	return out, int(total), nil
}

// Pending devuelve los hitos sin cumplir.
func (r *Repository) Pending(ctx context.Context) ([]domain.Milestone, error) {
	rows, err := r.queries(ctx).ListPendingMilestones(ctx)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.Milestone, len(rows))
	for i, row := range rows {
		out[i] = toMilestone(sqlcgen.GetMilestoneRow(row))
	}
	return out, nil
}

// FindUser devuelve NOT_FOUND_RESOURCE si el usuario no existe.
func (r *Repository) FindUser(ctx context.Context, id shared.UserId) (domain.UserRef, error) {
	row, err := r.queries(ctx).GetUser(ctx, string(id))
	if err != nil {
		return domain.UserRef{}, corepg.TranslateError(err)
	}
	return domain.UserRef{Id: shared.UserId(row.ID), Name: row.DisplayName}, nil
}

func toMilestone(row sqlcgen.GetMilestoneRow) domain.Milestone {
	s := domain.Snapshot{
		Id: row.ID.String(), Title: row.Title, DueDate: row.DueDate.Time.UTC(), Notes: row.Notes,
		CreatedBy: domain.UserRef{Id: shared.UserId(row.CreatedBy), Name: row.CreatedByName},
		CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	if row.OwnerID != nil {
		s.Owner = &domain.UserRef{Id: shared.UserId(*row.OwnerID)}
		if row.OwnerName != nil {
			s.Owner.Name = *row.OwnerName
		}
	}
	if row.CompletedAt != nil {
		at := row.CompletedAt.UTC()
		s.CompletedAt = &at
	}
	return domain.Rehydrate(s)
}

func date(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

func ownerId(u *domain.UserRef) *string {
	if u == nil {
		return nil
	}
	id := string(u.Id)
	return &id
}

// parseID trata un id mal formado como uno que no existe.
func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.New(axis.NotFoundResource, "El hito no existe.")
	}
	return id, nil
}
