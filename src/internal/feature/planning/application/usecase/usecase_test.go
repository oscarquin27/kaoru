package usecase_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/application/usecase"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	milestones map[string]domain.Milestone
	windows    []domain.Window
	audit      []port.AuditRecord
	auditErr   error
	locked     []string
	inTx       bool
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	milestones, audit, locked := maps.Clone(w.milestones), slices.Clone(w.audit), slices.Clone(w.locked)
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.milestones, w.audit, w.locked = milestones, audit, locked
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

var users = map[shared.UserId]string{"u1": "Carla Cambios", "u7": "Omar Ops"}

func (w *world) Insert(_ context.Context, m domain.Milestone) error {
	if !w.inTx {
		return errors.New("Insert fuera de una transacción")
	}
	w.milestones[m.Snapshot().Id] = m
	return nil
}

func (w *world) Update(_ context.Context, m domain.Milestone) error {
	if !w.inTx || !slices.Contains(w.locked, m.Snapshot().Id) {
		return errors.New("Update sin bloquear el hito en una transacción")
	}
	w.milestones[m.Snapshot().Id] = m
	return nil
}

func (w *world) Lock(_ context.Context, id string) error {
	if !w.inTx {
		return errors.New("Lock fuera de una transacción")
	}
	if _, ok := w.milestones[id]; !ok {
		return notFound
	}
	w.locked = append(w.locked, id)
	return nil
}

// Find devuelve el hito con los nombres, como la base.
func (w *world) Find(_ context.Context, id string) (domain.Milestone, error) {
	m, ok := w.milestones[id]
	if !ok {
		return domain.Milestone{}, notFound
	}
	s := m.Snapshot()
	s.CreatedBy.Name = users[s.CreatedBy.Id]
	return domain.Rehydrate(s), nil
}

func (w *world) sorted(keep func(domain.Snapshot) bool) []domain.Milestone {
	var out []domain.Milestone
	for _, m := range w.milestones {
		if keep(m.Snapshot()) {
			out = append(out, m)
		}
	}
	slices.SortFunc(out, func(a, b domain.Milestone) int {
		return cmp.Or(a.Snapshot().DueDate.Compare(b.Snapshot().DueDate), cmp.Compare(a.Snapshot().Id, b.Snapshot().Id))
	})
	return out
}

func (w *world) List(_ context.Context, f port.Filter) ([]domain.Milestone, int, error) {
	out := w.sorted(func(s domain.Snapshot) bool {
		return f.Status == "" || (f.Status == port.StatusPending) == (s.CompletedAt == nil)
	})
	return out, len(out), nil
}

func (w *world) Pending(context.Context) ([]domain.Milestone, error) {
	return w.sorted(func(s domain.Snapshot) bool { return s.CompletedAt == nil }), nil
}

func (w *world) FindUser(_ context.Context, id shared.UserId) (domain.UserRef, error) {
	name, ok := users[id]
	if !ok {
		return domain.UserRef{}, notFound
	}
	return domain.UserRef{Id: id, Name: name}, nil
}

func (w *world) Windows(context.Context) ([]domain.Window, error) { return w.windows, nil }

var granted = map[string][]string{"change-manager": {usecase.PermissionManage}}

func (w *world) Require(_ context.Context, roles []string, permission string) error {
	for _, role := range roles {
		if slices.Contains(granted[role], permission) {
			return nil
		}
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

func (w *world) Record(_ context.Context, r port.AuditRecord) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	if !w.inTx {
		return errors.New("auditoría fuera de una transacción")
	}
	w.audit = append(w.audit, r)
	return nil
}

type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", s.n), nil
}

var (
	t0      = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	qa      = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	ctx     = context.Background()
)

func setup(t *testing.T) (*usecase.UseCases, *world, *fixedClock) {
	t.Helper()
	w := &world{milestones: map[string]domain.Milestone{}}
	clock := &fixedClock{now: t0}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Changes: w, Permissions: w, AuditTrail: w, UnitOfWork: w, Clock: clock, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w, clock
}

