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
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria que comparten los dobles. Lo escrito dentro de Within
// se descarta si la función falla, como en una transacción real.
type world struct {
	requirements map[string]domain.Requirement // por código
	links        map[string]domain.Link        // por id
	checks       map[string]string             // ítem → nota
	components   map[string][]string           // id del REQ → códigos de COMP
	audit        []port.AuditRecord
	sequence     int

	risk     *domain.RiskFact
	coverage domain.CoverageFact
	defects  []domain.DefectFact
	auditErr error
}

func newWorld() *world {
	return &world{
		requirements: map[string]domain.Requirement{}, links: map[string]domain.Link{}, checks: map[string]string{},
		components: map[string][]string{},
	}
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	requirements, links, checks := maps.Clone(w.requirements), maps.Clone(w.links), maps.Clone(w.checks)
	components, audit, sequence := maps.Clone(w.components), slices.Clone(w.audit), w.sequence
	if err := fn(ctx); err != nil {
		w.requirements, w.links, w.checks, w.audit, w.sequence = requirements, links, checks, audit, sequence
		w.components = components
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

func (w *world) NextCode(_ context.Context, year int) (shared.Code, error) {
	w.sequence++
	return shared.NewCode(shared.PrefixRequirement, year, w.sequence), nil
}

func (w *world) FindSystem(_ context.Context, key string) (domain.System, error) {
	switch key {
	case "SYPAGO":
		return domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"}, nil
	case "SIMF":
		return domain.System{Key: "SIMF", Name: "SIMF", DefaultRoute: "high"}, nil
	}
	return domain.System{}, notFound
}

func (w *world) FindChange(_ context.Context, code string) (domain.ChangeRef, error) {
	if code != "CHG-2026-0007" {
		return domain.ChangeRef{}, notFound
	}
	return domain.ChangeRef{Id: "c1", Code: code, Title: "Pagos", Route: "light", SystemKey: "SYPAGO"}, nil
}

func (w *world) FindUser(_ context.Context, id shared.UserId) (domain.UserRef, error) {
	if !strings.HasPrefix(string(id), "u") {
		return domain.UserRef{}, notFound
	}
	return domain.UserRef{Id: id, Name: "Usuario " + string(id)}, nil
}

func (w *world) Insert(_ context.Context, r domain.Requirement) error {
	w.requirements[r.Snapshot().Code.String()] = r
	return nil
}

func (w *world) Update(_ context.Context, r domain.Requirement) error {
	w.requirements[r.Snapshot().Code.String()] = r
	return nil
}

func (w *world) FindByCode(_ context.Context, code shared.Code) (domain.Requirement, error) {
	r, ok := w.requirements[code.String()]
	if !ok {
		return domain.Requirement{}, notFound
	}
	return r, nil
}

func (w *world) List(context.Context, port.Filter) ([]domain.Requirement, int, error) {
	return slices.Collect(maps.Values(w.requirements)), len(w.requirements), nil
}

func (w *world) InsertLink(_ context.Context, l domain.Link) error {
	w.links[l.Snapshot().Id] = l
	return nil
}

func (w *world) UpdateLink(_ context.Context, l domain.Link) error {
	w.links[l.Snapshot().Id] = l
	return nil
}

func (w *world) DeleteLink(_ context.Context, id string) error {
	delete(w.links, id)
	return nil
}

func (w *world) FindLink(_ context.Context, requirementId, linkId string) (domain.Link, error) {
	l, ok := w.links[linkId]
	if !ok || l.Snapshot().RequirementId != requirementId {
		return domain.Link{}, notFound
	}
	return l, nil
}

func (w *world) ListLinks(_ context.Context, requirementId string) ([]domain.Link, error) {
	var out []domain.Link
	for _, l := range w.links {
		if l.Snapshot().RequirementId == requirementId {
			out = append(out, l)
		}
	}
	return out, nil
}

func (w *world) FindDocumentId(_ context.Context, code string) (string, error) {
	if code != "DOC-2026-0031" {
		return "", notFound
	}
	return "d1", nil
}

func (w *world) ListEntryContract(context.Context, string) ([]domain.EntryItem, error) {
	items := []domain.EntryItem{{Key: "ids", Label: "IDs de REQ y CHG"}, {Key: "rollback", Label: "Notas de rollback"}}
	for i := range items {
		if note, ok := w.checks[items[i].Key]; ok {
			items[i].Done, items[i].Note = true, note
		}
	}
	return items, nil
}

func (w *world) CheckEntryItem(_ context.Context, _, key, note string, _ shared.UserId, _ time.Time) error {
	w.checks[key] = note
	return nil
}

func (w *world) UncheckEntryItem(_ context.Context, _, key string) error {
	delete(w.checks, key)
	return nil
}

func (w *world) Record(_ context.Context, r port.AuditRecord) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	w.audit = append(w.audit, r)
	return nil
}

func (w *world) Risk(context.Context, shared.Code) (*domain.RiskFact, error) { return w.risk, nil }
func (w *world) Coverage(context.Context, shared.Code) (domain.CoverageFact, error) {
	return w.coverage, nil
}
func (w *world) OpenDefects(context.Context, shared.Code) ([]domain.DefectFact, error) {
	return w.defects, nil
}

// fakeWorkflow es un flujo corto: Registrado → Riesgo evaluado → Entregado a QA → Aprobado QA,
// y Registrado → Rechazado con motivo. Cada rol solo puede lo suyo.
type fakeWorkflow struct{}

var (
	states = map[string]domain.State{
		"registered":    {Id: "s1", Key: "registered", Name: "Registrado"},
		"risk_assessed": {Id: "s2", Key: "risk_assessed", Name: "Riesgo evaluado"},
		"delivered":     {Id: "s3", Key: "delivered", Name: "Entregado a QA"},
		"approved_qa":   {Id: "s4", Key: "approved_qa", Name: "Aprobado QA"},
		"rejected":      {Id: "s5", Key: "rejected", Name: "Rechazado"},
	}
	transitions = []struct {
		from, to, role string
		reason         bool
		rules          []domain.Rule
	}{
		{"registered", "risk_assessed", "qa", false, []domain.Rule{{Key: domain.RuleRiskAssessed, Blocks: true}}},
		{"registered", "rejected", "change-manager", true, nil},
		{"risk_assessed", "delivered", "dev", false, []domain.Rule{{Key: domain.RuleEntryContractComplete, Blocks: true}}},
		{"delivered", "approved_qa", "qa", false, []domain.Rule{
			{Key: domain.RuleMinCoverage, Params: map[string]any{"low": 100.0}, Blocks: true},
			{Key: domain.RuleNoOpenDefect, Params: map[string]any{"severity": "high"}, Blocks: true},
			{Key: domain.RuleLinksHaveSnapshot, Blocks: false},
		}},
	}
	grants = map[string][]string{
		"change-manager": {usecase.PermissionCreate, usecase.PermissionEdit, usecase.PermissionManageLinks},
		"dev":            {usecase.PermissionEdit, usecase.PermissionEntryContract},
		"qa":             {usecase.PermissionEdit, usecase.PermissionManageLinks},
	}
)

func (fakeWorkflow) Require(_ context.Context, roles []string, permission string) error {
	for _, role := range roles {
		if slices.Contains(grants[role], permission) {
			return nil
		}
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

func (fakeWorkflow) InitialState(context.Context) (domain.State, error) {
	return states["registered"], nil
}

func (fakeWorkflow) ResolveTransition(_ context.Context, from, to string, roles []string) (port.Transition, error) {
	for _, t := range transitions {
		if t.from == from && t.to == to {
			if !slices.Contains(roles, t.role) {
				return port.Transition{}, axis.New(axis.PermissionDenied, "Sin permiso.")
			}
			return port.Transition{To: states[to], RequiresReason: t.reason, Rules: t.rules}, nil
		}
	}
	return port.Transition{}, axis.New(axis.InvalidAction, "No se puede.")
}

func (fakeWorkflow) Outgoing(_ context.Context, from string) ([]port.Transition, error) {
	var out []port.Transition
	for _, t := range transitions {
		if t.from == from {
			out = append(out, port.Transition{To: states[t.to], RequiresReason: t.reason, Rules: t.rules})
		}
	}
	return out, nil
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC) }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	tester  = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	dev     = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	auditor = shared.Actor{Id: "u4", Roles: []string{"auditor"}}
	ctx     = context.Background()
)

const (
	req1  = "REQ-2026-0001"
	sheet = "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789/edit#gid=0"
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := newWorld()
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Workflow: fakeWorkflow{}, Facts: w, Components: w, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateRequirement.Execute(ctx, usecase.CreateRequirementInput{
		Actor: manager, Title: "Límite diario", SystemKey: "SYPAGO", ChangeCode: "CHG-2026-0007",
	}); err != nil {
		t.Fatal(err)
	}
	return uc, w
}

func move(t *testing.T, uc *usecase.UseCases, actor shared.Actor, to string) error {
	t.Helper()
	_, err := uc.TransitionRequirement.Execute(ctx, usecase.TransitionRequirementInput{Actor: actor, Code: req1, To: to})
	return err
}

func state(w *world) string { return w.requirements[req1].Snapshot().State.Key }

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

func TestCreateRequirement(t *testing.T) {
	_, w := setup(t)
	s := w.requirements[req1].Snapshot()
	if s.State.Key != "registered" || s.Change == nil || s.Change.Code != "CHG-2026-0007" ||
		s.CreatedBy != (domain.UserRef{Id: "u1", Name: "Usuario u1"}) {
		t.Errorf("REQ inesperado: %+v", s)
	}
	if len(w.audit) != 1 || w.audit[0].Action != port.AuditCreate || w.audit[0].Code.String() != req1 || w.audit[0].Actor != "u1" {
		t.Errorf("auditoría inesperada: %+v", w.audit)
	}
}

func TestCreateRequirementErrors(t *testing.T) {
	cases := []struct {
		name       string
		in         usecase.CreateRequirementInput
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", usecase.CreateRequirementInput{Actor: tester, Title: "x", SystemKey: "SYPAGO"}, axis.PermissionDenied, nil},
		{"junta sistema, CHG y responsable", usecase.CreateRequirementInput{Actor: manager, Title: "x", ChangeCode: "CHG-2026-0099", OwnerId: "nadie"},
			axis.InvalidRequest, []string{"system:required", "change:invalid", "owner:invalid"}},
		{"sistema desconocido", usecase.CreateRequirementInput{Actor: manager, Title: "x", SystemKey: "OTRO"}, axis.InvalidRequest, []string{"system:invalid"}},
		{"CHG de otro sistema", usecase.CreateRequirementInput{Actor: manager, Title: "x", SystemKey: "SIMF", ChangeCode: "CHG-2026-0007"},
			axis.InvalidRequest, []string{"change:invalid"}},
		{"sin título", usecase.CreateRequirementInput{Actor: manager, SystemKey: "SYPAGO"}, axis.InvalidRequest, []string{"title:required"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			_, err := uc.CreateRequirement.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.requirements) != 1 || len(w.audit) != 1 || w.sequence != 1 {
				t.Error("un alta rechazada no debe dejar nada")
			}
		})
	}
}

