package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/application/usecase"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	coordinator = shared.Actor{Id: "u1", Roles: []string{"qa-admin"}}
	changes     = map[string]domain.ChangeRef{
		"CHG-2026-0001": {Id: "00000000-0000-0000-0000-0000000000c1", Code: "CHG-2026-0001", Title: "Límite diario", System: "SyPago"},
		"CHG-2026-0002": {Id: "00000000-0000-0000-0000-0000000000c2", Code: "CHG-2026-0002", Title: "Mensajería", System: "SIMF"},
	}
)

func (w *world) NextActivityCode(_ context.Context, year int) (shared.Code, error) {
	w.actSeq++
	return shared.NewCode(shared.PrefixActivity, year, w.actSeq), nil
}

func (w *world) InsertActivity(_ context.Context, a domain.Activity) error {
	if !w.inTx {
		return errors.New("InsertActivity fuera de una transacción")
	}
	w.activities[a.Snapshot().Code] = a
	return nil
}

func (w *world) UpdateActivity(_ context.Context, a domain.Activity) error {
	if !w.inTx || !slices.Contains(w.locked, a.Snapshot().Code) {
		return errors.New("UpdateActivity sin bloquear la actividad en una transacción")
	}
	w.activities[a.Snapshot().Code] = a
	return nil
}

func (w *world) LockActivity(_ context.Context, code string) error {
	if !w.inTx {
		return errors.New("LockActivity fuera de una transacción")
	}
	if _, ok := w.activities[code]; !ok {
		return notFound
	}
	w.locked = append(w.locked, code)
	return nil
}

func (w *world) FindActivity(_ context.Context, code string) (domain.Activity, error) {
	a, ok := w.activities[code]
	if !ok {
		return domain.Activity{}, notFound
	}
	s := a.Snapshot()
	s.CreatedBy.Name = users[s.CreatedBy.Id]
	return domain.RehydrateActivity(s), nil
}

func (w *world) all(keep func(domain.Activity) bool) []domain.Activity {
	var out []domain.Activity
	for _, a := range w.activities {
		if keep(a) {
			out = append(out, a)
		}
	}
	domain.SortActivities(out)
	return out
}

func (w *world) ListActivities(_ context.Context, f port.ActivityFilter) ([]domain.Activity, int, error) {
	out := w.all(func(a domain.Activity) bool {
		s := a.Snapshot()
		return (f.Operator == "" || a.HasOperator(shared.UserId(f.Operator))) &&
			(f.ChangeCode == "" || s.Change != nil && s.Change.Code == f.ChangeCode) &&
			(f.Kind == "" || string(s.Kind) == f.Kind) &&
			(f.Status == "" || string(a.Status(f.Today)) == f.Status) &&
			(f.From == nil || !s.End.Before(*f.From)) && (f.To == nil || !s.Start.After(*f.To))
	})
	return out, len(out), nil
}

func (w *world) ActivitiesBetween(_ context.Context, from, to time.Time) ([]domain.Activity, error) {
	r := domain.Range{From: from, To: to}
	return w.all(func(a domain.Activity) bool {
		return !a.Cancelled() && r.Intersects(a.Snapshot().Start, a.Snapshot().End)
	}), nil
}

