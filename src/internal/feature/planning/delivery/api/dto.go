package api

import (
	"time"

	"kaoru/internal/feature/planning/application/usecase"
	"kaoru/internal/feature/planning/domain"
)

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type milestoneDTO struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	DueDate     string      `json:"dueDate"`
	Notes       string      `json:"notes"`
	Owner       *userRefDTO `json:"owner,omitempty"`
	Completed   bool        `json:"completed"`
	CompletedAt *time.Time  `json:"completedAt,omitempty"`
	DaysLate    int         `json:"daysLate"`
	CreatedBy   userRefDTO  `json:"createdBy"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

type milestoneInputDTO struct {
	Title   string `json:"title"`
	DueDate string `json:"dueDate"`
	Owner   string `json:"owner"`
	Notes   string `json:"notes"`
}

type milestonePatchDTO struct {
	Title     *string `json:"title"`
	DueDate   *string `json:"dueDate"`
	Owner     *string `json:"owner"`
	Notes     *string `json:"notes"`
	Completed *bool   `json:"completed"`
}

type codeTitleDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type stateRefDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type windowDTO struct {
	Change   codeTitleDTO `json:"change"`
	System   string       `json:"system"`
	Route    string       `json:"route"`
	State    stateRefDTO  `json:"state"`
	Start    time.Time    `json:"start"`
	End      time.Time    `json:"end"`
	DaysLate int          `json:"daysLate"`
}

type alertDTO struct {
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Change    string    `json:"change,omitempty"`
	Milestone string    `json:"milestone,omitempty"`
	DueAt     time.Time `json:"dueAt"`
	DaysLate  int       `json:"daysLate"`
}

type calendarDTO struct {
	Windows []windowDTO `json:"windows"`
	Alerts  []alertDTO  `json:"alerts"`
}

func toMilestoneDTO(o usecase.MilestoneOutput) milestoneDTO {
	s := o.Milestone
	dto := milestoneDTO{
		ID: s.Id, Title: s.Title, DueDate: s.DueDate.Format(domain.DateLayout), Notes: s.Notes,
		Completed: s.CompletedAt != nil, CompletedAt: s.CompletedAt, DaysLate: o.DaysLate,
		CreatedBy: userRefDTO{ID: string(s.CreatedBy.Id), DisplayName: s.CreatedBy.Name}, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if s.Owner != nil {
		dto.Owner = &userRefDTO{ID: string(s.Owner.Id), DisplayName: s.Owner.Name}
	}
	return dto
}

func toCalendarDTO(o usecase.CalendarOutput) calendarDTO {
	dto := calendarDTO{Windows: make([]windowDTO, len(o.Windows)), Alerts: make([]alertDTO, len(o.Alerts))}
	for i, w := range o.Windows {
		dto.Windows[i] = windowDTO{
			Change: codeTitleDTO{Code: w.Window.ChangeCode, Title: w.Window.ChangeTitle}, System: w.Window.System,
			Route: w.Window.Route, State: stateRefDTO{Key: w.Window.StateKey, Name: w.Window.StateName},
			Start: w.Window.Start, End: w.Window.End, DaysLate: w.DaysLate,
		}
	}
	for i, a := range o.Alerts {
		dto.Alerts[i] = alertDTO{
			Kind: string(a.Kind), Title: a.Title, Change: a.ChangeCode, Milestone: a.MilestoneId, DueAt: a.DueAt, DaysLate: a.DaysLate,
		}
	}
	return dto
}