func TestUpdateRequirement(t *testing.T) {
	uc, w := setup(t)
	title := "Límite semanal"
	out, err := uc.UpdateRequirement.Execute(ctx, usecase.UpdateRequirementInput{
		Actor: dev, Code: req1, Title: &title, ChangeSet: true, OwnerSet: true, OwnerId: "u2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != title || out.Change != nil || out.Owner == nil || out.Owner.Name != "Usuario u2" {
		t.Errorf("edición no aplicada: %+v", out)
	}
	var fields []string
	for _, a := range w.audit[1:] {
		fields = append(fields, a.Field+"="+a.NewValue)
	}
	if !slices.Equal(fields, []string{"title=Límite semanal", "change=", "owner=u2"}) {
		t.Errorf("auditoría de la edición: %v", fields)
	}
	if _, err := uc.UpdateRequirement.Execute(ctx, usecase.UpdateRequirementInput{Actor: auditor, Code: req1, Title: &title}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("sin permiso: %v", err)
	}
	if _, err := uc.UpdateRequirement.Execute(ctx, usecase.UpdateRequirementInput{Actor: dev, Code: "REQ-2026-0099", Title: &title}); !errors.Is(err, usecase.ErrRequirementNotFound) {
		t.Errorf("REQ inexistente: %v", err)
	}
	if _, err := uc.UpdateRequirement.Execute(ctx, usecase.UpdateRequirementInput{Actor: dev, Code: req1, ChangeSet: true, ChangeCode: "CHG-2026-0099"}); !slices.Equal(fieldsOf(err), []string{"change:invalid"}) {
		t.Errorf("CHG inexistente: %v", err)
	}
}

func TestTransitionIsBlockedUntilRulesAreMet(t *testing.T) {
	uc, w := setup(t)

	// Sin RISK no se puede marcar el riesgo como evaluado.
	err := move(t, uc, tester, "risk_assessed")
	if !axis.IsCode(err, axis.InvalidAction) || err.Error() != "No se puede pasar a «Riesgo evaluado». Falta evaluar el riesgo." {
		t.Fatalf("debía bloquear con lo que falta: %v", err)
	}
	if state(w) != "registered" || len(w.audit) != 1 {
		t.Fatal("una transición bloqueada no cambia el estado ni se audita")
	}
	w.risk = &domain.RiskFact{Level: "low"}
	if err := move(t, uc, tester, "risk_assessed"); err != nil {
		t.Fatal(err)
	}

	// Sin el contrato de entrada completo, dev no puede entregar a QA.
	err = move(t, uc, dev, "delivered")
	if !axis.IsCode(err, axis.InvalidAction) || !strings.Contains(err.Error(), "Faltan 2 ítems del contrato de entrada") {
		t.Fatalf("debía pedir el contrato de entrada: %v", err)
	}
	for _, key := range []string{"ids", "rollback"} {
		if _, err := uc.CheckEntryItem.Execute(ctx, usecase.CheckEntryItemInput{Actor: dev, Code: req1, ItemKey: key}); err != nil {
			t.Fatal(err)
		}
	}
	if err := move(t, uc, dev, "delivered"); err != nil {
		t.Fatal(err)
	}

	// Aprobar exige cobertura y cero defectos altos; el enlace sin instantánea solo avisa.
	if _, err := uc.AddLink.Execute(ctx, usecase.AddLinkInput{Actor: tester, Code: req1, URL: sheet}); err != nil {
		t.Fatal(err)
	}
	w.coverage = domain.CoverageFact{Required: 2, Pending: []string{"TC-2026-0112"}}
	w.defects = []domain.DefectFact{{Code: "DEF-2026-0091", Severity: "high"}}
	err = move(t, uc, tester, "approved_qa")
	if !axis.IsCode(err, axis.InvalidAction) || !strings.Contains(err.Error(), "TC-2026-0112") ||
		!strings.Contains(err.Error(), "DEF-2026-0091") || strings.Contains(err.Error(), "instantánea") {
		t.Fatalf("debía listar lo que bloquea y callar lo que solo avisa: %v", err)
	}

	readiness, err := uc.GetReadiness.Execute(ctx, usecase.GetReadinessInput{Code: req1})
	if err != nil || len(readiness) != 1 || readiness[0].To.Key != "approved_qa" || len(readiness[0].Conditions) != 3 {
		t.Fatalf("«qué falta para avanzar» debe traer las dos que bloquean y el aviso: %+v, %v", readiness, err)
	}
	if blocking := domain.Blocking(readiness[0].Conditions); len(blocking) != 2 {
		t.Errorf("bloquean %d, quiero 2", len(blocking))
	}

	w.coverage.Pending, w.defects = nil, nil
	if err := move(t, uc, tester, "approved_qa"); err != nil {
		t.Fatalf("con las reglas cumplidas debe avanzar aunque quede un aviso: %v", err)
	}
	last := w.audit[len(w.audit)-1]
	if state(w) != "approved_qa" || last.Action != port.AuditTransition || last.OldValue != "delivered" || last.NewValue != "approved_qa" {
		t.Errorf("transición o auditoría inesperada: %s %+v", state(w), last)
	}
}

func TestTransitionErrors(t *testing.T) {
	cases := []struct {
		name     string
		in       usecase.TransitionRequirementInput
		wantCode axis.Code
	}{
		{"sin estado destino", usecase.TransitionRequirementInput{Actor: tester, Code: req1}, axis.InvalidRequest},
		{"el flujo no lo permite", usecase.TransitionRequirementInput{Actor: tester, Code: req1, To: "approved_qa"}, axis.InvalidAction},
		{"otro rol", usecase.TransitionRequirementInput{Actor: dev, Code: req1, To: "risk_assessed"}, axis.PermissionDenied},
		{"rechazar exige motivo", usecase.TransitionRequirementInput{Actor: manager, Code: req1, To: "rejected", Reason: " "}, axis.InvalidRequest},
		{"REQ inexistente", usecase.TransitionRequirementInput{Actor: tester, Code: "REQ-2026-0099", To: "risk_assessed"}, axis.NotFoundResource},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			if _, err := uc.TransitionRequirement.Execute(ctx, tc.in); !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if state(w) != "registered" || len(w.audit) != 1 {
				t.Error("no debe cambiar el estado ni auditar")
			}
		})
	}

	uc, w := setup(t)
	if _, err := uc.TransitionRequirement.Execute(ctx, usecase.TransitionRequirementInput{Actor: manager, Code: req1, To: "rejected", Reason: " Duplicado "}); err != nil {
		t.Fatal(err)
	}
	if last := w.audit[len(w.audit)-1]; state(w) != "rejected" || last.Reason != "Duplicado" {
		t.Errorf("el motivo debe quedar en la auditoría: %+v", last)
	}
}

