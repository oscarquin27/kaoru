// Package postgres implementa los puertos de requirement con sqlc.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/requirement/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los REQ, sus enlaces y su contrato de entrada.
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
		Prefix: shared.PrefixRequirement, Year: int32(year), //nolint:gosec // un año cabe en int32
	})
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixRequirement, year, int(seq)), nil
}

// FindSystem busca un sistema por su clave.
func (r *Repository) FindSystem(ctx context.Context, key string) (domain.System, error) {
	row, err := r.queries(ctx).GetSystem(ctx, key)
	if err != nil {
		return domain.System{}, corepg.TranslateError(err)
	}
	return domain.System{Key: row.Key, Name: row.Name, DefaultRoute: row.DefaultRoute}, nil
}

// FindChange busca un CHG por su código.
func (r *Repository) FindChange(ctx context.Context, code string) (domain.ChangeRef, error) {
	row, err := r.queries(ctx).GetChangeRef(ctx, code)
	if err != nil {
		return domain.ChangeRef{}, corepg.TranslateError(err)
	}
	return domain.ChangeRef{Id: row.ID.String(), Code: row.Code, Title: row.Title, Route: row.Route, SystemKey: row.SystemKey}, nil
}

// FindUser busca un usuario por su id.
func (r *Repository) FindUser(ctx context.Context, id shared.UserId) (domain.UserRef, error) {
	name, err := r.queries(ctx).GetUserName(ctx, string(id))
	if err != nil {
		return domain.UserRef{}, corepg.TranslateError(err)
	}
	return domain.UserRef{Id: id, Name: name}, nil
}

