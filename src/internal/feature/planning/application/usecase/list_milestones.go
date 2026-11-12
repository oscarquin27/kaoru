package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/planning/application/port"
)

// ListMilestones devuelve una página de hitos, por fecha.
type ListMilestones struct {
	d Deps
}

// ListMilestonesInput es la entrada de ListMilestones. Status es pending, completed o vacío para todos.
type ListMilestonesInput struct {
	Status string
	Page   int
	Size   int
}

// ListMilestonesOutput es una página de hitos.
type ListMilestonesOutput struct {
	Milestones []MilestoneOutput
	Total      int
}

// Execute lista los hitos. Leer no exige permiso.
func (uc *ListMilestones) Execute(ctx context.Context, in ListMilestonesInput) (ListMilestonesOutput, error) {
	if in.Status != "" && in.Status != port.StatusPending && in.Status != port.StatusCompleted {
		return ListMilestonesOutput{}, axis.Invalid(axis.InvalidField("status", "El estado debe ser pending o completed."))
	}
	milestones, total, err := uc.d.Repository.List(ctx, port.Filter{Status: in.Status, Page: in.Page, Size: in.Size})
	if err != nil {
		return ListMilestonesOutput{}, err
	}
	now := uc.d.Clock.Now()
	out := ListMilestonesOutput{Milestones: make([]MilestoneOutput, len(milestones)), Total: total}
	for i, m := range milestones {
		out.Milestones[i] = toOutput(m, now)
	}
	return out, nil
}