func TestLinks(t *testing.T) {
	uc, w := setup(t)
	link, err := uc.AddLink.Execute(ctx, usecase.AddLinkInput{Actor: tester, Code: req1, URL: sheet})
	if err != nil {
		t.Fatal(err)
	}
	if link.Kind != domain.LinkGoogleSheets || link.Sheet == nil || link.AddedBy.Name != "Usuario u2" {
		t.Errorf("enlace inesperado: %+v", link)
	}
	if _, err := uc.AddLink.Execute(ctx, usecase.AddLinkInput{Actor: dev, Code: req1, URL: sheet}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("dev no gestiona enlaces en este flujo: %v", err)
	}
	if _, err := uc.AddLink.Execute(ctx, usecase.AddLinkInput{Actor: tester, Code: req1, URL: "nada"}); !slices.Equal(fieldsOf(err), []string{"url:invalid"}) {
		t.Errorf("URL inválida: %v", err)
	}

	official := true
	updated, err := uc.UpdateLink.Execute(ctx, usecase.UpdateLinkInput{
		Actor: tester, Code: req1, LinkId: link.Id, IsOfficial: &official, SnapshotSet: true, SnapshotCode: "DOC-2026-0031",
	})
	if err != nil || !updated.IsOfficial || updated.SnapshotCode != "DOC-2026-0031" || updated.SnapshotDocumentId != "d1" {
		t.Errorf("adjuntar instantánea: %+v, %v", updated, err)
	}
	if _, err := uc.UpdateLink.Execute(ctx, usecase.UpdateLinkInput{Actor: tester, Code: req1, LinkId: link.Id, SnapshotSet: true, SnapshotCode: "DOC-2026-0099"}); !slices.Equal(fieldsOf(err), []string{"snapshotDocument:invalid"}) {
		t.Errorf("documento inexistente: %v", err)
	}
	if _, err := uc.UpdateLink.Execute(ctx, usecase.UpdateLinkInput{Actor: tester, Code: req1, LinkId: "otro"}); !errors.Is(err, usecase.ErrLinkNotFound) {
		t.Errorf("enlace inexistente: %v", err)
	}

	if err := uc.RemoveLink.Execute(ctx, usecase.RemoveLinkInput{Actor: tester, Code: req1, LinkId: link.Id}); err != nil {
		t.Fatal(err)
	}
	links, _ := uc.ListLinks.Execute(ctx, usecase.ListLinksInput{Code: req1})
	last := w.audit[len(w.audit)-1]
	if len(links) != 0 || last.Action != port.AuditDelete || last.Field != "link" || last.OldValue != sheet {
		t.Errorf("quitar un enlace debe dejar su URL en la auditoría: %+v", last)
	}
}

