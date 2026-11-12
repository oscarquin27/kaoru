package api

import (
	"time"

	"kaoru/internal/feature/component/application/usecase"
	"kaoru/internal/feature/component/domain"
)

type systemDTO struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	DefaultRoute string `json:"defaultRoute"`
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type componentDTO struct {
	Code        string     `json:"code"`
	System      systemDTO  `json:"system"`
	Kind        string     `json:"kind"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Retired     bool       `json:"retired"`
	CreatedBy   userRefDTO `json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type pageDTO struct {
	Items []componentDTO `json:"items"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
	Total int            `json:"total"`
}

type createDTO struct {
	System      string `json:"system"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type patchDTO struct {
	Kind        *string `json:"kind"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Retired     *bool   `json:"retired"`
}

func (p patchDTO) toEdit() domain.Edit {
	return domain.Edit{Kind: p.Kind, Name: p.Name, Description: p.Description, Retired: p.Retired}
}

func toDTO(v usecase.ComponentOutput) componentDTO {
	s := v.Component
	return componentDTO{
		Code:   s.Code.String(),
		System: systemDTO{Key: s.System.Key, Name: s.System.Name, DefaultRoute: s.System.DefaultRoute},
		Kind:   string(s.Kind), Name: s.Name, Description: s.Description, Retired: s.RetiredAt != nil,
		CreatedBy: userRefDTO{ID: string(s.CreatedBy), DisplayName: v.CreatedByName},
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}
