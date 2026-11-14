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

// MilestonesBetween devuelve los hitos con fecha en el rango.
func (r *Repository) MilestonesBetween(ctx context.Context, from, to time.Time) ([]domain.Milestone, error) {
	rows, err := r.queries(ctx).ListMilestonesBetween(ctx, sqlcgen.ListMilestonesBetweenParams{FromDate: date(from), ToDate: date(to)})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.Milestone, len(rows))
	for i, row := range rows {
		out[i] = toMilestone(sqlcgen.GetMilestoneRow(row))
	}
	return out, nil
}

// NextActivityCode reserva el siguiente ACT- del año.
func (r *Repository) NextActivityCode(ctx context.Context, year int) (shared.Code, error) {
	seq, err := r.queries(ctx).NextActivityCode(ctx, int32(year)) //nolint:gosec // un año cabe en int32
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixActivity, year, int(seq)), nil
}

// InsertActivity guarda la actividad y sus operadores.
func (r *Repository) InsertActivity(ctx context.Context, a domain.Activity) error {
	s := a.Snapshot()
	id, err := uuid.Parse(s.Id)
	if err != nil {
		return axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	changeId, err := changeID(s.Change)
	if err != nil {
		return err
	}
	q := r.queries(ctx)
	if err := q.InsertActivity(ctx, sqlcgen.InsertActivityParams{
		ID: id, Code: s.Code, Kind: string(s.Kind), Title: s.Title, ChangeID: changeId, StartDate: date(s.Start),
		EndDate: date(s.End), Notes: s.Notes, CancelledAt: s.CancelledAt, CreatedBy: string(s.CreatedBy.Id),
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}); err != nil {
		return corepg.TranslateError(err)
	}
	return r.saveOperators(ctx, id, s.Operators)
}

// UpdateActivity guarda los datos editables y reemplaza los operadores.
func (r *Repository) UpdateActivity(ctx context.Context, a domain.Activity) error {
	s := a.Snapshot()
	id, err := uuid.Parse(s.Id)
	if err != nil {
		return axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	changeId, err := changeID(s.Change)
	if err != nil {
		return err
	}
	q := r.queries(ctx)
	if err := q.UpdateActivity(ctx, sqlcgen.UpdateActivityParams{
		ID: id, Kind: string(s.Kind), Title: s.Title, ChangeID: changeId, StartDate: date(s.Start), EndDate: date(s.End),
		Notes: s.Notes, CancelledAt: s.CancelledAt, UpdatedAt: s.UpdatedAt,
	}); err != nil {
		return corepg.TranslateError(err)
	}
	if err := q.DeleteActivityOperators(ctx, id); err != nil {
		return corepg.TranslateError(err)
	}
	return r.saveOperators(ctx, id, s.Operators)
}

func (r *Repository) saveOperators(ctx context.Context, id uuid.UUID, operators []domain.UserRef) error {
	if len(operators) == 0 {
		return nil
	}
	ids := make([]string, len(operators))
	for i, op := range operators {
		ids[i] = string(op.Id)
	}
	return corepg.TranslateError(r.queries(ctx).InsertActivityOperators(ctx, sqlcgen.InsertActivityOperatorsParams{ActivityID: id, UserIds: ids}))
}

// LockActivity bloquea la actividad; NOT_FOUND_RESOURCE si no existe.
func (r *Repository) LockActivity(ctx context.Context, code string) error {
	_, err := r.queries(ctx).LockActivity(ctx, code)
	return corepg.TranslateError(err)
}

// FindActivity devuelve la actividad con su CHG y sus operadores.
func (r *Repository) FindActivity(ctx context.Context, code string) (domain.Activity, error) {
	row, err := r.queries(ctx).GetActivity(ctx, code)
	if err != nil {
		return domain.Activity{}, corepg.TranslateError(err)
	}
	return toActivity(row), nil
}

// ListActivities devuelve una página de actividades.
func (r *Repository) ListActivities(ctx context.Context, f port.ActivityFilter) ([]domain.Activity, int, error) {
	q := r.queries(ctx)
	count := sqlcgen.CountActivitiesParams{
		Operator: text(f.Operator), ChangeCode: text(f.ChangeCode), Kind: text(f.Kind), Status: text(f.Status),
		Today: date(f.Today), FromDate: optionalDate(f.From), ToDate: optionalDate(f.To),
	}
	total, err := q.CountActivities(ctx, count)
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListActivities(ctx, sqlcgen.ListActivitiesParams{
		Operator: count.Operator, ChangeCode: count.ChangeCode, Kind: count.Kind, Status: count.Status, Today: count.Today,
		FromDate: count.FromDate, ToDate: count.ToDate,
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	out := make([]domain.Activity, len(rows))
	for i, row := range rows {
		out[i] = toActivity(sqlcgen.GetActivityRow(row))
	}
	return out, int(total), nil
}

// ActivitiesBetween devuelve las no canceladas que tocan el rango.
func (r *Repository) ActivitiesBetween(ctx context.Context, from, to time.Time) ([]domain.Activity, error) {
	return r.between(ctx, from, to, nil)
}

// ActivitiesOfOperators devuelve las no canceladas de esos operadores que tocan el rango.
func (r *Repository) ActivitiesOfOperators(ctx context.Context, operators []shared.UserId, from, to time.Time) ([]domain.Activity, error) {
	if len(operators) == 0 {
		return nil, nil
	}
	ids := make([]string, len(operators))
	for i, id := range operators {
		ids[i] = string(id)
	}
	return r.between(ctx, from, to, ids)
}

func (r *Repository) between(ctx context.Context, from, to time.Time, operators []string) ([]domain.Activity, error) {
	rows, err := r.queries(ctx).ListActivitiesBetween(ctx, sqlcgen.ListActivitiesBetweenParams{
		FromDate: date(from), ToDate: date(to), Operators: operators,
	})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.Activity, len(rows))
	for i, row := range rows {
		out[i] = toActivity(sqlcgen.GetActivityRow(row))
	}
	return out, nil
}

// Certifications devuelve las actividades de certificación del CHG.
func (r *Repository) Certifications(ctx context.Context, changeCode string) ([]domain.Activity, error) {
	rows, err := r.queries(ctx).ListCertifications(ctx, changeCode)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.Activity, len(rows))
	for i, row := range rows {
		out[i] = toActivity(sqlcgen.GetActivityRow(row))
	}
	return out, nil
}

// FindChange devuelve NOT_FOUND_RESOURCE si el CHG no existe.
func (r *Repository) FindChange(ctx context.Context, code string) (domain.ChangeRef, error) {
	row, err := r.queries(ctx).GetChange(ctx, code)
	if err != nil {
		return domain.ChangeRef{}, corepg.TranslateError(err)
	}
	return domain.ChangeRef{Id: row.ID.String(), Code: row.Code, Title: row.Title, System: row.SystemName}, nil
}

// FindUsers devuelve los usuarios que existen.
func (r *Repository) FindUsers(ctx context.Context, ids []shared.UserId) ([]domain.UserRef, error) {
	raw := make([]string, len(ids))
	for i, id := range ids {
		raw[i] = string(id)
	}
	rows, err := r.queries(ctx).GetUsers(ctx, raw)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.UserRef, len(rows))
	for i, row := range rows {
		out[i] = domain.UserRef{Id: shared.UserId(row.ID), Name: row.DisplayName}
	}
	return out, nil
}

func toActivity(row sqlcgen.GetActivityRow) domain.Activity {
	s := domain.ActivitySnapshot{
		Id: row.ID.String(), Code: row.Code, Kind: domain.ActivityKind(row.Kind), Title: row.Title,
		Start: row.StartDate.Time.UTC(), End: row.EndDate.Time.UTC(), Notes: row.Notes,
		CreatedBy: domain.UserRef{Id: shared.UserId(row.CreatedBy), Name: row.CreatedByName},
		CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
		Operators: make([]domain.UserRef, len(row.OperatorIds)),
	}
	if row.ChangeID.Valid && row.ChangeCode != nil {
		s.Change = &domain.ChangeRef{Id: row.ChangeID.UUID.String(), Code: *row.ChangeCode}
		if row.ChangeTitle != nil {
			s.Change.Title = *row.ChangeTitle
		}
		if row.SystemName != nil {
			s.Change.System = *row.SystemName
		}
	}
	for i, id := range row.OperatorIds {
		s.Operators[i] = domain.UserRef{Id: shared.UserId(id)}
		if i < len(row.OperatorNames) {
			s.Operators[i].Name = row.OperatorNames[i]
		}
	}
	if row.CancelledAt != nil {
		at := row.CancelledAt.UTC()
		s.CancelledAt = &at
	}
	return domain.RehydrateActivity(s)
}

func changeID(c *domain.ChangeRef) (uuid.NullUUID, error) {
	if c == nil {
		return uuid.NullUUID{}, nil
	}
	id, err := uuid.Parse(c.Id)
	if err != nil {
		return uuid.NullUUID{}, axis.Wrap(axis.InternalServerError, "id de CHG no válido", err)
	}
	return uuid.NullUUID{UUID: id, Valid: true}, nil
}

func text(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return date(*t)
}
