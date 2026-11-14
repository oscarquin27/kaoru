package usecase

import (
	"context"
	"time"

	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// GetSchedule devuelve la planificación de un rango de días: quién hace qué, qué se solapa y qué
// proyectos coinciden, con las ventanas de cambio y los hitos.
type GetSchedule struct {
	d Deps
}

// GetScheduleInput es la entrada de GetSchedule. Sin fechas, seis semanas desde el lunes de esta semana.
type GetScheduleInput struct {
	From     *time.Time
	To       *time.Time
	Operator string
}

// ScheduleOutput es la planificación con el «ahora» con que se calcularon los estados.
type ScheduleOutput struct {
	Schedule domain.Schedule
	Now      time.Time
}

// Execute arma la planificación. Leer no exige permiso.
func (uc *GetSchedule) Execute(ctx context.Context, in GetScheduleInput) (ScheduleOutput, error) {
	now := uc.d.Clock.Now()
	r, err := domain.ScheduleRange(in.From, in.To, now)
	if err != nil {
		return ScheduleOutput{}, err
	}
	activities, err := uc.d.Repository.ActivitiesBetween(ctx, r.From, r.To)
	if err != nil {
		return ScheduleOutput{}, err
	}
	windows, err := uc.d.Changes.Windows(ctx)
	if err != nil {
		return ScheduleOutput{}, err
	}
	milestones, err := uc.d.Repository.MilestonesBetween(ctx, r.From, r.To)
	if err != nil {
		return ScheduleOutput{}, err
	}
	return ScheduleOutput{
		Schedule: domain.BuildSchedule(r, activities, windows, milestones, shared.UserId(in.Operator)),
		Now:      now,
	}, nil
}
