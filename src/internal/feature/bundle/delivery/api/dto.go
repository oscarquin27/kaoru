package api

import (
	"time"

	"kaoru/internal/feature/bundle/application/usecase"
	"kaoru/internal/feature/bundle/domain"
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

type buildRefDTO struct {
	Code     string `json:"code"`
	Version  string `json:"version"`
	Checksum string `json:"checksum"`
}

type requirementRefDTO struct {
	Code  string   `json:"code"`
	Title string   `json:"title"`
	State stateDTO `json:"state"`
}

type conditionDTO struct {
	RuleKey string   `json:"ruleKey"`
	Message string   `json:"message"`
	Blocks  bool     `json:"blocks"`
	Related []string `json:"related"`
}

type missingDTO struct {
	To         stateDTO       `json:"to"`
	Conditions []conditionDTO `json:"conditions"`
}

type packageDTO struct {
	Code         string              `json:"code"`
	Title        string              `json:"title"`
	Description  string              `json:"description"`
	System       systemDTO           `json:"system"`
	State        stateDTO            `json:"state"`
	Builds       []buildRefDTO       `json:"builds"`
	Requirements []requirementRefDTO `json:"requirements"`
	Missing      []missingDTO        `json:"missing"`
	CreatedBy    userRefDTO          `json:"createdBy"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
}

type createDTO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	System      string `json:"system"`
}

type patchDTO struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func (p patchDTO) toEdit() domain.Edit {
	return domain.Edit{Title: p.Title, Description: p.Description}
}

type transitionDTO struct {
	To     string `json:"to"`
	Reason string `json:"reason"`
}

func toDTO(out usecase.PackageOutput) packageDTO {
	s := out.Package
	dto := packageDTO{
		Code: s.Code.String(), Title: s.Title, Description: s.Description,
		System:       systemDTO{Key: s.System.Key, Name: s.System.Name, DefaultRoute: s.System.DefaultRoute},
		State:        stateDTO{Key: s.State.Key, Name: s.State.Name},
		Builds:       make([]buildRefDTO, len(s.Builds)),
		Requirements: make([]requirementRefDTO, len(s.Requirements)),
		Missing:      make([]missingDTO, len(out.Missing)),
		CreatedBy:    userRefDTO{ID: string(s.CreatedBy.Id), DisplayName: s.CreatedBy.Name},
		CreatedAt:    s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	for i, b := range s.Builds {
		dto.Builds[i] = buildRefDTO{Code: b.Code, Version: b.Version, Checksum: b.Checksum}
	}
	for i, r := range s.Requirements {
		dto.Requirements[i] = requirementRefDTO{Code: r.Code, Title: r.Title, State: stateDTO{Key: r.State.Key, Name: r.State.Name}}
	}
	for i, m := range out.Missing {
		missing := missingDTO{To: stateDTO{Key: m.To.Key, Name: m.To.Name}, Conditions: make([]conditionDTO, len(m.Conditions))}
		for j, c := range m.Conditions {
			related := c.Related
			if related == nil {
				related = []string{}
			}
			missing.Conditions[j] = conditionDTO{RuleKey: c.RuleKey, Message: c.Message, Blocks: c.Blocks, Related: related}
		}
		dto.Missing[i] = missing
	}
	return dto
}
