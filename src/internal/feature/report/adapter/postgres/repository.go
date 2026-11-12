// Package postgres implementa los puertos de report con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/report/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/report/application/port"
	"kaoru/internal/feature/report/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los RPT.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// NextCode reserva el siguiente ID legible del año para ese prefijo.
func (r *Repository) NextCode(ctx context.Context, prefix string, year int) (shared.Code, error) {
	seq, err := r.queries(ctx).NextCode(ctx, sqlcgen.NextCodeParams{Prefix: prefix, Year: int32(year)}) //nolint:gosec // un año cabe en int32
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(prefix, year, int(seq)), nil
}

// FindPackage devuelve NOT_FOUND_RESOURCE si el PKG no existe.
func (r *Repository) FindPackage(ctx context.Context, code string) (domain.PackageRef, error) {
	row, err := r.queries(ctx).GetPackageRef(ctx, code)
	if err != nil {
		return domain.PackageRef{}, corepg.TranslateError(err)
	}
	return domain.PackageRef{Id: row.ID.String(), Code: row.Code, Title: row.Title}, nil
}

// LockPackage bloquea el PKG hasta el fin de la transacción en curso.
func (r *Repository) LockPackage(ctx context.Context, packageId string) error {
	id, err := parseID(packageId)
	if err != nil {
		return err
	}
	_, err = r.queries(ctx).LockPackage(ctx, id)
	return corepg.TranslateError(err)
}

// Latest devuelve la versión vigente del RPT del PKG, o NOT_FOUND_RESOURCE si no tiene.
func (r *Repository) Latest(ctx context.Context, packageId string) (domain.Report, error) {
	id, err := parseID(packageId)
	if err != nil {
		return domain.Report{}, err
	}
	row, err := r.queries(ctx).GetLatestReport(ctx, id)
	if err != nil {
		return domain.Report{}, corepg.TranslateError(err)
	}
	return toReport(sqlcgen.GetReportByCodeRow(row)), nil
}

// Insert guarda un RPT; DUPLICATE_RESOURCE si su versión ya existe.
func (r *Repository) Insert(ctx context.Context, report domain.Report) error {
	s := report.Snapshot()
	ids, err := parseIDs(s.Id, s.Package.Id)
	if err != nil {
		return err
	}
	previous := uuid.NullUUID{}
	if s.PreviousId != "" {
		id, err := parseID(s.PreviousId)
		if err != nil {
			return err
		}
		previous = uuid.NullUUID{UUID: id, Valid: true}
	}
	return corepg.TranslateError(r.queries(ctx).InsertReport(ctx, sqlcgen.InsertReportParams{
		ID: ids[0], Code: s.Code.String(), PackageID: ids[1], Version: int32(s.Version), //nolint:gosec // una versión cabe en int32
		PreviousID: previous, Content: s.Content, Checksum: s.Checksum, IssuedBy: string(s.IssuedBy.Id), CreatedAt: s.CreatedAt,
	}))
}

// FindByCode devuelve el RPT con su contenido, o NOT_FOUND_RESOURCE.
func (r *Repository) FindByCode(ctx context.Context, code shared.Code) (domain.Report, error) {
	row, err := r.queries(ctx).GetReportByCode(ctx, code.String())
	if err != nil {
		return domain.Report{}, corepg.TranslateError(err)
	}
	return toReport(row), nil
}

// List devuelve una página de RPT sin su contenido.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]domain.Report, int, error) {
	q := r.queries(ctx)
	total, err := q.CountReports(ctx, nullable(f.PackageCode))
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListReports(ctx, sqlcgen.ListReportsParams{
		PackageCode: nullable(f.PackageCode),
		PageSize:    int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	reports := make([]domain.Report, len(rows))
	for i, row := range rows {
		reports[i] = toReport(sqlcgen.GetReportByCodeRow{
			ID: row.ID, Code: row.Code, Version: row.Version, PreviousID: row.PreviousID, PreviousCode: row.PreviousCode,
			SupersededBy: row.SupersededBy, Checksum: row.Checksum, IssuedBy: row.IssuedBy, IssuedByName: row.IssuedByName,
			CreatedAt: row.CreatedAt, PackageID: row.PackageID, PackageCode: row.PackageCode, PackageTitle: row.PackageTitle,
		})
	}
	return reports, int(total), nil
}

// UserName devuelve el nombre que se muestra de un usuario.
func (r *Repository) UserName(ctx context.Context, id shared.UserId) (string, error) {
	name, err := r.queries(ctx).GetUserName(ctx, string(id))
	return name, corepg.TranslateError(err)
}

func toReport(row sqlcgen.GetReportByCodeRow) domain.Report {
	code, _ := shared.ParseCode(row.Code)
	s := domain.Snapshot{
		Id: row.ID.String(), Code: code, Version: int(row.Version), SupersededBy: row.SupersededBy,
		Package: domain.PackageRef{Id: row.PackageID.String(), Code: row.PackageCode, Title: row.PackageTitle},
		Content: row.Content, Checksum: row.Checksum,
		IssuedBy: domain.UserRef{Id: shared.UserId(row.IssuedBy), Name: row.IssuedByName}, CreatedAt: row.CreatedAt.UTC(),
	}
	if row.PreviousID.Valid {
		s.PreviousId = row.PreviousID.UUID.String()
	}
	if row.PreviousCode != nil {
		s.PreviousCode = *row.PreviousCode
	}
	return domain.Rehydrate(s)
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}

func parseIDs(raw ...string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(raw))
	for i, r := range raw {
		id, err := parseID(r)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
