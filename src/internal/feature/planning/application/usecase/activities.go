package usecase

import (
	"context"
	"slices"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionPlan es el permiso de planificar actividades, que tienen el coordinador de QA y change management.
const PermissionPlan = "activity.manage"

// ErrActivityNotFound es el error de una actividad que no existe.
var ErrActivityNotFound = axis.New(axis.NotFoundResource, "La actividad no existe.")

// ActivityOutput es una actividad con su estado y sus solapes a la hora de leerla.
type ActivityOutput struct {
	Activity domain.ActivitySnapshot
	Status   domain.Status
	Overlaps []domain.Overlap
}

// SummaryOutput es una actividad con su estado, sin solapes.
type SummaryOutput struct {
	Activity domain.ActivitySnapshot
	Status   domain.Status
}

func toSummary(a domain.Activity, now time.Time) SummaryOutput {
	return SummaryOutput{Activity: a.Snapshot(), Status: a.Status(now)}
}

// withOverlaps lee las actividades de sus operadores en sus fechas y calcula los solapes.
func withOverlaps(ctx context.Context, repo port.Repository, a domain.Activity, now time.Time) (ActivityOutput, error) {
	s := a.Snapshot()
	out := ActivityOutput{Activity: s, Status: a.Status(now), Overlaps: []domain.Overlap{}}
	if a.Cancelled() || len(s.Operators) == 0 {
		return out, nil
	}
	ids := make([]shared.UserId, len(s.Operators))
	for i, op := range s.Operators {
		ids[i] = op.Id
	}
	others, err := repo.ActivitiesOfOperators(ctx, ids, s.Start, s.End)
	if err != nil {
		return ActivityOutput{}, err
	}
	out.Overlaps = domain.OverlapsOf(a, others)
	return out, nil
}

// findChange lee el CHG; vacío es sin CHG y uno que no existe es un error del campo change.
func findChange(ctx context.Context, repo port.Repository, code string) (*domain.ChangeRef, error) {
	if code == "" {
		return nil, nil
	}
	change, err := repo.FindChange(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return nil, axis.Invalid(axis.InvalidField("change", "El CHG no existe."))
	}
	if err != nil {
		return nil, err
	}
	return &change, nil
}

// findOperators lee los operadores; si alguno no existe es un error del campo operators.
func findOperators(ctx context.Context, repo port.Repository, raw []string) ([]domain.UserRef, error) {
	if len(raw) == 0 {
		return []domain.UserRef{}, nil
	}
	ids := make([]shared.UserId, 0, len(raw))
	for _, id := range raw {
		if !slices.Contains(ids, shared.UserId(id)) {
			ids = append(ids, shared.UserId(id))
		}
	}
	users, err := repo.FindUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(users) != len(ids) {
		return nil, axis.Invalid(axis.InvalidField("operators",
			"Algún operador no existe. Solo se asigna a quien ya entró una vez a Kaoru."))
	}
	return users, nil
}

func lockActivity(ctx context.Context, repo port.Repository, code string) (domain.Activity, error) {
	err := repo.LockActivity(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Activity{}, ErrActivityNotFound
	}
	if err != nil {
		return domain.Activity{}, err
	}
	return repo.FindActivity(ctx, code)
}