func TestEntryContract(t *testing.T) {
	uc, w := setup(t)
	if _, err := uc.CheckEntryItem.Execute(ctx, usecase.CheckEntryItemInput{Actor: tester, Code: req1, ItemKey: "ids"}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("solo dev marca el contrato: %v", err)
	}
	if _, err := uc.CheckEntryItem.Execute(ctx, usecase.CheckEntryItemInput{Actor: dev, Code: req1, ItemKey: "inventado"}); !errors.Is(err, usecase.ErrEntryItemNotFound) {
		t.Errorf("ítem inexistente: %v", err)
	}
	item, err := uc.CheckEntryItem.Execute(ctx, usecase.CheckEntryItemInput{Actor: dev, Code: req1, ItemKey: "ids", Note: " listo "})
	if err != nil || !item.Done || item.Note != "listo" {
		t.Fatalf("marcar: %+v, %v", item, err)
	}
	before := len(w.audit)
	if _, err := uc.CheckEntryItem.Execute(ctx, usecase.CheckEntryItemInput{Actor: dev, Code: req1, ItemKey: "ids", Note: "listo"}); err != nil || len(w.audit) != before {
		t.Errorf("marcar lo ya marcado con la misma nota no se audita: %v", err)
	}
	if err := uc.UncheckEntryItem.Execute(ctx, usecase.UncheckEntryItemInput{Actor: dev, Code: req1, ItemKey: "ids"}); err != nil {
		t.Fatal(err)
	}
	last := w.audit[len(w.audit)-1]
	if _, done := w.checks["ids"]; done || last.Field != "entryContract.ids" || last.OldValue != "cumplido" || last.NewValue != "pendiente" {
		t.Errorf("quitar la marca: %+v", last)
	}
	before = len(w.audit)
	if err := uc.UncheckEntryItem.Execute(ctx, usecase.UncheckEntryItemInput{Actor: dev, Code: req1, ItemKey: "ids"}); err != nil || len(w.audit) != before {
		t.Errorf("quitar una marca que no estaba no cambia nada: %v", err)
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	w.risk = &domain.RiskFact{Level: "low"}
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")
	title := "Otro"

	attempts := map[string]func() error{
		"crear": func() error {
			_, err := uc.CreateRequirement.Execute(ctx, usecase.CreateRequirementInput{Actor: manager, Title: "x", SystemKey: "SYPAGO"})
			return err
		},
		"editar": func() error {
			_, err := uc.UpdateRequirement.Execute(ctx, usecase.UpdateRequirementInput{Actor: dev, Code: req1, Title: &title})
			return err
		},
		"transición": func() error { return move(t, uc, tester, "risk_assessed") },
		"enlace": func() error {
			_, err := uc.AddLink.Execute(ctx, usecase.AddLinkInput{Actor: tester, Code: req1, URL: sheet})
			return err
		},
		"contrato": func() error {
			_, err := uc.CheckEntryItem.Execute(ctx, usecase.CheckEntryItemInput{Actor: dev, Code: req1, ItemKey: "ids"})
			return err
		},
	}
	for name, attempt := range attempts {
		if err := attempt(); !axis.IsCode(err, axis.ServiceUnavailable) {
			t.Errorf("%s: esperaba el error de auditoría, obtuve %v", name, err)
		}
	}
	s := w.requirements[req1].Snapshot()
	if len(w.requirements) != 1 || s.Title != "Límite diario" || s.State.Key != "registered" ||
		len(w.links) != 0 || len(w.checks) != 0 || w.sequence != 1 || len(w.audit) != 1 {
		t.Error("si la auditoría falla, nada debe quedar guardado")
	}
}

// Lock no hace nada en memoria: el doble ya ejecuta una transacción a la vez.
func (w *world) Lock(ctx context.Context, code shared.Code) error {
	_, err := w.FindByCode(ctx, code)
	return err
}
