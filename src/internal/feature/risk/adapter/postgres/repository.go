// Package postgres implementa los puertos de risk con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/risk/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/risk/application/port"
	"kaoru/internal/feature/risk/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee el RISK de cada REQ.
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
		Prefix: shared.PrefixRisk, Year: int32(year), //nolint:gosec // un año cabe en int32
	})
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixRisk, year, int(seq)), nil
}

// FindByRequirement devuelve NOT_FOUND_RESOURCE si el REQ no tiene RISK.
func (r *Repository) FindByRequirement(ctx context.Context, requirementId string) (domain.Assessment, error) {
	requirementID, err := parseID(requirementId)
	if err != nil {
		return domain.Assessment{}, err
	}
	q := r.queries(ctx)
	row, err := q.GetRiskByRequirement(ctx, requirementID)
	if err != nil {
		return domain.Assessment{}, corepg.TranslateError(err)
	}
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return domain.Assessment{}, axis.Wrap(axis.InternalServerError, "código de RISK no válido en la base", err)
	}
	chars, err := q.ListCharacteristics(ctx, row.ID)
	if err != nil {
		return domain.Assessment{}, corepg.TranslateError(err)
	}
	s := domain.Snapshot{
		Id: row.ID.String(), Code: code, RequirementId: row.RequirementID.String(), Level: domain.Level(row.Level),
		Justification: row.Justification, Characteristics: make([]domain.Characteristic, len(chars)),
		AssessedBy: shared.UserId(row.AssessedBy), AssessedByName: row.AssessedByName,
		CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	for i, c := range chars {
		s.Characteristics[i] = domain.Characteristic(c)
	}
	return domain.Rehydrate(s), nil
}

// Insert guarda un RISK nuevo con sus características.
func (r *Repository) Insert(ctx context.Context, assessment domain.Assessment) error {
	s := assessment.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	requirementID, err := parseID(s.RequirementId)
	if err != nil {
		return err
	}
	if err := r.queries(ctx).InsertRisk(ctx, sqlcgen.InsertRiskParams{
		ID: id, Code: s.Code.String(), RequirementID: requirementID, Level: string(s.Level),
		Justification: s.Justification, AssessedBy: string(s.AssessedBy), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}); err != nil {
		return corepg.TranslateError(err)
	}
	return r.insertCharacteristics(ctx, id, s.Characteristics)
}

// Update guarda una reevaluación y reemplaza sus características.
func (r *Repository) Update(ctx context.Context, assessment domain.Assessment) error {
	s := assessment.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	q := r.queries(ctx)
	rows, err := q.UpdateRisk(ctx, sqlcgen.UpdateRiskParams{
		ID: id, Level: string(s.Level), Justification: s.Justification, AssessedBy: string(s.AssessedBy), UpdatedAt: s.UpdatedAt,
	})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "No existe.")
	}
	if err := q.DeleteCharacteristics(ctx, id); err != nil {
		return corepg.TranslateError(err)
	}
	return r.insertCharacteristics(ctx, id, s.Characteristics)
}

func (r *Repository) insertCharacteristics(ctx context.Context, id uuid.UUID, chars []domain.Characteristic) error {
	q := r.queries(ctx)
	for _, c := range chars {
		if err := q.InsertCharacteristic(ctx, sqlcgen.InsertCharacteristicParams{RiskAssessmentID: id, Characteristic: string(c)}); err != nil {
			return corepg.TranslateError(err)
		}
	}
	return nil
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}
