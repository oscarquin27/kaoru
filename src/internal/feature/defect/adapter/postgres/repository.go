// Package postgres implementa los puertos de defect con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/defect/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/defect/application/port"
	"kaoru/internal/feature/defect/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los DEF.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// NextCode reserva el siguiente ID legible del año.
func (r *Repository) NextCode(ctx context.Context, year int) (shared.Code, error) {
	seq, err := r.queries(ctx).NextCode(ctx, sqlcgen.NextCodeParams{
		Prefix: shared.PrefixDefect, Year: int32(year), //nolint:gosec // un año cabe en int32
	})
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixDefect, year, int(seq)), nil
}

// FindRequirement busca un REQ por su código.
func (r *Repository) FindRequirement(ctx context.Context, code string) (domain.RequirementRef, error) {
	row, err := r.queries(ctx).GetRequirementRef(ctx, code)
	if err != nil {
		return domain.RequirementRef{}, corepg.TranslateError(err)
	}
	return domain.RequirementRef{Id: row.ID.String(), Code: row.Code}, nil
}

// FindRun busca un RUN por su código.
func (r *Repository) FindRun(ctx context.Context, code string) (domain.RunRef, error) {
	row, err := r.queries(ctx).GetRunRef(ctx, code)
	if err != nil {
		return domain.RunRef{}, corepg.TranslateError(err)
	}
	return domain.RunRef{Id: row.ID.String(), Code: row.Code, Result: row.Result}, nil
}

// FindUser busca un usuario por su id.
func (r *Repository) FindUser(ctx context.Context, id shared.UserId) (domain.UserRef, error) {
	name, err := r.queries(ctx).GetUserName(ctx, string(id))
	if err != nil {
		return domain.UserRef{}, corepg.TranslateError(err)
	}
	return domain.UserRef{Id: id, Name: name}, nil
}

// Insert guarda un DEF nuevo.
func (r *Repository) Insert(ctx context.Context, defect domain.Defect) error {
	s := defect.Snapshot()
	ids, err := parseIDs(s.Id, s.Requirement.Id, s.Run.Id, s.State.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertDefect(ctx, sqlcgen.InsertDefectParams{
		ID: ids[0], Code: s.Code.String(), Title: s.Title, Description: s.Description, RequirementID: ids[1],
		TestRunID: ids[2], Severity: string(s.Severity), StateID: ids[3], AssigneeID: assigneeID(s.Assignee),
		CreatedBy: string(s.CreatedBy.Id), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// Update guarda los cambios de un DEF.
func (r *Repository) Update(ctx context.Context, defect domain.Defect) error {
	s := defect.Snapshot()
	ids, err := parseIDs(s.Id, s.State.Id)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateDefect(ctx, sqlcgen.UpdateDefectParams{
		ID: ids[0], Title: s.Title, Description: s.Description, Severity: string(s.Severity), StateID: ids[1],
		AssigneeID: assigneeID(s.Assignee), UpdatedAt: s.UpdatedAt,
	})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "No existe.")
	}
	return nil
}

// FindByCode busca un DEF por su ID legible.
func (r *Repository) FindByCode(ctx context.Context, code shared.Code) (domain.Defect, error) {
	row, err := r.queries(ctx).GetDefectByCode(ctx, code.String())
	if err != nil {
		return domain.Defect{}, corepg.TranslateError(err)
	}
	return toDefect(sqlcgen.ListDefectsRow(row))
}

// List devuelve una página de DEF, del más reciente al más antiguo.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]domain.Defect, int, error) {
	q := r.queries(ctx)
	total, err := q.CountDefects(ctx, sqlcgen.CountDefectsParams{
		StateKey: nullable(f.StateKey), RequirementCode: nullable(f.RequirementCode), Severity: nullable(f.Severity),
		AssigneeID: nullable(f.AssigneeId), OnlyOpen: f.OnlyOpen, Query: nullable(f.Query),
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListDefects(ctx, sqlcgen.ListDefectsParams{
		StateKey: nullable(f.StateKey), RequirementCode: nullable(f.RequirementCode), Severity: nullable(f.Severity),
		AssigneeID: nullable(f.AssigneeId), OnlyOpen: f.OnlyOpen, Query: nullable(f.Query),
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	defects := make([]domain.Defect, len(rows))
	for i, row := range rows {
		if defects[i], err = toDefect(row); err != nil {
			return nil, 0, err
		}
	}
	return defects, int(total), nil
}

func toDefect(row sqlcgen.ListDefectsRow) (domain.Defect, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return domain.Defect{}, axis.Wrap(axis.InternalServerError, "código de DEF no válido en la base", err)
	}
	s := domain.Snapshot{
		Id: row.ID.String(), Code: code, Title: row.Title, Description: row.Description,
		Requirement: domain.RequirementRef{Id: row.RequirementID.String(), Code: row.RequirementCode},
		Run:         domain.RunRef{Id: row.TestRunID.String(), Code: row.RunCode},
		Severity:    domain.Severity(row.Severity),
		State:       domain.State{Id: row.StateID.String(), Key: row.StateKey, Name: row.StateName, IsFinal: row.StateIsFinal},
		CreatedBy:   domain.UserRef{Id: shared.UserId(row.CreatedBy), Name: row.CreatedByName},
		CreatedAt:   row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	if row.AssigneeID != nil {
		s.Assignee = &domain.UserRef{Id: shared.UserId(*row.AssigneeID)}
		if row.AssigneeName != nil {
			s.Assignee.Name = *row.AssigneeName
		}
	}
	return domain.Rehydrate(s), nil
}

func parseIDs(raw ...string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(raw))
	for i, r := range raw {
		id, err := uuid.Parse(r)
		if err != nil {
			return nil, axis.Wrap(axis.InternalServerError, "id no válido", err)
		}
		ids[i] = id
	}
	return ids, nil
}

func assigneeID(assignee *domain.UserRef) *string {
	if assignee == nil {
		return nil
	}
	id := string(assignee.Id)
	return &id
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
