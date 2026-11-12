// Package postgres implementa los puertos de document con sqlc.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/document/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/document/application/port"
	"kaoru/internal/feature/document/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los metadatos de los documentos.
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
		Prefix: shared.PrefixDocument, Year: int32(year), //nolint:gosec // un año cabe en int32
	})
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixDocument, year, int(seq)), nil
}

// Insert guarda un documento nuevo con su primera versión y su primer enlace.
func (r *Repository) Insert(ctx context.Context, document domain.Document, linkedBy shared.UserId) error {
	s := document.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	if err := r.queries(ctx).InsertDocument(ctx, sqlcgen.InsertDocumentParams{
		ID: id, Code: s.Code.String(), Title: s.Title, Kind: string(s.Kind), CreatedBy: string(s.CreatedBy), CreatedAt: s.CreatedAt,
	}); err != nil {
		return corepg.TranslateError(err)
	}
	if err := r.insertVersion(ctx, id, s.Versions[0]); err != nil {
		return err
	}
	return r.AddLink(ctx, s.Id, s.LinkedTo[0], linkedBy, s.CreatedAt)
}

// AddVersion agrega una versión con el número siguiente y lo devuelve.
func (r *Repository) AddVersion(ctx context.Context, code shared.Code, version domain.Version) (int, error) {
	q := r.queries(ctx)
	id, err := q.LockDocument(ctx, code.String())
	if err != nil {
		return 0, corepg.TranslateError(err)
	}
	number, err := q.NextVersion(ctx, id)
	if err != nil {
		return 0, corepg.TranslateError(err)
	}
	version.Number = int(number)
	if err := r.insertVersion(ctx, id, version); err != nil {
		return 0, err
	}
	return version.Number, corepg.TranslateError(q.TouchDocument(ctx, sqlcgen.TouchDocumentParams{ID: id, UpdatedAt: version.CreatedAt}))
}

func (r *Repository) insertVersion(ctx context.Context, documentID uuid.UUID, v domain.Version) error {
	id, err := parseID(v.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertVersion(ctx, sqlcgen.InsertVersionParams{
		ID: id, DocumentID: documentID, Version: int32(v.Number), FileName: v.FileName, MediaType: v.MediaType, //nolint:gosec // contador pequeño
		SizeBytes: v.SizeBytes, Checksum: v.Checksum, UploadedBy: string(v.UploadedBy.Id), CreatedAt: v.CreatedAt,
	}))
}

// AddLink enlaza el documento a una entidad; enlazar dos veces no cambia nada.
func (r *Repository) AddLink(ctx context.Context, documentId string, entity domain.EntityRef, by shared.UserId, at time.Time) error {
	id, err := parseID(documentId)
	if err != nil {
		return err
	}
	entityID, err := parseID(entity.Id)
	if err != nil {
		return err
	}
	_, err = r.queries(ctx).InsertLink(ctx, sqlcgen.InsertLinkParams{
		DocumentID: id, EntityType: entity.Type, EntityID: entityID, EntityCode: entity.Code, LinkedBy: string(by), CreatedAt: at,
	})
	return corepg.TranslateError(err)
}

// FindByCode busca un documento con sus versiones y sus enlaces.
func (r *Repository) FindByCode(ctx context.Context, code shared.Code) (domain.Document, error) {
	row, err := r.queries(ctx).GetDocumentByCode(ctx, code.String())
	if err != nil {
		return domain.Document{}, corepg.TranslateError(err)
	}
	documents, err := r.hydrate(ctx, []sqlcgen.Document{row})
	if err != nil {
		return domain.Document{}, err
	}
	return documents[0], nil
}

// ListByEntity devuelve los documentos enlazados a una entidad.
func (r *Repository) ListByEntity(ctx context.Context, entity domain.EntityRef) ([]domain.Document, error) {
	entityID, err := parseID(entity.Id)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListDocumentsByEntity(ctx, sqlcgen.ListDocumentsByEntityParams{EntityType: entity.Type, EntityID: entityID})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	return r.hydrate(ctx, rows)
}

// hydrate agrega a cada documento sus versiones y sus enlaces con dos consultas.
func (r *Repository) hydrate(ctx context.Context, rows []sqlcgen.Document) ([]domain.Document, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	q := r.queries(ctx)
	versionRows, err := q.ListVersions(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	versions := map[uuid.UUID][]domain.Version{}
	for _, v := range versionRows {
		versions[v.DocumentID] = append(versions[v.DocumentID], domain.Version{
			Number: int(v.Version), FileName: v.FileName, MediaType: v.MediaType, SizeBytes: v.SizeBytes, Checksum: v.Checksum,
			UploadedBy: domain.UserRef{Id: shared.UserId(v.UploadedBy), Name: v.UploadedByName}, CreatedAt: v.CreatedAt.UTC(),
		})
	}
	linkRows, err := q.ListLinks(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	links := map[uuid.UUID][]domain.EntityRef{}
	for _, l := range linkRows {
		links[l.DocumentID] = append(links[l.DocumentID], domain.EntityRef{Type: l.EntityType, Id: l.EntityID.String(), Code: l.EntityCode})
	}
	out := make([]domain.Document, len(rows))
	for i, row := range rows {
		code, err := shared.ParseCode(row.Code)
		if err != nil {
			return nil, axis.Wrap(axis.InternalServerError, "código de DOC no válido en la base", err)
		}
		out[i] = domain.Rehydrate(domain.Snapshot{
			Id: row.ID.String(), Code: code, Title: row.Title, Kind: domain.Kind(row.Kind),
			Versions: versions[row.ID], LinkedTo: links[row.ID], CreatedBy: shared.UserId(row.CreatedBy),
			CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
		})
	}
	return out, nil
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}

var _ port.ChecksumIndex = (*Repository)(nil)

// Referenced devuelve, de los checksums dados, los que usa alguna versión de un documento.
func (r *Repository) Referenced(ctx context.Context, checksums []string) ([]string, error) {
	rows, err := sqlcgen.New(r.db.Conn(ctx)).ReferencedChecksums(ctx, checksums)
	return rows, corepg.TranslateError(err)
}
