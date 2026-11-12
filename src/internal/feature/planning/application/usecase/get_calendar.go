package usecase

import (
	"context"

	"kaoru/internal/feature/planning/domain"
)

// GetCalendar devuelve las ventanas de cambio vigentes y las alertas de atraso, calculadas al leer.
type GetCalendar struct {
	d Deps
}

// WindowOutput es una ventana con sus días de atraso.
type WindowOutput struct {
	Window   domain.Window
	DaysLate int
}

// CalendarOutput es el calendario.
type CalendarOutput struct {
	Windows []WindowOutput
	Alerts  []domain.Alert
}

// Execute lee el calendario. Leer no exige permiso.
func (uc *GetCalendar) Execute(ctx context.Context) (CalendarOutput, error) {
	windows, err := uc.d.Changes.Windows(ctx)
	if err != nil {
		return CalendarOutput{}, err
	}
	pending, err := uc.d.Repository.Pending(ctx)
	if err != nil {
		return CalendarOutput{}, err
	}
	now := uc.d.Clock.Now()
	out := CalendarOutput{Windows: make([]WindowOutput, len(windows)), Alerts: domain.Alerts(windows, pending, now)}
	for i, w := range windows {
		out.Windows[i] = WindowOutput{Window: w, DaysLate: w.DaysLate(now)}
	}
	return out, nil
}