func fieldsOf(err error) []string {
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

func create(t *testing.T, uc *usecase.UseCases, title, due string) usecase.MilestoneOutput {
	t.Helper()
	out, err := uc.CreateMilestone.Execute(ctx, usecase.CreateMilestoneInput{Actor: manager, Title: title, DueDate: due})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func ptr[T any](v T) *T { return &v }

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestCreateMilestone(t *testing.T) {
	uc, w, _ := setup(t)
	in := usecase.CreateMilestoneInput{Actor: qa, Title: "Congelamiento", DueDate: "2026-11-30"}
	if _, err := uc.CreateMilestone.Execute(ctx, in); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("QA no mantiene hitos: %v", err)
	}
	in.Actor, in.Owner = manager, "nadie"
	if _, err := uc.CreateMilestone.Execute(ctx, in); !slices.Equal(fieldsOf(err), []string{"owner:invalid"}) {
		t.Errorf("responsable inexistente: %v", err)
	}
	in.Owner = "u7"
	out, err := uc.CreateMilestone.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	s := out.Milestone
	if s.Title != "Congelamiento" || s.Owner == nil || s.Owner.Name != "Omar Ops" || s.CreatedBy.Name != "Carla Cambios" || out.DaysLate != 0 {
		t.Errorf("hito = %+v", out)
	}
	if len(w.audit) != 1 || w.audit[0].Action != port.AuditCreate || w.audit[0].NewValue != "2026-11-30" {
		t.Errorf("auditoría = %+v", w.audit)
	}
}

func TestUpdateMilestone(t *testing.T) {
	uc, w, clock := setup(t)
	id := create(t, uc, "Congelamiento", "2026-11-10").Milestone.Id
	update := func(actor shared.Actor, in usecase.UpdateMilestoneInput) (usecase.MilestoneOutput, error) {
		in.Actor = actor
		if in.Id == "" {
			in.Id = id
		}
		return uc.UpdateMilestone.Execute(ctx, in)
	}

	if _, err := update(qa, usecase.UpdateMilestoneInput{Completed: ptr(true)}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("QA no mantiene hitos: %v", err)
	}
	if _, err := update(manager, usecase.UpdateMilestoneInput{Id: "otro"}); !errors.Is(err, usecase.ErrMilestoneNotFound) {
		t.Errorf("hito inexistente: %v", err)
	}
	if _, err := update(manager, usecase.UpdateMilestoneInput{Owner: ptr("nadie")}); !slices.Equal(fieldsOf(err), []string{"owner:invalid"}) {
		t.Errorf("responsable inexistente: %v", err)
	}

	out, err := update(manager, usecase.UpdateMilestoneInput{Owner: ptr("u7")})
	if err != nil || out.DaysLate != 2 || out.Milestone.Owner == nil {
		t.Fatalf("hito = %+v %v", out, err)
	}
	clock.now = t0.Add(time.Hour)
	out, err = update(manager, usecase.UpdateMilestoneInput{Completed: ptr(true), Owner: ptr("")})
	if err != nil || out.DaysLate != 0 || out.Milestone.CompletedAt == nil || out.Milestone.Owner != nil {
		t.Fatalf("hito cumplido = %+v %v", out, err)
	}
	var fields []string
	for _, r := range w.audit[1:] {
		fields = append(fields, r.Field+":"+r.OldValue+">"+r.NewValue)
	}
	if want := []string{"owner:>u7", "owner:u7>", "completed:false>true"}; !slices.Equal(fields, want) {
		t.Errorf("auditoría = %v", fields)
	}

	// Sin cambios no se audita nada; si la auditoría falla no se guarda nada.
	audited := len(w.audit)
	if _, err := update(manager, usecase.UpdateMilestoneInput{Completed: ptr(true)}); err != nil || len(w.audit) != audited {
		t.Errorf("una edición sin cambios auditó: %v", err)
	}
	w.auditErr = errors.New("auditoría caída")
	if _, err := update(manager, usecase.UpdateMilestoneInput{Title: ptr("Otro")}); err == nil {
		t.Fatal("debía fallar")
	}
	if w.milestones[id].Snapshot().Title != "Congelamiento" {
		t.Error("se guardó un cambio sin auditar")
	}
}

func TestListMilestones(t *testing.T) {
	uc, _, _ := setup(t)
	create(t, uc, "Atrasado", "2026-11-01")
	create(t, uc, "Próximo", "2026-12-01")
	done := create(t, uc, "Cumplido", "2026-10-01").Milestone.Id
	if _, err := uc.UpdateMilestone.Execute(ctx, usecase.UpdateMilestoneInput{Actor: manager, Id: done, Completed: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	titles := func(status string) []string {
		out, err := uc.ListMilestones.Execute(ctx, usecase.ListMilestonesInput{Status: status, Page: 1, Size: 20})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, m := range out.Milestones {
			got = append(got, fmt.Sprintf("%s:%d", m.Milestone.Title, m.DaysLate))
		}
		return got
	}
	if got := titles(""); !slices.Equal(got, []string{"Cumplido:0", "Atrasado:11", "Próximo:0"}) {
		t.Errorf("todos = %v", got)
	}
	if got := titles("pending"); !slices.Equal(got, []string{"Atrasado:11", "Próximo:0"}) {
		t.Errorf("pendientes = %v", got)
	}
	if got := titles("completed"); !slices.Equal(got, []string{"Cumplido:0"}) {
		t.Errorf("cumplidos = %v", got)
	}
	if _, err := uc.ListMilestones.Execute(ctx, usecase.ListMilestonesInput{Status: "otro"}); !slices.Equal(fieldsOf(err), []string{"status:invalid"}) {
		t.Errorf("estado raro: %v", err)
	}
}

func TestGetCalendar(t *testing.T) {
	uc, w, _ := setup(t)
	create(t, uc, "Congelamiento", "2026-11-09")
	create(t, uc, "Próximo", "2026-12-01")
	w.windows = []domain.Window{
		{ChangeCode: "CHG-2026-0001", ChangeTitle: "Mensajería", StateKey: "in_progress", Start: t0.Add(-48 * time.Hour), End: t0.Add(-time.Hour)},
		{ChangeCode: "CHG-2026-0002", ChangeTitle: "Pagos", StateKey: "approved", Start: t0.Add(24 * time.Hour), End: t0.Add(48 * time.Hour)},
	}
	out, err := uc.GetCalendar.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Windows) != 2 || out.Windows[0].DaysLate != 1 || out.Windows[1].DaysLate != 0 {
		t.Errorf("ventanas = %+v", out.Windows)
	}
	if len(out.Alerts) != 2 || out.Alerts[0].Title != "Congelamiento" || out.Alerts[0].DaysLate != 3 || out.Alerts[1].ChangeCode != "CHG-2026-0001" {
		t.Errorf("alertas = %+v", out.Alerts)
	}
}
