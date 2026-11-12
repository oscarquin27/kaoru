package api

import (
	"encoding/json"
	"time"

	"kaoru/internal/feature/requirement/domain"
)

type systemDTO struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	DefaultRoute string `json:"defaultRoute"`
}

type stateDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type entityRefDTO struct {
	Type  string `json:"type"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

type requirementDTO struct {
	Code               string        `json:"code"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	AcceptanceCriteria string        `json:"acceptanceCriteria"`
	System             systemDTO     `json:"system"`
	Change             *entityRefDTO `json:"change,omitempty"`
	State              stateDTO      `json:"state"`
	Owner              *userRefDTO   `json:"owner,omitempty"`
	CreatedBy          userRefDTO    `json:"createdBy"`
	CreatedAt          time.Time     `json:"createdAt"`
	UpdatedAt          time.Time     `json:"updatedAt"`
}

type pageDTO struct {
	Items []requirementDTO `json:"items"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
	Total int              `json:"total"`
}

type createDTO struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptanceCriteria"`
	System             string `json:"system"`
	Change             string `json:"change"`
	Owner              string `json:"owner"`
}

// nullableString distingue «no vino» de «vino null» en un PATCH.
type nullableString struct {
	set   bool
	value string
}

func (n *nullableString) UnmarshalJSON(data []byte) error {
	n.set = true
	if string(data) == "null" {
		return nil
	}
	return json.Unmarshal(data, &n.value)
}

type patchDTO struct {
	Title              *string        `json:"title"`
	Description        *string        `json:"description"`
	AcceptanceCriteria *string        `json:"acceptanceCriteria"`
	Change             nullableString `json:"change"`
	Owner              nullableString `json:"owner"`
}

type transitionDTO struct {
	To     string `json:"to"`
	Reason string `json:"reason"`
}

type sheetDTO struct {
	ID    string `json:"id"`
	Gid   string `json:"gid,omitempty"`
	Range string `json:"range,omitempty"`
}

type linkDTO struct {
	ID               string     `json:"id"`
	Kind             string     `json:"kind"`
	Label            string     `json:"label"`
	URL              string     `json:"url"`
	Sheet            *sheetDTO  `json:"sheet,omitempty"`
	IsOfficial       bool       `json:"isOfficial"`
	SnapshotDocument string     `json:"snapshotDocument,omitempty"`
	AddedBy          userRefDTO `json:"addedBy"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type linkCreateDTO struct {
	URL        string `json:"url"`
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	IsOfficial bool   `json:"isOfficial"`
}

type linkPatchDTO struct {
	Label            *string        `json:"label"`
	IsOfficial       *bool          `json:"isOfficial"`
	SnapshotDocument nullableString `json:"snapshotDocument"`
}

type entryItemDTO struct {
	Key       string      `json:"key"`
	Label     string      `json:"label"`
	Done      bool        `json:"done"`
	Note      string      `json:"note,omitempty"`
	CheckedBy *userRefDTO `json:"checkedBy,omitempty"`
	CheckedAt *time.Time  `json:"checkedAt,omitempty"`
}

type entryCheckDTO struct {
	Note string `json:"note"`
}

func toUserRef(u *domain.UserRef) *userRefDTO {
	if u == nil {
		return nil
	}
	return &userRefDTO{ID: string(u.Id), DisplayName: u.Name}
}

func toDTO(s domain.Snapshot) requirementDTO {
	dto := requirementDTO{
		Code: s.Code.String(), Title: s.Title, Description: s.Description, AcceptanceCriteria: s.AcceptanceCriteria,
		System:    systemDTO{Key: s.System.Key, Name: s.System.Name, DefaultRoute: s.System.DefaultRoute},
		State:     stateDTO{Key: s.State.Key, Name: s.State.Name},
		Owner:     toUserRef(s.Owner),
		CreatedBy: *toUserRef(&s.CreatedBy),
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if s.Change != nil {
		dto.Change = &entityRefDTO{Type: "change", Code: s.Change.Code, Title: s.Change.Title}
	}
	return dto
}

func toLinkDTO(s domain.LinkSnapshot) linkDTO {
	dto := linkDTO{
		ID: s.Id, Kind: string(s.Kind), Label: s.Label, URL: s.URL, IsOfficial: s.IsOfficial,
		SnapshotDocument: s.SnapshotCode, AddedBy: *toUserRef(&s.AddedBy), CreatedAt: s.CreatedAt,
	}
	if s.Sheet != nil {
		dto.Sheet = &sheetDTO{ID: s.Sheet.Id, Gid: s.Sheet.Gid, Range: s.Sheet.Range}
	}
	return dto
}

func toEntryItemDTO(item domain.EntryItem) entryItemDTO {
	return entryItemDTO{
		Key: item.Key, Label: item.Label, Done: item.Done, Note: item.Note,
		CheckedBy: toUserRef(item.CheckedBy), CheckedAt: item.CheckedAt,
	}
}

type componentRefDTO struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// componentsDTO es el cuerpo que declara los componentes de un REQ; sin `components` no vale.
type componentsDTO struct {
	Components []string `json:"components"`
}

func toComponentDTOs(refs []domain.ComponentRef) []componentRefDTO {
	items := make([]componentRefDTO, len(refs))
	for i, r := range refs {
		items[i] = componentRefDTO{Code: r.Code, Name: r.Name, Kind: r.Kind}
	}
	return items
}
