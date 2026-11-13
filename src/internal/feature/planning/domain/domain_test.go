package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/planning/domain"
)

var now = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)

func fields(err error) []string {
	var ge *axis.GeneralError
	if !errors.As(err, &ge) {
		return nil
	}
	var out []string
	for _, f := range ge.Fields() {
		out = append(out, f.Field+":"+f.Code)
	}
	return out
}

func milestone(t *testing.T, due string) domain.Milestone {
	t.Helper()
	m, err := domain.New(domain.NewParams{Id: "m1", Title: " Congelamiento ", DueDate: due, CreatedBy: "u1", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNewValidates(t *testing.T) {
	cases := map[string]struct {
		title, due string
		want       []string
	}{
		"vacío":        {"", "", []string{"title:required", "dueDate:required"}},
		"fecha rara":   {"Hito", "30/11/2026", []string{"dueDate:invalid"}},
		"título largo": {strings.Repeat("a", 201), "2026-11-30", []string{"title:invalid"}},
	}
	for name, c := range cases {
		_, err := domain.New(domain.NewParams{Title: c.title, DueDate: c.due, Now: now})
		if !slices.Equal(fields(err), c.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
	s := milestone(t, "2026-11-30").Snapshot()
	if s.Title != "Congelamiento" || s.DueDate != time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC) || s.CompletedAt != nil {
		t.Errorf("hito = %+v", s)
	}
}

func TestApplyReportsOnlyWhatChanged(t *testing.T) {
	m := milestone(t, "2026-11-30")
	title, due, yes, no := "Congelamiento", "2026-12-01", true, false
	later := now.Add(time.Hour)
	changes, err := m.Apply(domain.Edit{Title: &title, DueDate: &due, Completed: &yes,
		Owner: domain.OwnerEdit{Set: true, Value: &domain.UserRef{Id: "u7"}}}, later)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range changes {
		got = append(got, c.Field+":"+c.Old+">"+c.New)
	}
	if want := []string{"dueDate:2026-11-30>2026-12-01", "owner:>u7", "completed:false>true"}; !slices.Equal(got, want) {
		t.Errorf("cambios = %v", got)
	}
	if s := m.Snapshot(); s.CompletedAt == nil || !s.CompletedAt.Equal(later) || !s.UpdatedAt.Equal(later) {
		t.Errorf("hito = %+v", s)
	}
	// Marcarlo cumplido otra vez no mueve la fecha; volver a pendiente la borra.
	if changes, _ := m.Apply(domain.Edit{Completed: &yes}, later.Add(time.Hour)); len(changes) != 0 {
		t.Errorf("cambios = %v", changes)
	}
	if _, err := m.Apply(domain.Edit{Completed: &no}, later); err != nil || m.Snapshot().CompletedAt != nil {
		t.Errorf("no volvió a pendiente: %v", err)
	}
	empty := ""
	if _, err := m.Apply(domain.Edit{Title: &empty}, later); !slices.Equal(fields(err), []string{"title:required"}) {
		t.Errorf("título vacío: %v", err)
	}
}

func TestDaysLate(t *testing.T) {
	for due, want := range map[string]int{"2026-11-12": 0, "2026-11-30": 0, "2026-11-10": 2} {
		if got := milestone(t, due).DaysLate(now); got != want {
			t.Errorf("%s: %d días, se esperaban %d", due, got, want)
		}
	}
	m := milestone(t, "2026-11-01")
	yes := true
	_, _ = m.Apply(domain.Edit{Completed: &yes}, now)
	if m.DaysLate(now) != 0 {
		t.Error("un hito cumplido no está atrasado")
	}

	w := domain.Window{StateKey: "in_progress", End: now.Add(-time.Hour)}
	if w.DaysLate(now) != 1 {
		t.Errorf("ventana que terminó hace una hora: %d", w.DaysLate(now))
	}
	w.End = now.Add(-49 * time.Hour)
	if w.DaysLate(now) != 3 {
		t.Errorf("ventana que terminó hace 49 horas: %d", w.DaysLate(now))
	}
	w.StateKey = domain.StateReleased
	if w.DaysLate(now) != 0 {
		t.Error("un CHG liberado no está atrasado")
	}
	if (domain.Window{End: now.Add(time.Hour)}).DaysLate(now) != 0 {
		t.Error("una ventana abierta no está atrasada")
	}
}

func TestAlertsPutTheLatestFirst(t *testing.T) {
	windows := []domain.Window{
		{ChangeCode: "CHG-2026-0001", ChangeTitle: "Mensajería", StateKey: "in_progress", End: now.Add(-time.Hour)},
		{ChangeCode: "CHG-2026-0002", ChangeTitle: "Liberado", StateKey: domain.StateReleased, End: now.Add(-72 * time.Hour)},
	}
	alerts := domain.Alerts(windows, []domain.Milestone{milestone(t, "2026-11-09"), milestone(t, "2026-12-01")}, now)
	if len(alerts) != 2 || alerts[0].Kind != domain.AlertMilestone || alerts[0].DaysLate != 3 ||
		alerts[1].ChangeCode != "CHG-2026-0001" || alerts[1].DaysLate != 1 {
		t.Errorf("alertas = %+v", alerts)
	}
	if got := domain.Alerts(nil, nil, now); got == nil || len(got) != 0 {
		t.Errorf("sin atrasos debe dar una lista vacía: %v", got)
	}
}
