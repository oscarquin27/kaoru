package domain

import (
	"cmp"
	"slices"
	"time"
)

// StateReleased es el estado del CHG que ya salió: su ventana terminada no es un atraso.
// Cerrado y rechazado ni siquiera llegan al calendario.
const StateReleased = "released"

// Window es la ventana de cambio de un CHG vigente.
type Window struct {
	ChangeCode  string
	ChangeTitle string
	System      string
	Route       string
	StateKey    string
	StateName   string
	Start       time.Time
	End         time.Time
}

// DaysLate son los días desde que terminó la ventana sin que el CHG esté liberado; cero si no hay atraso.
// Una ventana que terminó hace una hora ya cuenta un día.
func (w Window) DaysLate(now time.Time) int {
	if w.StateKey == StateReleased || !now.After(w.End) {
		return 0
	}
	return int((now.Sub(w.End) + 24*time.Hour - 1) / (24 * time.Hour))
}

// AlertKind es de qué es una alerta.
type AlertKind string

// Clases de alerta.
const (
	AlertMilestone    AlertKind = "milestone"
	AlertChangeWindow AlertKind = "change_window"
)

// Alert es algo atrasado. ChangeCode o MilestoneId, según la clase.
type Alert struct {
	Kind        AlertKind
	Title       string
	ChangeCode  string
	MilestoneId string
	DueAt       time.Time
	DaysLate    int
}

// Alerts junta lo atrasado de las ventanas y de los hitos pendientes, lo más atrasado primero.
func Alerts(windows []Window, milestones []Milestone, now time.Time) []Alert {
	alerts := []Alert{}
	for _, w := range windows {
		if late := w.DaysLate(now); late > 0 {
			alerts = append(alerts, Alert{
				Kind: AlertChangeWindow, Title: w.ChangeTitle, ChangeCode: w.ChangeCode, DueAt: w.End, DaysLate: late,
			})
		}
	}
	for _, m := range milestones {
		if late := m.DaysLate(now); late > 0 {
			s := m.Snapshot()
			alerts = append(alerts, Alert{Kind: AlertMilestone, Title: s.Title, MilestoneId: s.Id, DueAt: s.DueDate, DaysLate: late})
		}
	}
	slices.SortStableFunc(alerts, func(a, b Alert) int {
		return cmp.Or(cmp.Compare(b.DaysLate, a.DaysLate), a.DueAt.Compare(b.DueAt), cmp.Compare(a.Title, b.Title))
	})
	return alerts
}
