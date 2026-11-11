package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/change/application/port"
	"kaoru/internal/feature/change/application/usecase"
	"kaoru/internal/feature/change/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria que comparten los dobles. Las escrituras hechas dentro
// de Within se descartan si la función falla, como en una transacción real.
type world struct {
	changes  map[string]domain.Change
	audit    []port.AuditRecord
	sequence int

	auditErr error
	inTx     bool
}

func newWorld() *world { return &world{changes: map[string]domain.Change{}} }

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	changes, audit, sequence := cloneMap(w.changes), slices.Clone(w.audit), w.sequence
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.changes, w.audit, w.sequence = changes, audit, sequence
		return err
	}
	return nil
}

func cloneMap(m map[string]domain.Change) map[string]domain.Change {
	out := make(map[string]domain.Change, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (w *world) NextCode(_ context.Context, year int) (shared.Code, error) {
	w.sequence++
	return shared.NewCode(shared.PrefixChange, year, w.sequence), nil
}

func (w *world) FindSystem(_ context.Context, key string) (domain.System, error) {
	if key != "SYPAGO" {
		return domain.System{}, axis.New(axis.NotFoundResource, "No existe.")
	}
	return domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: domain.RouteLight}, nil
}

func (w *world) Insert(_ context.Context, c domain.Change) error {
	if !w.inTx {
		return errors.New("Insert fuera de una transacción")
	}
	w.changes[c.Snapshot().Code.String()] = c
	return nil
}

func (w *world) Update(_ context.Context, c domain.Change) error {
	if !w.inTx {
		return errors.New("Update fuera de una transacción")
	}
	w.changes[c.Snapshot().Code.String()] = c
	return nil
}

func (w *world) FindByCode(_ context.Context, code shared.Code) (port.ChangeView, error) {
	c, ok := w.changes[code.String()]
	if !ok {
		return port.ChangeView{}, axis.New(axis.NotFoundResource, "No existe.")
	}
	return port.ChangeView{Change: c, CreatedByName: "Ana"}, nil
}

func (w *world) List(context.Context, port.Filter) ([]port.ChangeView, int, error) {
	var views []port.ChangeView
	for _, c := range w.changes {
		views = append(views, port.ChangeView{Change: c, CreatedByName: "Ana"})
	}
	return views, len(views), nil
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

// fakeWorkflow: change-manager puede todo; Registrado → Aprobado y Registrado → Rechazado (con motivo).
type fakeWorkflow struct{}

var states = map[string]domain.State{
	"registered": {Id: "s1", Key: "registered", Name: "Registrado"},
	"approved":   {Id: "s2", Key: "approved", Name: "Aprobado"},
	"rejected":   {Id: "s3", Key: "rejected", Name: "Rechazado"},
}

func (fakeWorkflow) Require(_ context.Context, roles []string, _ string) error {
	if slices.Contains(roles, "change-manager") {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

func (fakeWorkflow) InitialState(context.Context) (domain.State, error) {
	return states["registered"], nil
}

func (f fakeWorkflow) ResolveTransition(ctx context.Context, from, to string, roles []string) (port.Transition, error) {
	if from != "registered" || (to != "approved" && to != "rejected") {
		return port.Transition{}, axis.New(axis.InvalidAction, "No se puede.")
	}
	if err := f.Require(ctx, roles, ""); err != nil {
		return port.Transition{}, err
	}
	return port.Transition{To: states[to], RequiresReason: to == "rejected"}, nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	now     = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	tester  = shared.Actor{Id: "u2", Roles: []string{"qa"}}
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := newWorld()
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Workflow: fakeWorkflow{}, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{now}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

func create(t *testing.T, uc *usecase.UseCases) usecase.ChangeOutput {
	t.Helper()
	out, err := uc.CreateChange.Execute(context.Background(), usecase.CreateChangeInput{
		Actor: manager, Title: "Límite diario", SystemKey: "SYPAGO",
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestCreateChange(t *testing.T) {
	uc, w := setup(t)
	out := create(t, uc)

	s := out.Change
	if s.Code.String() != "CHG-2026-0001" || s.State.Key != "registered" || s.Route != domain.RouteLight ||
		s.CreatedBy != "u1" || s.CreatedAt != now || out.CreatedByName != "Ana" {
		t.Errorf("CHG inesperado: %+v", out)
	}
	if len(w.audit) != 1 || w.audit[0].Action != port.AuditCreate || w.audit[0].Actor != "u1" ||
		w.audit[0].Code != s.Code || w.audit[0].ChangeId != s.Id {
		t.Errorf("auditoría inesperada: %+v", w.audit)
	}
	if second := create(t, uc); second.Change.Code.String() != "CHG-2026-0002" {
		t.Errorf("el segundo código = %s", second.Change.Code)
	}
}

func TestCreateChangeErrors(t *testing.T) {
	cases := []struct {
		name      string
		in        usecase.CreateChangeInput
		wantCode  axis.Code
		wantField string
	}{
		{"sin permiso", usecase.CreateChangeInput{Actor: tester, Title: "x", SystemKey: "SYPAGO"}, axis.PermissionDenied, ""},
		{"sin sistema", usecase.CreateChangeInput{Actor: manager, Title: "x"}, axis.InvalidRequest, "system:required"},
		{"sistema desconocido", usecase.CreateChangeInput{Actor: manager, Title: "x", SystemKey: "OTRO"}, axis.InvalidRequest, "system:invalid"},
		{"sin título", usecase.CreateChangeInput{Actor: manager, SystemKey: "SYPAGO"}, axis.InvalidRequest, "title:required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			_, err := uc.CreateChange.Execute(context.Background(), tc.in)
			if !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if tc.wantField != "" {
				var ge *axis.GeneralError
				errors.As(err, &ge)
				if f := ge.Fields(); len(f) != 1 || f[0].Field+":"+f[0].Code != tc.wantField {
					t.Errorf("campos = %+v, quiero %s", f, tc.wantField)
				}
			}
			if len(w.changes) != 0 || len(w.audit) != 0 || w.sequence != 0 {
				t.Error("un alta rechazada no debe dejar cambios, auditoría ni gastar un código")
			}
		})
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	created := create(t, uc)
	before := w.changes[created.Change.Code.String()].Snapshot()
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")
	ctx := context.Background()
	title := "Otro"

	if _, err := uc.CreateChange.Execute(ctx, usecase.CreateChangeInput{Actor: manager, Title: "x", SystemKey: "SYPAGO"}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("crear: %v", err)
	}
	if _, err := uc.UpdateChange.Execute(ctx, usecase.UpdateChangeInput{Actor: manager, Code: "CHG-2026-0001", Edit: domain.Edit{Title: &title}}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("editar: %v", err)
	}
	if _, err := uc.TransitionChange.Execute(ctx, usecase.TransitionChangeInput{Actor: manager, Code: "CHG-2026-0001", To: "approved"}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("transición: %v", err)
	}
	if len(w.changes) != 1 || w.changes["CHG-2026-0001"].Snapshot() != before || w.sequence != 1 || len(w.audit) != 1 {
		t.Error("si la auditoría falla, nada debe quedar guardado")
	}
}

func TestGetChange(t *testing.T) {
	uc, _ := setup(t)
	create(t, uc)
	if out, err := uc.GetChange.Execute(context.Background(), usecase.GetChangeInput{Code: "CHG-2026-0001"}); err != nil || out.Change.Title != "Límite diario" {
		t.Errorf("GetChange = %+v, %v", out, err)
	}
	for _, code := range []string{"CHG-2026-0099", "no-es-un-codigo", ""} {
		if _, err := uc.GetChange.Execute(context.Background(), usecase.GetChangeInput{Code: code}); !errors.Is(err, usecase.ErrChangeNotFound) {
			t.Errorf("GetChange(%q): esperaba ErrChangeNotFound, obtuve %v", code, err)
		}
	}
}

func TestUpdateChange(t *testing.T) {
	uc, w := setup(t)
	create(t, uc)
	ctx := context.Background()
	title, route := "Límite semanal", "high"

	out, err := uc.UpdateChange.Execute(ctx, usecase.UpdateChangeInput{
		Actor: manager, Code: "CHG-2026-0001", Edit: domain.Edit{Title: &title, Route: &route},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Change.Title != title || out.Change.Route != domain.RouteHigh {
		t.Errorf("no aplicó la edición: %+v", out.Change)
	}
	if len(w.audit) != 3 || w.audit[1].Field != "title" || w.audit[1].OldValue != "Límite diario" ||
		w.audit[1].NewValue != title || w.audit[2].Field != "route" || w.audit[2].Action != port.AuditUpdate {
		t.Errorf("debe auditar cada campo que cambió: %+v", w.audit[1:])
	}

	// Editar sin cambiar nada no escribe ni audita.
	if _, err := uc.UpdateChange.Execute(ctx, usecase.UpdateChangeInput{Actor: manager, Code: "CHG-2026-0001", Edit: domain.Edit{Title: &title}}); err != nil {
		t.Fatal(err)
	}
	if len(w.audit) != 3 {
		t.Error("una edición sin cambios no debe auditarse")
	}

	if _, err := uc.UpdateChange.Execute(ctx, usecase.UpdateChangeInput{Actor: tester, Code: "CHG-2026-0001", Edit: domain.Edit{Title: &title}}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("sin permiso: %v", err)
	}
	if _, err := uc.UpdateChange.Execute(ctx, usecase.UpdateChangeInput{Actor: manager, Code: "CHG-2026-0099", Edit: domain.Edit{Title: &title}}); !errors.Is(err, usecase.ErrChangeNotFound) {
		t.Errorf("CHG inexistente: %v", err)
	}
}

func TestTransitionChange(t *testing.T) {
	cases := []struct {
		name      string
		in        usecase.TransitionChangeInput
		wantCode  axis.Code
		wantState string
	}{
		{"aprueba", usecase.TransitionChangeInput{Actor: manager, Code: "CHG-2026-0001", To: "approved"}, "", "approved"},
		{"rechaza con motivo", usecase.TransitionChangeInput{Actor: manager, Code: "CHG-2026-0001", To: "rejected", Reason: " Duplicado "}, "", "rejected"},
		{"rechazar exige motivo", usecase.TransitionChangeInput{Actor: manager, Code: "CHG-2026-0001", To: "rejected", Reason: "  "}, axis.InvalidRequest, "registered"},
		{"sin estado destino", usecase.TransitionChangeInput{Actor: manager, Code: "CHG-2026-0001"}, axis.InvalidRequest, "registered"},
		{"el flujo no lo permite", usecase.TransitionChangeInput{Actor: manager, Code: "CHG-2026-0001", To: "closed"}, axis.InvalidAction, "registered"},
		{"sin permiso", usecase.TransitionChangeInput{Actor: tester, Code: "CHG-2026-0001", To: "approved"}, axis.PermissionDenied, "registered"},
		{"CHG inexistente", usecase.TransitionChangeInput{Actor: manager, Code: "CHG-2026-0099", To: "approved"}, axis.NotFoundResource, "registered"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			create(t, uc)
			out, err := uc.TransitionChange.Execute(context.Background(), tc.in)
			if tc.wantCode != "" && !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if tc.wantCode == "" && err != nil {
				t.Fatal(err)
			}
			if got := w.changes["CHG-2026-0001"].Snapshot().State.Key; got != tc.wantState {
				t.Errorf("estado guardado = %s, quiero %s", got, tc.wantState)
			}
			if tc.wantCode != "" {
				if len(w.audit) != 1 {
					t.Error("una transición rechazada no debe auditarse")
				}
				return
			}
			last := w.audit[len(w.audit)-1]
			if out.Change.State.Key != tc.wantState || last.Action != port.AuditTransition || last.Field != "state" ||
				last.OldValue != "registered" || last.NewValue != tc.wantState || last.Reason != "Duplicado" && tc.in.Reason != "" {
				t.Errorf("salida o auditoría inesperada: %+v / %+v", out.Change.State, last)
			}
		})
	}
}

// Lock no hace nada en memoria: el doble ya ejecuta una transacción a la vez.
func (w *world) Lock(ctx context.Context, code shared.Code) error {
	_, err := w.FindByCode(ctx, code)
	return err
}
