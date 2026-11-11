package postgres

import (
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/requirement/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

func toRequirement(row sqlcgen.ListRequirementsRow) (domain.Requirement, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return domain.Requirement{}, axis.Wrap(axis.InternalServerError, "código de REQ no válido en la base", err)
	}
	s := domain.Snapshot{
		Id: row.ID.String(), Code: code, Title: row.Title, Description: row.Description,
		AcceptanceCriteria: row.AcceptanceCriteria,
		System:             domain.System{Key: row.SystemKey, Name: row.SystemName, DefaultRoute: row.SystemDefaultRoute},
		State:              domain.State{Id: row.StateID.String(), Key: row.StateKey, Name: row.StateName},
		CreatedBy:          domain.UserRef{Id: shared.UserId(row.CreatedBy), Name: row.CreatedByName},
		CreatedAt:          row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	if row.ChangeID.Valid {
		s.Change = &domain.ChangeRef{
			Id: row.ChangeID.UUID.String(), Code: value(row.ChangeCode), Title: value(row.ChangeTitle),
			Route: value(row.ChangeRoute), SystemKey: row.SystemKey,
		}
	}
	if row.OwnerID != nil {
		s.Owner = &domain.UserRef{Id: shared.UserId(*row.OwnerID), Name: value(row.OwnerName)}
	}
	return domain.Rehydrate(s), nil
}

func toLink(row sqlcgen.ListLinksRow) domain.Link {
	s := domain.LinkSnapshot{
		Id: row.ID.String(), RequirementId: row.RequirementID.String(), Kind: domain.LinkKind(row.Kind),
		Label: row.Label, URL: row.Url, IsOfficial: row.IsOfficial, SnapshotCode: value(row.SnapshotCode),
		AddedBy:   domain.UserRef{Id: shared.UserId(row.AddedBy), Name: row.AddedByName},
		CreatedAt: row.CreatedAt.UTC(),
	}
	if row.SheetID != nil {
		s.Sheet = &domain.SheetRef{Id: *row.SheetID, Gid: value(row.SheetGid), Range: value(row.SheetRange)}
	}
	if row.SnapshotDocumentID.Valid {
		s.SnapshotDocumentId = row.SnapshotDocumentID.UUID.String()
	}
	return domain.RehydrateLink(s)
}

type requirementIDs struct {
	id     uuid.UUID
	state  uuid.UUID
	change uuid.NullUUID
}

func parseRequirementIDs(s domain.Snapshot) (requirementIDs, error) {
	var ids requirementIDs
	var err error
	if ids.id, err = parseID(s.Id, "REQ"); err != nil {
		return ids, err
	}
	if ids.state, err = parseID(s.State.Id, "estado"); err != nil {
		return ids, err
	}
	if s.Change != nil {
		ids.change, err = parseNullID(s.Change.Id, "CHG")
	}
	return ids, err
}

func parseID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id de "+what+" no válido", err)
	}
	return id, nil
}

// parseNullID convierte un id opcional; vacío es NULL.
func parseNullID(raw, what string) (uuid.NullUUID, error) {
	if raw == "" {
		return uuid.NullUUID{}, nil
	}
	id, err := parseID(raw, what)
	return uuid.NullUUID{UUID: id, Valid: err == nil}, err
}

// affected traduce el error y devuelve NOT_FOUND_RESOURCE si la sentencia no tocó ninguna fila.
func affected(rows int64, err error) error {
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "No existe.")
	}
	return nil
}

func ownerID(owner *domain.UserRef) *string {
	if owner == nil {
		return nil
	}
	id := string(owner.Id)
	return &id
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

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
