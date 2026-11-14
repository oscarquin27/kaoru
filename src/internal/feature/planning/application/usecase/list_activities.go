package usecase

import (
	"context"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
)

// ListActivities devuelve una página de actividades por fecha de inicio.
type ListActivities struct {
	d Deps
}

// ListActivitiesInput es la entrada de ListActivities. Vacío no filtra; From y To son días incluidos.
type ListActivitiesInput struct {
	Operator   string
	ChangeCode string
	Kind       string
	Status     string
	From       *time.Time
	To         *time.Time
	Page       int
	Size       int
}

// ListActivitiesOutput es una página de actividades.
type ListActivitiesOutput struct {
	Activities []SummaryOutput
	Total      int
}

// Execute lista las actividades. Leer no exige permiso.
func (uc *ListActivities) Execute(ctx context.Context, in ListActivitiesInput) (ListActivitiesOutput, error) {
	var errs axis.FieldErrors
	if in.Kind != "" && !domain.ValidKind(domain.ActivityKind(in.Kind)) {
		errs.Add(axis.InvalidField("kind", "La clase debe ser certification, regression, production u other."))
	}
	if in.Status != "" && !domain.ValidStatus(domain.Status(in.Status)) {
		errs.Add(axis.InvalidField("status", "El estado debe ser planned, active, done o cancelled."))
	}
	if in.From != nil && in.To != nil && in.To.Before(*in.From) {
		errs.Add(axis.InvalidField("to", "La fecha final no puede ser anterior a la inicial."))
	}
	if err := errs.Err(); err != nil {
		return ListActivitiesOutput{}, err
	}
	now := uc.d.Clock.Now()
	activities, total, err := uc.d.Repository.ListActivities(ctx, port.ActivityFilter{
		Operator: in.Operator, ChangeCode: in.ChangeCode, Kind: in.Kind, Status: in.Status, From: in.From, To: in.To,
		Today: domain.Day(now), Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListActivitiesOutput{}, err
	}
	out := ListActivitiesOutput{Activities: make([]SummaryOutput, len(activities)), Total: total}
	for i, a := range activities {
		out.Activities[i] = toSummary(a, now)
	}
	return out, nil
}
