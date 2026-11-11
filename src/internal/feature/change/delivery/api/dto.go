package api

import (
	"encoding/json"
	"time"

	"kaoru/internal/feature/change/application/usecase"
	"kaoru/internal/feature/change/domain"
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

type changeDTO struct {
	Code        string     `json:"code"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	System      systemDTO  `json:"system"`
	Route       string     `json:"route"`
	State       stateDTO   `json:"state"`
	WindowStart *time.Time `json:"windowStart,omitempty"`
	WindowEnd   *time.Time `json:"windowEnd,omitempty"`
	CreatedBy   userRefDTO `json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type pageDTO struct {
	Items []changeDTO `json:"items"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Total int         `json:"total"`
}

type createDTO struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	System      string     `json:"system"`
	Route       string     `json:"route"`
	WindowStart *time.Time `json:"windowStart"`
	WindowEnd   *time.Time `json:"windowEnd"`
}

// nullableTime distingue «no vino» de «vino null» en un PATCH.
type nullableTime struct {
	set   bool
	value *time.Time
}

func (n *nullableTime) UnmarshalJSON(data []byte) error {
	n.set = true
	if string(data) == "null" {
		return nil
	}
	var t time.Time
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	n.value = &t
	return nil
}

type patchDTO struct {
	Title       *string      `json:"title"`
	Description *string      `json:"description"`
	Route       *string      `json:"route"`
	WindowStart nullableTime `json:"windowStart"`
	WindowEnd   nullableTime `json:"windowEnd"`
}

func (p patchDTO) toEdit() domain.Edit {
	return domain.Edit{
		Title: p.Title, Description: p.Description, Route: p.Route,
		WindowStart: domain.TimeEdit{Set: p.WindowStart.set, Value: p.WindowStart.value},
		WindowEnd:   domain.TimeEdit{Set: p.WindowEnd.set, Value: p.WindowEnd.value},
	}
}

type transitionDTO struct {
	To     string `json:"to"`
	Reason string `json:"reason"`
}

func toDTO(v usecase.ChangeOutput) changeDTO {
	s := v.Change
	return changeDTO{
		Code: s.Code.String(), Title: s.Title, Description: s.Description,
		System: systemDTO{Key: s.System.Key, Name: s.System.Name, DefaultRoute: string(s.System.DefaultRoute)},
		Route:  string(s.Route), State: stateDTO{Key: s.State.Key, Name: s.State.Name},
		WindowStart: s.WindowStart, WindowEnd: s.WindowEnd,
		CreatedBy: userRefDTO{ID: string(s.CreatedBy), DisplayName: v.CreatedByName},
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}
