package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/defect/application/port"
	"kaoru/internal/feature/defect/application/usecase"
	"kaoru/internal/feature/defect/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	defects  map[string]domain.Defect // por código
	audit    []port.AuditRecord
	sequence int
	auditErr error
	now      time.Time
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	defects, audit, sequence := maps.Clone(w.defects), slices.Clone(w.audit), w.sequence
	if err := fn(ctx); err != nil {
		w.defects, w.audit, w.sequence = defects, audit, sequence
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

func (w *world) NextCode(_ context.Context, year int) (shared.Code, error) {
	w.sequence++
	return shared.NewCode(shared.PrefixDefect, year, w.sequence), nil
}

func (w *world) FindRequirement(_ context.Context, code string) (domain.RequirementRef, error) {
	if code != "REQ-2026-0042" {
		return domain.RequirementRef{}, notFound
	}
	return domain.RequirementRef{Id: "r1", Code: code}, nil
}

func (w *world) FindRun(_ context.Context, code string) (domain.RunRef, error) {
	runs := map[string]domain.RunRef{
		"RUN-2026-0001": {Id: "x1", Code: "RUN-2026-0001", Result: "failed", TestCaseId: "t1"},
		"RUN-2026-0002": {Id: "x2", Code: "RUN-2026-0002", Result: "passed", TestCaseId: "t1"},
		"RUN-2026-0003": {Id: "x3", Code: "RUN-2026-0003", Result: "failed", TestCaseId: "t9"},
	}
	run, ok := runs[code]
	if !ok {
		return domain.RunRef{}, notFound
	}
	return run, nil
}

func (w *world) FindUser(_ context.Context, id shared.UserId) (domain.UserRef, error) {
	if !strings.HasPrefix(string(id), "u") {
		return domain.UserRef{}, notFound
	}
	return domain.UserRef{Id: id, Name: "Usuario " + string(id)}, nil
}

func (w *world) Insert(_ context.Context, d domain.Defect) error {
	w.defects[d.Snapshot().Code.String()] = d
	return nil
}

func (w *world) Update(_ context.Context, d domain.Defect) error {
	w.defects[d.Snapshot().Code.String()] = d
	return nil
}

func (w *world) FindByCode(_ context.Context, code shared.Code) (domain.Defect, error) {
	d, ok := w.defects[code.String()]
	if !ok {
		return domain.Defect{}, notFound
	}
	return d, nil
}

func (w *world) List(_ context.Context, f port.Filter) ([]domain.Defect, int, error) {
	var out []domain.Defect
	for _, d := range w.defects {
		s := d.Snapshot()
		if (f.OnlyOpen && s.State.IsFinal) || (f.RequirementCode != "" && s.Requirement.Code != f.RequirementCode) {
			continue
		}
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b domain.Defect) int {
		return strings.Compare(a.Snapshot().Code.String(), b.Snapshot().Code.String())
	})
	return out, len(out), nil
}

func (w *world) RequirementTestCases(context.Context, string) ([]string, error) {
	return []string{"t1", "t2"}, nil
}

// LastTransitions lee la «auditoría» en memoria: cada transición ocurrió en w.now de ese momento.
func (w *world) LastTransitions(_ context.Context, ids []string) (map[string]time.Time, error) {
	out := map[string]time.Time{}
	for _, a := range w.audit {
		if a.Action == port.AuditTransition && slices.Contains(ids, a.DefectId) {
			out[a.DefectId] = transitionTimes[a.DefectId+a.NewValue]
		}
	}
	return out, nil
}

var transitionTimes = map[string]time.Time{}

func (w *world) Record(_ context.Context, r port.AuditRecord) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	if r.Action == port.AuditTransition {
		transitionTimes[r.DefectId+r.NewValue] = w.now
	}
	w.audit = append(w.audit, r)
	return nil
}

func (w *world) Now() time.Time { return w.now }

// fakeWorkflow: Abierto → Triado (qa) → Diferido (change-manager, con motivo, nunca si es alta)
// y Triado → Cerrado (qa).
type fakeWorkflow struct{}