// Insert guarda un REQ nuevo.
func (r *Repository) Insert(ctx context.Context, requirement domain.Requirement) error {
	s := requirement.Snapshot()
	ids, err := parseRequirementIDs(s)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertRequirement(ctx, sqlcgen.InsertRequirementParams{
		ID: ids.id, Code: s.Code.String(), Title: s.Title, Description: s.Description,
		AcceptanceCriteria: s.AcceptanceCriteria, SystemKey: s.System.Key, ChangeID: ids.change, StateID: ids.state,
		OwnerID: ownerID(s.Owner), CreatedBy: string(s.CreatedBy.Id), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// Update guarda los cambios de un REQ.
func (r *Repository) Update(ctx context.Context, requirement domain.Requirement) error {
	s := requirement.Snapshot()
	ids, err := parseRequirementIDs(s)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateRequirement(ctx, sqlcgen.UpdateRequirementParams{
		ID: ids.id, Title: s.Title, Description: s.Description, AcceptanceCriteria: s.AcceptanceCriteria,
		ChangeID: ids.change, StateID: ids.state, OwnerID: ownerID(s.Owner), UpdatedAt: s.UpdatedAt,
	})
	return affected(rows, err)
}

// FindByCode busca un REQ por su ID legible.
func (r *Repository) FindByCode(ctx context.Context, code shared.Code) (domain.Requirement, error) {
	row, err := r.queries(ctx).GetRequirementByCode(ctx, code.String())
	if err != nil {
		return domain.Requirement{}, corepg.TranslateError(err)
	}
	return toRequirement(sqlcgen.ListRequirementsRow(row))
}

// List devuelve una página de REQ.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]domain.Requirement, int, error) {
	q := r.queries(ctx)
	total, err := q.CountRequirements(ctx, sqlcgen.CountRequirementsParams{
		SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey), ChangeCode: nullable(f.ChangeCode),
		OwnerID: nullable(f.OwnerId), Query: nullable(f.Query),
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListRequirements(ctx, sqlcgen.ListRequirementsParams{
		SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey), ChangeCode: nullable(f.ChangeCode),
		OwnerID: nullable(f.OwnerId), Query: nullable(f.Query),
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	requirements := make([]domain.Requirement, len(rows))
	for i, row := range rows {
		if requirements[i], err = toRequirement(row); err != nil {
			return nil, 0, err
		}
	}
	return requirements, int(total), nil
}

// InsertLink guarda un enlace nuevo.
func (r *Repository) InsertLink(ctx context.Context, link domain.Link) error {
	s := link.Snapshot()
	id, err := parseID(s.Id, "enlace")
	if err != nil {
		return err
	}
	requirementID, err := parseID(s.RequirementId, "REQ")
	if err != nil {
		return err
	}
	snapshot, err := parseNullID(s.SnapshotDocumentId, "documento")
	if err != nil {
		return err
	}
	params := sqlcgen.InsertLinkParams{
		ID: id, RequirementID: requirementID, Kind: string(s.Kind), Label: s.Label, Url: s.URL,
		IsOfficial: s.IsOfficial, SnapshotDocumentID: snapshot, AddedBy: string(s.AddedBy.Id), CreatedAt: s.CreatedAt,
	}
	if s.Sheet != nil {
		params.SheetID, params.SheetGid, params.SheetRange = &s.Sheet.Id, nullable(s.Sheet.Gid), nullable(s.Sheet.Range)
	}
	return corepg.TranslateError(r.queries(ctx).InsertLink(ctx, params))
}

// UpdateLink guarda los cambios de un enlace.
func (r *Repository) UpdateLink(ctx context.Context, link domain.Link) error {
	s := link.Snapshot()
	id, err := parseID(s.Id, "enlace")
	if err != nil {
		return err
	}
	snapshot, err := parseNullID(s.SnapshotDocumentId, "documento")
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateLink(ctx, sqlcgen.UpdateLinkParams{
		ID: id, Label: s.Label, IsOfficial: s.IsOfficial, SnapshotDocumentID: snapshot, UpdatedAt: s.UpdatedAt,
	})
	return affected(rows, err)
}

// DeleteLink borra un enlace.
func (r *Repository) DeleteLink(ctx context.Context, linkId string) error {
	id, err := parseID(linkId, "enlace")
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).DeleteLink(ctx, id)
	return affected(rows, err)
}

// FindLink busca un enlace de un REQ. Un id que no es un UUID tampoco existe.
func (r *Repository) FindLink(ctx context.Context, requirementId, linkId string) (domain.Link, error) {
	id, err := uuid.Parse(linkId)
	if err != nil {
		return domain.Link{}, axis.Wrap(axis.NotFoundResource, "No existe.", err)
	}
	requirementID, err := parseID(requirementId, "REQ")
	if err != nil {
		return domain.Link{}, err
	}
	row, err := r.queries(ctx).GetLink(ctx, sqlcgen.GetLinkParams{ID: id, RequirementID: requirementID})
	if err != nil {
		return domain.Link{}, corepg.TranslateError(err)
	}
	return toLink(sqlcgen.ListLinksRow(row)), nil
}

// ListLinks devuelve los enlaces de un REQ en el orden en que se agregaron.
func (r *Repository) ListLinks(ctx context.Context, requirementId string) ([]domain.Link, error) {
	requirementID, err := parseID(requirementId, "REQ")
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListLinks(ctx, requirementID)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	links := make([]domain.Link, len(rows))
	for i, row := range rows {
		links[i] = toLink(row)
	}
	return links, nil
}

// FindDocumentId devuelve la llave del DOC con ese código.
func (r *Repository) FindDocumentId(ctx context.Context, code string) (string, error) {
	id, err := r.queries(ctx).GetDocumentIdByCode(ctx, code)
	if err != nil {
		return "", corepg.TranslateError(err)
	}
	return id.String(), nil
}

// ListEntryContract devuelve los ítems activos del contrato con su estado para el REQ.
func (r *Repository) ListEntryContract(ctx context.Context, requirementId string) ([]domain.EntryItem, error) {
	requirementID, err := parseID(requirementId, "REQ")
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListEntryContract(ctx, requirementID)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	items := make([]domain.EntryItem, len(rows))
	for i, row := range rows {
		items[i] = domain.EntryItem{Key: row.Key, Label: row.Label}
		if row.CheckedBy != nil {
			items[i].Done = true
			items[i].Note = value(row.Note)
			items[i].CheckedBy = &domain.UserRef{Id: shared.UserId(*row.CheckedBy), Name: value(row.CheckedByName)}
			items[i].CheckedAt = utc(row.CheckedAt)
		}
	}
	return items, nil
}

// CheckEntryItem marca un ítem como cumplido.
func (r *Repository) CheckEntryItem(ctx context.Context, requirementId, itemKey, note string, by shared.UserId, at time.Time) error {
	requirementID, err := parseID(requirementId, "REQ")
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).UpsertEntryCheck(ctx, sqlcgen.UpsertEntryCheckParams{
		RequirementID: requirementID, ItemKey: itemKey, Note: note, CheckedBy: string(by), CheckedAt: at,
	}))
}

// UncheckEntryItem quita la marca de un ítem.
func (r *Repository) UncheckEntryItem(ctx context.Context, requirementId, itemKey string) error {
	requirementID, err := parseID(requirementId, "REQ")
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).DeleteEntryCheck(ctx, sqlcgen.DeleteEntryCheckParams{RequirementID: requirementID, ItemKey: itemKey})
	return affected(rows, err)
}

// Lock bloquea la entidad hasta el fin de la transacción en curso.
func (r *Repository) Lock(ctx context.Context, code shared.Code) error {
	_, err := r.queries(ctx).LockRequirement(ctx, code.String())
	return corepg.TranslateError(err)
}