func (w *world) ActivitiesOfOperators(ctx context.Context, ids []shared.UserId, from, to time.Time) ([]domain.Activity, error) {
	between, _ := w.ActivitiesBetween(ctx, from, to)
	var out []domain.Activity
	for _, a := range between {
		if slices.ContainsFunc(ids, a.HasOperator) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (w *world) Certifications(_ context.Context, code string) ([]domain.Activity, error) {
	return w.all(func(a domain.Activity) bool {
		s := a.Snapshot()
		return s.Kind == domain.KindCertification && s.Change != nil && s.Change.Code == code
	}), nil
}

func (w *world) FindChange(_ context.Context, code string) (domain.ChangeRef, error) {
	c, ok := changes[code]
	if !ok {
		return domain.ChangeRef{}, notFound
	}
	return c, nil
}

func (w *world) FindUsers(_ context.Context, ids []shared.UserId) ([]domain.UserRef, error) {
	var out []domain.UserRef
	for _, id := range ids {
		if name, ok := users[id]; ok {
			out = append(out, domain.UserRef{Id: id, Name: name})
		}
	}
	return out, nil
}

func (w *world) MilestonesBetween(_ context.Context, from, to time.Time) ([]domain.Milestone, error) {
	r := domain.Range{From: from, To: to}
	return w.sorted(func(s domain.Snapshot) bool { return r.Intersects(s.DueDate, s.DueDate) }), nil
}

func plan(t *testing.T, uc *usecase.UseCases, kind, change, start, end string, ops ...string) usecase.ActivityOutput {
	t.Helper()
	out, err := uc.CreateActivity.Execute(ctx, usecase.CreateActivityInput{
		Actor: coordinator, Kind: kind, Title: kind + " " + change, Change: change, Start: start, End: end, Operators: ops,
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func overlapText(list []domain.Overlap) []string {
	var out []string
	for _, o := range list {
		out = append(out, fmt.Sprintf("%s %s/%s %d", o.Operator.Name, o.First.Code, o.Second.Code, o.Days()))
	}
	return out
}

func TestCreateActivity(t *testing.T) {
	uc, w, _ := setup(t)
	in := usecase.CreateActivityInput{Actor: qa, Kind: "certification", Title: "Certificación", Change: "CHG-2026-0001", Start: "2026-11-16", End: "2026-11-27"}
	if _, err := uc.CreateActivity.Execute(ctx, in); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("QA no planifica: %v", err)
	}
	in.Actor, in.Change = coordinator, "CHG-2026-0099"
	if _, err := uc.CreateActivity.Execute(ctx, in); !slices.Equal(fieldsOf(err), []string{"change:invalid"}) {
		t.Errorf("CHG inexistente: %v", err)
	}
	in.Change, in.Operators = "CHG-2026-0001", []string{"u8", "nadie"}
	if _, err := uc.CreateActivity.Execute(ctx, in); !slices.Equal(fieldsOf(err), []string{"operators:invalid"}) {
		t.Errorf("operador inexistente: %v", err)
	}
	if w.actSeq != 0 || len(w.audit) != 0 {
		t.Errorf("un intento fallido no reserva código ni audita: %d %v", w.actSeq, w.audit)
	}

	in.Operators = []string{"u9", "u8", "u8"}
	first, err := uc.CreateActivity.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	s := first.Activity
	if s.Code != "ACT-2026-0001" || s.Change.Code != "CHG-2026-0001" || len(s.Operators) != 2 || s.Operators[0].Name != "Ana Rojas" ||
		first.Status != domain.StatusPlanned || len(first.Overlaps) != 0 || s.CreatedBy.Name != "Carla Cambios" {
		t.Errorf("actividad = %+v", first)
	}
	if len(w.audit) != 1 || w.audit[0].Entity != port.EntityActivity || w.audit[0].Code != "ACT-2026-0001" || w.audit[0].NewValue != "2026-11-16 a 2026-11-27" {
		t.Errorf("auditoría = %+v", w.audit)
	}

	// Una regresión de Ana que pisa los tres últimos días: se guarda y avisa.
	second := plan(t, uc, "regression", "", "2026-11-25", "2026-12-02", "u8")
	if got := overlapText(second.Overlaps); !slices.Equal(got, []string{"Ana Rojas ACT-2026-0002/ACT-2026-0001 3"}) {
		t.Errorf("solapes = %v", got)
	}
}

func TestUpdateActivity(t *testing.T) {
	uc, w, _ := setup(t)
	cert := plan(t, uc, "certification", "CHG-2026-0001", "2026-11-16", "2026-11-27", "u8")
	regr := plan(t, uc, "regression", "", "2026-12-01", "2026-12-04", "u8")
	if len(regr.Overlaps) != 0 {
		t.Fatalf("no debían cruzarse: %v", regr.Overlaps)
	}
	update := func(actor shared.Actor, in usecase.UpdateActivityInput) (usecase.ActivityOutput, error) {
		in.Actor = actor
		if in.Code == "" {
			in.Code = regr.Activity.Code
		}
		return uc.UpdateActivity.Execute(ctx, in)
	}
	if _, err := update(qa, usecase.UpdateActivityInput{Start: ptr("2026-11-20")}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("QA no planifica: %v", err)
	}
	if _, err := update(coordinator, usecase.UpdateActivityInput{Code: "ACT-2026-0099"}); !errors.Is(err, usecase.ErrActivityNotFound) {
		t.Errorf("actividad inexistente: %v", err)
	}

	// Reajustar el inicio hace que se cruce con la certificación.
	out, err := update(coordinator, usecase.UpdateActivityInput{Start: ptr("2026-11-26"), Operators: &[]string{"u8", "u9"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := overlapText(out.Overlaps); !slices.Equal(got, []string{"Ana Rojas ACT-2026-0002/ACT-2026-0001 2"}) {
		t.Errorf("solapes = %v", got)
	}
	// Cancelar quita el solape, también visto desde la otra actividad.
	if out, err = update(coordinator, usecase.UpdateActivityInput{Cancelled: ptr(true)}); err != nil || out.Status != domain.StatusCancelled || len(out.Overlaps) != 0 {
		t.Fatalf("cancelada = %+v %v", out, err)
	}
	if got, _ := uc.GetActivity.Execute(ctx, cert.Activity.Code); len(got.Overlaps) != 0 {
		t.Errorf("la cancelada no se solapa: %v", got.Overlaps)
	}
	var fields []string
	for _, r := range w.audit[2:] {
		fields = append(fields, r.Field+":"+r.OldValue+">"+r.NewValue)
	}
	if want := []string{"startDate:2026-12-01>2026-11-26", "operators:u8>u8,u9", "cancelled:false>true"}; !slices.Equal(fields, want) {
		t.Errorf("auditoría = %v", fields)
	}

	// Si la auditoría falla no se guarda nada.
	w.auditErr = errors.New("auditoría caída")
	if _, err := update(coordinator, usecase.UpdateActivityInput{Title: ptr("Otra")}); err == nil {
		t.Fatal("debía fallar")
	}
	if w.activities[regr.Activity.Code].Snapshot().Title == "Otra" {
		t.Error("se guardó un cambio sin auditar")
	}
}

func TestListActivitiesAndSchedule(t *testing.T) {
	uc, w, _ := setup(t)
	plan(t, uc, "certification", "CHG-2026-0001", "2026-11-16", "2026-11-27", "u8", "u9")
	plan(t, uc, "certification", "CHG-2026-0002", "2026-11-23", "2026-12-04", "u8")
	plan(t, uc, "other", "", "2026-11-10", "2026-11-13")
	create(t, uc, "Congelamiento", "2026-11-20")
	w.windows = []domain.Window{{ChangeCode: "CHG-2026-0001", Start: time.Date(2026, 11, 28, 22, 0, 0, 0, time.UTC), End: time.Date(2026, 11, 29, 2, 0, 0, 0, time.UTC)}}

	codes := func(in usecase.ListActivitiesInput) []string {
		in.Page, in.Size = 1, 20
		out, err := uc.ListActivities.Execute(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, a := range out.Activities {
			got = append(got, a.Activity.Code+":"+string(a.Status))
		}
		return got
	}
	if got := codes(usecase.ListActivitiesInput{}); !slices.Equal(got, []string{"ACT-2026-0003:active", "ACT-2026-0001:planned", "ACT-2026-0002:planned"}) {
		t.Errorf("todas = %v", got)
	}
	if got := codes(usecase.ListActivitiesInput{Operator: "u9"}); !slices.Equal(got, []string{"ACT-2026-0001:planned"}) {
		t.Errorf("de Luis = %v", got)
	}
	if got := codes(usecase.ListActivitiesInput{Status: "active"}); !slices.Equal(got, []string{"ACT-2026-0003:active"}) {
		t.Errorf("en curso = %v", got)
	}
	from, to := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	if _, err := uc.ListActivities.Execute(ctx, usecase.ListActivitiesInput{Kind: "x", Status: "y", From: &from, To: &to}); !slices.Equal(fieldsOf(err), []string{"kind:invalid", "status:invalid", "to:invalid"}) {
		t.Errorf("filtros inválidos: %v", err)
	}

	out, err := uc.GetSchedule.Execute(ctx, usecase.GetScheduleInput{})
	if err != nil {
		t.Fatal(err)
	}
	s := out.Schedule
	if s.Range.From.Format(domain.DateLayout) != "2026-11-09" || len(s.Operators) != 2 || s.OverlapCount() != 1 ||
		len(s.Unassigned) != 1 || len(s.Projects) != 2 || len(s.Windows) != 1 || len(s.Milestones) != 1 {
		t.Errorf("planificación = %+v", s)
	}
	if _, err := uc.GetSchedule.Execute(ctx, usecase.GetScheduleInput{From: &from, To: &to}); !slices.Equal(fieldsOf(err), []string{"to:invalid"}) {
		t.Errorf("rango al revés: %v", err)
	}
}

func TestCheckCertification(t *testing.T) {
	uc, _, clock := setup(t)
	if got, _ := uc.CheckCertification.Execute(ctx, "CHG-2026-0001"); got.State != domain.CertificationNone {
		t.Errorf("sin plan: %v", got.State)
	}
	plan(t, uc, "certification", "CHG-2026-0001", "2026-11-16", "2026-11-27")
	if got, _ := uc.CheckCertification.Execute(ctx, "CHG-2026-0001"); got.State != domain.CertificationNoOperators {
		t.Errorf("sin operadores: %v", got.State)
	}
	plan(t, uc, "certification", "CHG-2026-0001", "2026-11-16", "2026-11-27", "u8")
	if got, _ := uc.CheckCertification.Execute(ctx, "CHG-2026-0001"); got.State != domain.CertificationOutside {
		t.Errorf("todavía no empieza: %v", got.State)
	}
	clock.now = time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	if got, _ := uc.CheckCertification.Execute(ctx, "CHG-2026-0001"); got.State != domain.CertificationReady {
		t.Errorf("el primer día: %v", got.State)
	}
}