var states = map[string]domain.State{
	"open":     {Id: "s1", Key: "open", Name: "Abierto"},
	"triaged":  {Id: "s2", Key: "triaged", Name: "Triado"},
	"deferred": {Id: "s3", Key: "deferred", Name: "Diferido"},
	"closed":   {Id: "s4", Key: "closed", Name: "Cerrado", IsFinal: true},
}

func (fakeWorkflow) Require(_ context.Context, roles []string, _ string) error {
	if slices.Contains(roles, "qa") {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

func (fakeWorkflow) InitialState(context.Context) (domain.State, error) { return states["open"], nil }

func (fakeWorkflow) ResolveTransition(_ context.Context, from, to string, roles []string) (port.Transition, error) {
	type step struct {
		role   string
		reason bool
		rules  []domain.Rule
	}
	allowed := map[string]step{
		"open>triaged":   {role: "qa"},
		"triaged>closed": {role: "qa"},
		"triaged>deferred": {role: "change-manager", reason: true, rules: []domain.Rule{
			{Key: domain.RuleSeverityIsNot, Params: map[string]any{"severity": "high"}, Blocks: true},
		}},
	}
	s, ok := allowed[from+">"+to]
	if !ok {
		return port.Transition{}, axis.New(axis.InvalidAction, "No se puede.")
	}
	if !slices.Contains(roles, s.role) {
		return port.Transition{}, axis.New(axis.PermissionDenied, "Sin permiso.")
	}
	return port.Transition{To: states[to], RequiresReason: s.reason, Rules: s.rules}, nil
}

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	t0      = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	qa      = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	dev     = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	ctx     = context.Background()
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{defects: map[string]domain.Defect{}, now: t0}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Workflow: fakeWorkflow{}, TestDesign: w, Timeline: w, AuditTrail: w,
		UnitOfWork: w, Clock: w, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

func open(t *testing.T, uc *usecase.UseCases, severity string) string {
	t.Helper()
	out, err := uc.CreateDefect.Execute(ctx, usecase.CreateDefectInput{
		Actor: qa, Title: "Saldo negativo", Description: "Pasos", RequirementCode: "REQ-2026-0042",
		RunCode: "RUN-2026-0001", Severity: severity,
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.Defect.Code.String()
}

func move(uc *usecase.UseCases, actor shared.Actor, code, to, reason string) error {
	_, err := uc.TransitionDefect.Execute(ctx, usecase.TransitionDefectInput{Actor: actor, Code: code, To: to, Reason: reason})
	return err
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

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestCreateDefect(t *testing.T) {
	uc, w := setup(t)
	code := open(t, uc, "high")
	s := w.defects[code].Snapshot()
	if code != "DEF-2026-0001" || s.State.Key != "open" || s.Run.Code != "RUN-2026-0001" || s.CreatedBy.Name != "Usuario u2" ||
		len(w.audit) != 1 || w.audit[0].Action != port.AuditCreate || w.audit[0].NewValue != "high" {
		t.Errorf("DEF o auditoría inesperados: %+v / %+v", s, w.audit)
	}
}

func TestCreateDefectErrors(t *testing.T) {
	base := usecase.CreateDefectInput{
		Actor: qa, Title: "x", Description: "y", RequirementCode: "REQ-2026-0042", RunCode: "RUN-2026-0001", Severity: "low",
	}
	cases := []struct {
		name       string
		mutate     func(in *usecase.CreateDefectInput)
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", func(in *usecase.CreateDefectInput) { in.Actor = dev }, axis.PermissionDenied, nil},
		{"sin REQ ni RUN", func(in *usecase.CreateDefectInput) { in.RequirementCode, in.RunCode = "", "" }, axis.InvalidRequest, []string{"requirement:required", "run:required"}},
		{"REQ y RUN inexistentes", func(in *usecase.CreateDefectInput) { in.RequirementCode, in.RunCode = "REQ-2026-0099", "RUN-2026-0099" }, axis.InvalidRequest, []string{"requirement:invalid", "run:invalid"}},
		{"RUN aprobado", func(in *usecase.CreateDefectInput) { in.RunCode = "RUN-2026-0002" }, axis.InvalidRequest, []string{"run:invalid"}},
		{"RUN de un TC de otro REQ", func(in *usecase.CreateDefectInput) { in.RunCode = "RUN-2026-0003" }, axis.InvalidRequest, []string{"run:invalid"}},
		{"asignado a nadie conocido", func(in *usecase.CreateDefectInput) { in.AssigneeId = "x" }, axis.InvalidRequest, []string{"assignee:invalid"}},
		{"sin título ni severidad", func(in *usecase.CreateDefectInput) { in.Title, in.Severity = "", "" }, axis.InvalidRequest, []string{"title:required", "severity:required"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			in := base
			tc.mutate(&in)
			_, err := uc.CreateDefect.Execute(ctx, in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.defects) != 0 || len(w.audit) != 0 || w.sequence != 0 {
				t.Error("un alta rechazada no debe dejar nada, ni gastar un código")
			}
		})
	}
}

func TestTransitionDefect(t *testing.T) {
	uc, w := setup(t)
	high, low := open(t, uc, "high"), open(t, uc, "low")
	for _, code := range []string{high, low} {
		if err := move(uc, dev, code, "triaged", ""); !axis.IsCode(err, axis.PermissionDenied) {
			t.Fatalf("dev no tría: %v", err)
		}
		if err := move(uc, qa, code, "triaged", ""); err != nil {
			t.Fatal(err)
		}
	}
	audited := len(w.audit)

	cases := []struct {
		name     string
		err      error
		wantCode axis.Code
	}{
		{"qa no difiere", move(uc, qa, low, "deferred", "Cosmético"), axis.PermissionDenied},
		{"diferir exige motivo", move(uc, manager, low, "deferred", "  "), axis.InvalidRequest},
		{"un DEF alto no se difiere", move(uc, manager, high, "deferred", "Lo pide negocio"), axis.InvalidAction},
		{"el flujo no lo permite", move(uc, qa, low, "open", ""), axis.InvalidAction},
		{"sin estado destino", move(uc, qa, low, "", ""), axis.InvalidRequest},
		{"DEF inexistente", move(uc, qa, "DEF-2026-0099", "closed", ""), axis.NotFoundResource},
	}
	for _, tc := range cases {
		if !axis.IsCode(tc.err, tc.wantCode) {
			t.Errorf("%s: esperaba %s, obtuve %v", tc.name, tc.wantCode, tc.err)
		}
	}
	if len(w.audit) != audited || w.defects[low].Snapshot().State.Key != "triaged" || w.defects[high].Snapshot().State.Key != "triaged" {
		t.Fatal("una transición rechazada no cambia el estado ni se audita")
	}
	if err := move(uc, manager, high, "deferred", "x"); err == nil || !strings.Contains(err.Error(), "severidad alta") {
		t.Errorf("el mensaje debe decir por qué: %v", err)
	}

	if err := move(uc, manager, low, "deferred", " Cosmético "); err != nil {
		t.Fatal(err)
	}
	last := w.audit[len(w.audit)-1]
	if w.defects[low].Snapshot().State.Key != "deferred" || last.Actor != "u1" || last.Reason != "Cosmético" ||
		last.OldValue != "triaged" || last.NewValue != "deferred" {
		t.Errorf("quién aceptó diferir y por qué debe quedar en la auditoría: %+v", last)
	}
}

func TestDaysInStateAndOpenDefects(t *testing.T) {
	uc, w := setup(t)
	high, low := open(t, uc, "high"), open(t, uc, "low")

	// Sin transiciones, cuenta desde que se abrió.
	w.now = t0.Add(36 * time.Hour)
	out, err := uc.GetDefect.Execute(ctx, usecase.GetDefectInput{Code: high})
	if err != nil || out.DaysInState != 1.5 {
		t.Fatalf("días en estado = %v, %v", out.DaysInState, err)
	}
	// Tras una transición, cuenta desde ella.
	if err := move(uc, qa, high, "triaged", ""); err != nil {
		t.Fatal(err)
	}
	w.now = t0.Add(60 * time.Hour)
	if out, _ = uc.GetDefect.Execute(ctx, usecase.GetDefectInput{Code: high}); out.DaysInState != 1 {
		t.Errorf("tras triar, días en estado = %v, quiero 1", out.DaysInState)
	}
	list, err := uc.ListDefects.Execute(ctx, usecase.ListDefectsInput{Page: 1, Size: 25})
	if err != nil || list.Total != 2 || list.Defects[0].DaysInState != 1 || list.Defects[1].DaysInState != 2.5 {
		t.Errorf("la lista calcula el tiempo de cada uno: %+v, %v", list, err)
	}

	// Un DEF cerrado deja de contar como abierto.
	open, _ := uc.ListOpenDefects.Execute(ctx, usecase.ListOpenDefectsInput{RequirementCode: "REQ-2026-0042"})
	if len(open) != 2 {
		t.Fatalf("defectos abiertos = %+v", open)
	}
	if err := move(uc, qa, high, "closed", ""); err != nil {
		t.Fatal(err)
	}
	open, _ = uc.ListOpenDefects.Execute(ctx, usecase.ListOpenDefectsInput{RequirementCode: "REQ-2026-0042"})
	if len(open) != 1 || open[0].Code != low || open[0].Severity != "low" {
		t.Errorf("tras cerrar el alto solo queda el bajo: %+v", open)
	}
}

func TestUpdateDefect(t *testing.T) {
	uc, w := setup(t)
	code := open(t, uc, "high")
	severity := "medium"
	out, err := uc.UpdateDefect.Execute(ctx, usecase.UpdateDefectInput{Actor: qa, Code: code, Severity: &severity, AssigneeSet: true, AssigneeId: "u3"})
	if err != nil || out.Defect.Severity != domain.SeverityMedium || out.Defect.Assignee == nil || out.Defect.Assignee.Name != "Usuario u3" || len(w.audit) != 3 {
		t.Errorf("edición: %+v, %v, %d registros", out.Defect, err, len(w.audit))
	}
	if _, err := uc.UpdateDefect.Execute(ctx, usecase.UpdateDefectInput{Actor: dev, Code: code, Severity: &severity}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("sin permiso: %v", err)
	}
	if _, err := uc.UpdateDefect.Execute(ctx, usecase.UpdateDefectInput{Actor: qa, Code: code, AssigneeSet: true, AssigneeId: "nadie"}); !slices.Equal(fieldsOf(err), []string{"assignee:invalid"}) {
		t.Errorf("asignado inexistente: %v", err)
	}
	if _, err := uc.UpdateDefect.Execute(ctx, usecase.UpdateDefectInput{Actor: qa, Code: "DEF-2026-0099", Severity: &severity}); !errors.Is(err, usecase.ErrDefectNotFound) {
		t.Errorf("DEF inexistente: %v", err)
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	code := open(t, uc, "low")
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")
	severity := "high"

	if _, err := uc.CreateDefect.Execute(ctx, usecase.CreateDefectInput{
		Actor: qa, Title: "x", Description: "y", RequirementCode: "REQ-2026-0042", RunCode: "RUN-2026-0001", Severity: "low",
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("crear: %v", err)
	}
	if _, err := uc.UpdateDefect.Execute(ctx, usecase.UpdateDefectInput{Actor: qa, Code: code, Severity: &severity}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("editar: %v", err)
	}
	if err := move(uc, qa, code, "triaged", ""); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("transición: %v", err)
	}
	s := w.defects[code].Snapshot()
	if len(w.defects) != 1 || s.Severity != domain.SeverityLow || s.State.Key != "open" || w.sequence != 1 || len(w.audit) != 1 {
		t.Error("si la auditoría falla, nada debe quedar guardado")
	}
}

// Lock no hace nada en memoria: el doble ya ejecuta una transacción a la vez.
func (w *world) Lock(ctx context.Context, code shared.Code) error {
	_, err := w.FindByCode(ctx, code)
	return err
}
