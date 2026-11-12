package usecase_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/regression/application/port"
	"kaoru/internal/feature/regression/application/usecase"
	"kaoru/internal/feature/regression/domain"
	shared "kaoru/internal/shared/domain"
)

// testCase es un TC del catálogo de los dobles.
type testCase struct {
	domain.TestCase
	title    string
	critical bool
}

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	selections map[string]domain.Selection       // por id de PKG
	items      map[string]map[string]domain.Item // por id de PKG y de TC
	candidates []domain.Candidate                // lo que el sistema propone para PKG-2026-0001
	lastRuns   map[string]string                 // resultado del último RUN sobre un BUILD del PKG, por id de TC
	audit      []port.AuditRecord
	auditErr   error
	locked     []string // códigos de PKG bloqueados, en orden
	inTx       bool
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	if w.inTx {
		return fn(ctx) // reutiliza la transacción en curso
	}
	selections, audit := maps.Clone(w.selections), slices.Clone(w.audit)
	items := make(map[string]map[string]domain.Item, len(w.items))
	for id, byTestCase := range w.items {
		items[id] = maps.Clone(byTestCase)
	}
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.selections, w.items, w.audit = selections, items, audit
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

// Los PKG de los dobles: uno de SyPago y uno de SIMF.
var packages = map[string]port.PackageRef{
	"PKG-2026-0001": {Id: "p1", Code: code("PKG-2026-0001"), SystemKey: "SYPAGO"},
	"PKG-2026-0009": {Id: "p9", Code: code("PKG-2026-0009"), SystemKey: "SIMF"},
}

// Los TC de los dobles: cinco activos de SyPago, uno retirado y uno de SIMF.
var testCases = map[string]testCase{
	"TC-2026-0001": {tc("1", "SYPAGO", false), "Rechaza sobre el límite", true},
	"TC-2026-0002": {tc("2", "SYPAGO", false), "Acepta bajo el límite", false},
	"TC-2026-0003": {tc("3", "SYPAGO", false), "Firma el lote", false},
	"TC-2026-0004": {tc("4", "SYPAGO", false), "Concilia el día", false},
	"TC-2026-0005": {tc("5", "SYPAGO", false), "Corte de día", false},
	"TC-2026-0006": {tc("6", "SYPAGO", true), "Caso obsoleto", false},
	"TC-2026-0007": {tc("7", "SIMF", false), "Mensaje MT103", false},
}

func code(raw string) shared.Code {
	c, err := shared.ParseCode(raw)
	if err != nil {
		panic(err)
	}
	return c
}

func tc(n, system string, retired bool) domain.TestCase {
	return domain.TestCase{Ref: domain.TestCaseRef{Id: "t" + n, Code: "TC-2026-000" + n}, SystemKey: system, Retired: retired}
}

func candidate(n string, origin domain.Origin) domain.Candidate {
	return domain.Candidate{TestCase: domain.TestCaseRef{Id: "t" + n, Code: "TC-2026-000" + n}, Origin: origin}
}

func (w *world) FindPackage(_ context.Context, c shared.Code) (port.PackageRef, error) {
	pkg, ok := packages[c.String()]
	if !ok {
		return port.PackageRef{}, notFound
	}
	return pkg, nil
}

func (w *world) LockPackage(ctx context.Context, c shared.Code) (port.PackageRef, error) {
	pkg, err := w.FindPackage(ctx, c)
	if err != nil {
		return port.PackageRef{}, err
	}
	if !w.inTx {
		return port.PackageRef{}, errors.New("LockPackage fuera de una transacción")
	}
	w.locked = append(w.locked, c.String())
	return pkg, nil
}

func (w *world) FindTestCase(_ context.Context, c shared.Code) (domain.TestCase, error) {
	found, ok := testCases[c.String()]
	if !ok {
		return domain.TestCase{}, notFound
	}
	return found.TestCase, nil
}

func (w *world) ListCandidates(_ context.Context, packageId string) ([]domain.Candidate, error) {
	if packageId != "p1" {
		return nil, nil
	}
	return w.candidates, nil
}

// FindSelection arma la selección como la lee la base: con los nombres, el título y el último RUN.
func (w *world) FindSelection(_ context.Context, packageId string) (port.SelectionView, error) {
	selection, ok := w.selections[packageId]
	if !ok {
		return port.SelectionView{}, notFound
	}
	view := port.SelectionView{Selection: selection, ProposedByName: "Usuario " + string(selection.ProposedBy)}
	if selection.Confirmed() {
		view.ConfirmedByName = "Usuario " + string(selection.ConfirmedBy)
	}
	byCode := map[string]domain.Item{}
	for _, item := range w.items[packageId] {
		byCode[item.TestCase.Code] = item
	}
	for _, c := range slices.Sorted(maps.Keys(byCode)) {
		item := byCode[c]
		iv := port.ItemView{Item: item, Title: testCases[c].title, IsCritical: testCases[c].critical}
		if item.DecidedBy != "" {
			iv.DecidedByName = "Usuario " + string(item.DecidedBy)
		}
		if item.Origin == domain.OriginImpact {
			iv.Components = []port.ComponentRef{{Code: "COMP-2026-0001", Name: "api-pagos", Kind: "microservice"}}
		}
		if result := w.lastRuns[item.TestCase.Id]; result != "" {
			iv.LastRun = &port.RunRef{Code: "RUN-2026-000" + item.TestCase.Id[1:], Result: result, BuildCode: "BUILD-2026-0001", ExecutedAt: t0}
		}
		view.Items = append(view.Items, iv)
	}
	return view, nil
}

func (w *world) mustBeLocked(packageId, what string) error {
	for c, pkg := range packages {
		if pkg.Id == packageId && w.inTx && slices.Contains(w.locked, c) {
			return nil
		}
	}
	return errors.New(what + " sin bloquear el PKG en una transacción")
}

func (w *world) SaveSelection(_ context.Context, s domain.Selection) error {
	if err := w.mustBeLocked(s.PackageId, "SaveSelection"); err != nil {
		return err
	}
	w.selections[s.PackageId] = s
	return nil
}

func (w *world) ReplaceItems(_ context.Context, packageId string, items []domain.Item) error {
	if err := w.mustBeLocked(packageId, "ReplaceItems"); err != nil {
		return err
	}
	w.items[packageId] = map[string]domain.Item{}
	for _, item := range items {
		if err := checkItem(item); err != nil {
			return err
		}
		w.items[packageId][item.TestCase.Id] = item
	}
	return nil
}

func (w *world) SaveItem(_ context.Context, packageId string, item domain.Item) error {
	if err := w.mustBeLocked(packageId, "SaveItem"); err != nil {
		return err
	}
	if err := checkItem(item); err != nil {
		return err
	}
	if w.items[packageId] == nil {
		w.items[packageId] = map[string]domain.Item{}
	}
	w.items[packageId][item.TestCase.Id] = item
	return nil
}

// checkItem hace de la restricción de la base: lo que difiere de lo propuesto lleva su criterio.
func checkItem(item domain.Item) error {
	if item.Included != item.Proposed && item.Reason == "" {
		return axis.New(axis.InvalidRequest, "regression_item_check")
	}
	return nil
}

func (w *world) DeleteItem(_ context.Context, packageId, testCaseId string) error {
	if err := w.mustBeLocked(packageId, "DeleteItem"); err != nil {
		return err
	}
	delete(w.items[packageId], testCaseId)
	return nil
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

// fakePermissions: solo qa administra la regresión.
type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	if slices.Contains(roles, "qa") && permission == usecase.PermissionManage {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

var (
	t0  = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	t1  = t0.Add(time.Hour)
	qa  = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	qa2 = shared.Actor{Id: "u5", Roles: []string{"qa"}}
	dev = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	ctx = context.Background()
)

const pkg1 = "PKG-2026-0001"

func setup(t *testing.T) (*usecase.UseCases, *world, *fixedClock) {
	t.Helper()
	w := &world{
		selections: map[string]domain.Selection{}, items: map[string]map[string]domain.Item{}, lastRuns: map[string]string{},
		candidates: []domain.Candidate{candidate("1", domain.OriginCritical), candidate("2", domain.OriginImpact), candidate("3", domain.OriginImpact)},
	}
	clock := &fixedClock{now: t0}
	uc, err := usecase.New(usecase.Deps{Repository: w, Permissions: fakePermissions{}, AuditTrail: w, UnitOfWork: w, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w, clock
}

func propose(t *testing.T, uc *usecase.UseCases) usecase.RegressionOutput {
	t.Helper()
	out, err := uc.ProposeRegression.Execute(ctx, usecase.ProposeRegressionInput{Actor: qa, PackageCode: pkg1})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func adjust(uc *usecase.UseCases, actor shared.Actor, testCase string, included bool, reason string) (usecase.RegressionOutput, error) {
	return uc.AdjustRegressionItem.Execute(ctx, usecase.AdjustRegressionItemInput{
		Actor: actor, PackageCode: pkg1, TestCaseCode: testCase, Included: &included, Reason: reason,
	})
}

func confirm(t *testing.T, uc *usecase.UseCases) {
	t.Helper()
	if _, err := uc.ConfirmRegression.Execute(ctx, usecase.ConfirmRegressionInput{Actor: qa, PackageCode: pkg1}); err != nil {
		t.Fatal(err)
	}
}

// shape resume la selección: cada TC con su origen, si se propuso (P) y si quedó incluido (I).
func shape(out usecase.RegressionOutput) string {
	var parts []string
	for _, item := range out.Items {
		flags := ""
		if item.Proposed {
			flags += "P"
		}
		if item.Included {
			flags += "I"
		}
		parts = append(parts, item.TestCase.Code[len("TC-2026-000"):]+":"+string(item.Origin)+":"+flags)
	}
	return strings.Join(parts, " ")
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

func TestWritesRequireThePermission(t *testing.T) {
	uc, w, _ := setup(t)
	propose(t, uc)
	saved, audited := maps.Clone(w.items["p1"]), len(w.audit)
	included := false
	calls := map[string]func() error{
		"proponer": func() error {
			_, err := uc.ProposeRegression.Execute(ctx, usecase.ProposeRegressionInput{Actor: dev, PackageCode: pkg1})
			return err
		},
		"ajustar": func() error {
			_, err := uc.AdjustRegressionItem.Execute(ctx, usecase.AdjustRegressionItemInput{
				Actor: dev, PackageCode: pkg1, TestCaseCode: "TC-2026-0002", Included: &included, Reason: "No aplica",
			})
			return err
		},
		"confirmar": func() error {
			_, err := uc.ConfirmRegression.Execute(ctx, usecase.ConfirmRegressionInput{Actor: dev, PackageCode: pkg1})
			return err
		},
	}
	for name, call := range calls {
		if err := call(); !axis.IsCode(err, axis.PermissionDenied) {
			t.Errorf("%s sin permiso: %v", name, err)
		}
	}
	if !reflect.DeepEqual(w.items["p1"], saved) || len(w.audit) != audited || w.selections["p1"].Confirmed() {
		t.Error("sin permiso no se guarda ni se audita nada")
	}
}

func TestProposeRegression(t *testing.T) {
	uc, w, clock := setup(t)

	for _, missing := range []string{"PKG-2026-9999", "basura"} {
		_, err := uc.ProposeRegression.Execute(ctx, usecase.ProposeRegressionInput{Actor: qa, PackageCode: missing})
		if !errors.Is(err, usecase.ErrPackageNotFound) {
			t.Errorf("PKG %q: %v", missing, err)
		}
	}

	// Primera propuesta: impacto y críticos, todos incluidos.
	out := propose(t, uc)
	if got, want := shape(out), "1:critical:PI 2:impact:PI 3:impact:PI"; got != want {
		t.Fatalf("propuesta = %s, quiero %s", got, want)
	}
	if out.PackageCode != pkg1 || out.ProposedBy != (usecase.UserRef{Id: "u2", Name: "Usuario u2"}) || !out.ProposedAt.Equal(t0) ||
		out.ConfirmedBy != nil || out.ConfirmedAt != nil {
		t.Errorf("cabecera: %+v", out)
	}
	if out.Summary != (domain.Summary{Proposed: 3, Included: 3, Pending: 3}) {
		t.Errorf("resumen: %+v", out.Summary)
	}
	if first := out.Items[0]; first.Title != "Rechaza sobre el límite" || !first.IsCritical || first.DecidedBy != nil || len(first.Components) != 0 ||
		len(out.Items[1].Components) != 1 {
		t.Errorf("ítems: %+v", out.Items)
	}
	wantAudit := port.AuditRecord{Actor: "u2", PackageId: "p1", Code: code(pkg1), Action: port.AuditCreate, Field: "proposed", NewValue: "3"}
	if len(w.audit) != 1 || w.audit[0] != wantAudit {
		t.Errorf("auditoría de la propuesta: %+v", w.audit)
	}
	if !slices.Contains(w.locked, pkg1) {
		t.Error("proponer debe bloquear el PKG")
	}

	// QA quita uno propuesto y agrega uno a mano.
	if _, err := adjust(uc, qa, "TC-2026-0002", false, "Cubierto por TC-2026-0003"); err != nil {
		t.Fatal(err)
	}
	if _, err := adjust(uc, qa, "TC-2026-0005", true, "Zona frágil"); err != nil {
		t.Fatal(err)
	}

	// Volver a proponer: el 3 ya no se propone, el 4 es nuevo; los ajustes de QA siguen.
	w.candidates = []domain.Candidate{candidate("1", domain.OriginImpact), candidate("2", domain.OriginImpact), candidate("4", domain.OriginCritical)}
	clock.now = t1
	out, err := uc.ProposeRegression.Execute(ctx, usecase.ProposeRegressionInput{Actor: qa2, PackageCode: pkg1})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shape(out), "1:impact:PI 2:impact:P 4:critical:PI 5:manual:I"; got != want {
		t.Errorf("segunda propuesta = %s, quiero %s", got, want)
	}
	if out.ProposedBy.Id != "u5" || !out.ProposedAt.Equal(t1) {
		t.Errorf("quién y cuándo volvió a proponer: %+v", out)
	}
	if kept := out.Items[1]; kept.Reason != "Cubierto por TC-2026-0003" || kept.DecidedBy == nil || kept.DecidedBy.Id != "u2" || !kept.DecidedAt.Equal(t0) {
		t.Errorf("el ajuste de QA debía conservarse: %+v", kept)
	}
	if out.Summary != (domain.Summary{Proposed: 3, Included: 3, Added: 1, Removed: 1, Pending: 3}) {
		t.Errorf("resumen: %+v", out.Summary)
	}
	last := w.audit[len(w.audit)-1]
	if last != (port.AuditRecord{Actor: "u5", PackageId: "p1", Code: code(pkg1), Action: port.AuditUpdate, Field: "proposed", OldValue: "3", NewValue: "3"}) {
		t.Errorf("auditoría de la segunda propuesta: %+v", last)
	}

	// Si la auditoría falla, la propuesta no se guarda.
	saved, header := maps.Clone(w.items["p1"]), w.selections["p1"]
	w.candidates, w.auditErr = nil, errors.New("sin auditoría")
	if _, err := uc.ProposeRegression.Execute(ctx, usecase.ProposeRegressionInput{Actor: qa, PackageCode: pkg1}); err == nil {
		t.Fatal("debía fallar con la auditoría")
	}
	if !reflect.DeepEqual(w.items["p1"], saved) || w.selections["p1"] != header {
		t.Error("nada se guarda si la auditoría falla")
	}
	w.auditErr = nil

	// Confirmada, no se vuelve a proponer.
	confirm(t, uc)
	audited := len(w.audit)
	_, err = uc.ProposeRegression.Execute(ctx, usecase.ProposeRegressionInput{Actor: qa, PackageCode: pkg1})
	if !errors.Is(err, domain.ErrConfirmed) || !axis.IsCode(err, axis.InvalidAction) {
		t.Errorf("proponer con la selección confirmada: %v", err)
	}
	if !reflect.DeepEqual(w.items["p1"], saved) || len(w.audit) != audited {
		t.Error("una propuesta rechazada no cambia nada")
	}
}

func TestProposeFirstTimeLeavesNothingIfAuditFails(t *testing.T) {
	uc, w, _ := setup(t)
	w.auditErr = errors.New("sin auditoría")
	if _, err := uc.ProposeRegression.Execute(ctx, usecase.ProposeRegressionInput{Actor: qa, PackageCode: pkg1}); err == nil {
		t.Fatal("debía fallar con la auditoría")
	}
	if len(w.selections) != 0 || len(w.items["p1"]) != 0 {
		t.Errorf("no debía quedar selección: %+v %+v", w.selections, w.items)
	}
}

func TestAdjustRegressionItem(t *testing.T) {
	uc, w, clock := setup(t)

	// Sin selección no hay nada que ajustar.
	if _, err := adjust(uc, qa, "TC-2026-0002", false, "x"); !errors.Is(err, domain.ErrNotProposed) || !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("ajustar sin proponer: %v", err)
	}
	propose(t, uc)
	audited := len(w.audit)

	// Errores que no guardan nada.
	_, err := uc.AdjustRegressionItem.Execute(ctx, usecase.AdjustRegressionItemInput{Actor: qa, PackageCode: pkg1, TestCaseCode: "TC-2026-0002"})
	if got := fieldsOf(err); !reflect.DeepEqual(got, []string{"included:required"}) {
		t.Errorf("sin included: %v (%v)", got, err)
	}
	_, err = uc.AdjustRegressionItem.Execute(ctx, usecase.AdjustRegressionItemInput{Actor: qa, PackageCode: "PKG-2026-9999", TestCaseCode: "TC-2026-0002", Included: new(bool)})
	if !errors.Is(err, usecase.ErrPackageNotFound) {
		t.Errorf("PKG inexistente: %v", err)
	}
	for _, missing := range []string{"TC-2026-9999", "basura"} {
		if _, err := adjust(uc, qa, missing, true, "x"); !errors.Is(err, usecase.ErrTestCaseNotFound) || !axis.IsCode(err, axis.NotFoundResource) {
			t.Errorf("TC %q: %v", missing, err)
		}
	}
	if _, err := adjust(uc, qa, "TC-2026-0002", false, "  "); !reflect.DeepEqual(fieldsOf(err), []string{"reason:required"}) {
		t.Errorf("quitar un propuesto sin criterio: %v", err)
	}
	if _, err := adjust(uc, qa, "TC-2026-0005", true, ""); !reflect.DeepEqual(fieldsOf(err), []string{"reason:required"}) {
		t.Errorf("agregar uno no propuesto sin criterio: %v", err)
	}
	if _, err := adjust(uc, qa, "TC-2026-0007", true, "x"); !errors.Is(err, domain.ErrOtherSystem) {
		t.Errorf("TC de otro sistema: %v", err)
	}
	if _, err := adjust(uc, qa, "TC-2026-0006", true, "x"); !errors.Is(err, domain.ErrTestCaseRetired) {
		t.Errorf("TC retirado: %v", err)
	}
	if len(w.audit) != audited || len(w.items["p1"]) != 3 {
		t.Fatalf("un ajuste rechazado no guarda ni audita: %+v", w.audit[audited:])
	}

	// Lo que no cambia nada tampoco audita: incluir uno ya incluido, excluir uno que no está.
	out, err := adjust(uc, qa, "TC-2026-0002", true, "")
	if err != nil || shape(out) != "1:critical:PI 2:impact:PI 3:impact:PI" {
		t.Errorf("incluir uno ya incluido: %v %s", err, shape(out))
	}
	if out, err = adjust(uc, qa, "TC-2026-0005", false, ""); err != nil || len(out.Items) != 3 {
		t.Errorf("excluir uno que no está: %v %s", err, shape(out))
	}
	if len(w.audit) != audited {
		t.Errorf("lo que no cambia nada no se audita: %+v", w.audit[audited:])
	}

	// Quitar un TC propuesto, con su criterio.
	clock.now = t1
	out, err = adjust(uc, qa2, "TC-2026-0002", false, "  Cubierto por TC-2026-0003  ")
	if err != nil {
		t.Fatal(err)
	}
	if shape(out) != "1:critical:PI 2:impact:P 3:impact:PI" {
		t.Errorf("tras quitar: %s", shape(out))
	}
	if item := out.Items[1]; item.Reason != "Cubierto por TC-2026-0003" || item.DecidedBy == nil ||
		*item.DecidedBy != (usecase.UserRef{Id: "u5", Name: "Usuario u5"}) || !item.DecidedAt.Equal(t1) {
		t.Errorf("ajuste guardado: %+v", item)
	}
	if out.Summary != (domain.Summary{Proposed: 3, Included: 2, Removed: 1, Pending: 2}) {
		t.Errorf("resumen: %+v", out.Summary)
	}
	want := port.AuditRecord{
		Actor: "u5", PackageId: "p1", Code: code(pkg1), Action: port.AuditDecision, Field: "item",
		OldValue: "TC-2026-0002: included", NewValue: "TC-2026-0002: excluded", Reason: "Cubierto por TC-2026-0003",
	}
	if len(w.audit) != audited+1 || w.audit[audited] != want {
		t.Errorf("auditoría del ajuste: %+v", w.audit[audited:])
	}

	// Agregar uno que no estaba: entra como manual.
	if out, err = adjust(uc, qa, "TC-2026-0005", true, "Zona frágil"); err != nil || shape(out) != "1:critical:PI 2:impact:P 3:impact:PI 5:manual:I" {
		t.Fatalf("agregar manual: %v %s", err, shape(out))
	}
	want = port.AuditRecord{
		Actor: "u2", PackageId: "p1", Code: code(pkg1), Action: port.AuditDecision, Field: "item",
		OldValue: "TC-2026-0005: absent", NewValue: "TC-2026-0005: included", Reason: "Zona frágil",
	}
	if w.audit[len(w.audit)-1] != want {
		t.Errorf("auditoría del agregado: %+v", w.audit[len(w.audit)-1])
	}

	// Volver a incluir el propuesto borra el criterio y el autor.
	if out, err = adjust(uc, qa, "TC-2026-0002", true, "da igual"); err != nil {
		t.Fatal(err)
	}
	if item := out.Items[1]; !item.Included || item.Reason != "" || item.DecidedBy != nil || item.DecidedAt != nil {
		t.Errorf("volver a incluir: %+v", item)
	}
	if last := w.audit[len(w.audit)-1]; last.OldValue != "TC-2026-0002: excluded" || last.NewValue != "TC-2026-0002: included" || last.Reason != "" {
		t.Errorf("auditoría de volver a incluir: %+v", last)
	}

	// Quitar el manual lo borra de la selección.
	if out, err = adjust(uc, qa, "TC-2026-0005", false, ""); err != nil || shape(out) != "1:critical:PI 2:impact:PI 3:impact:PI" {
		t.Fatalf("quitar manual: %v %s", err, shape(out))
	}
	if last := w.audit[len(w.audit)-1]; last.OldValue != "TC-2026-0005: included" || last.NewValue != "TC-2026-0005: absent" {
		t.Errorf("auditoría de quitar el manual: %+v", last)
	}

	// Si la auditoría falla, el ajuste no se guarda.
	saved := maps.Clone(w.items["p1"])
	w.auditErr = errors.New("sin auditoría")
	if _, err := adjust(uc, qa, "TC-2026-0003", false, "No aplica"); err == nil {
		t.Fatal("debía fallar con la auditoría")
	}
	if !reflect.DeepEqual(w.items["p1"], saved) {
		t.Error("nada se guarda si la auditoría falla")
	}
	w.auditErr = nil

	// Confirmada, la selección no cambia.
	confirm(t, uc)
	if _, err := adjust(uc, qa, "TC-2026-0003", false, "No aplica"); !errors.Is(err, domain.ErrConfirmed) || !axis.IsCode(err, axis.InvalidAction) {
		t.Errorf("ajustar con la selección confirmada: %v", err)
	}
	if !reflect.DeepEqual(w.items["p1"], saved) {
		t.Error("un ajuste rechazado no cambia nada")
	}
}

func TestConfirmAndReopenRegression(t *testing.T) {
	uc, w, clock := setup(t)
	reopen := func(pkg string) error {
		return uc.ReopenRegression.Execute(ctx, usecase.ReopenRegressionInput{ActorId: "u3", PackageCode: pkg, Reason: "Cambió el contenido del PKG."})
	}

	// Sin selección: confirmar no se puede; reabrir no hace nada.
	_, err := uc.ConfirmRegression.Execute(ctx, usecase.ConfirmRegressionInput{Actor: qa, PackageCode: pkg1})
	if !errors.Is(err, domain.ErrNotProposed) {
		t.Errorf("confirmar sin proponer: %v", err)
	}
	if _, err := uc.ConfirmRegression.Execute(ctx, usecase.ConfirmRegressionInput{Actor: qa, PackageCode: "PKG-2026-9999"}); !errors.Is(err, usecase.ErrPackageNotFound) {
		t.Errorf("confirmar un PKG inexistente: %v", err)
	}
	if err := reopen(pkg1); err != nil || len(w.audit) != 0 {
		t.Errorf("reabrir sin selección: %v %+v", err, w.audit)
	}
	if err := reopen("PKG-2026-9999"); !errors.Is(err, usecase.ErrPackageNotFound) {
		t.Errorf("reabrir un PKG inexistente: %v", err)
	}

	propose(t, uc)
	if err := reopen(pkg1); err != nil || len(w.audit) != 1 {
		t.Errorf("reabrir sin confirmación no audita: %v %+v", err, w.audit)
	}

	// Si la auditoría falla, la confirmación no se guarda.
	w.auditErr = errors.New("sin auditoría")
	if _, err := uc.ConfirmRegression.Execute(ctx, usecase.ConfirmRegressionInput{Actor: qa, PackageCode: pkg1}); err == nil || w.selections["p1"].Confirmed() {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	clock.now = t1
	out, err := uc.ConfirmRegression.Execute(ctx, usecase.ConfirmRegressionInput{Actor: qa2, PackageCode: pkg1})
	if err != nil {
		t.Fatal(err)
	}
	if out.ConfirmedBy == nil || *out.ConfirmedBy != (usecase.UserRef{Id: "u5", Name: "Usuario u5"}) || !out.ConfirmedAt.Equal(t1) || len(out.Items) != 3 {
		t.Errorf("confirmada: %+v", out)
	}
	want := port.AuditRecord{Actor: "u5", PackageId: "p1", Code: code(pkg1), Action: port.AuditDecision, Field: "confirmed", OldValue: "false", NewValue: "true"}
	if w.audit[len(w.audit)-1] != want {
		t.Errorf("auditoría de la confirmación: %+v", w.audit[len(w.audit)-1])
	}
	audited := len(w.audit)
	if _, err := uc.ConfirmRegression.Execute(ctx, usecase.ConfirmRegressionInput{Actor: qa, PackageCode: pkg1}); !errors.Is(err, domain.ErrAlreadyConfirmed) ||
		!axis.IsCode(err, axis.InvalidAction) || len(w.audit) != audited {
		t.Errorf("confirmar dos veces: %v", err)
	}

	// Reabrir: si la auditoría falla, la confirmación sigue; si no, se quita y queda el motivo.
	w.auditErr = errors.New("sin auditoría")
	if err := reopen(pkg1); err == nil || !w.selections["p1"].Confirmed() {
		t.Fatalf("reabrir con la auditoría caída: %v", err)
	}
	w.auditErr = nil
	if err := reopen(pkg1); err != nil || w.selections["p1"].Confirmed() {
		t.Fatalf("reabrir: %v %+v", err, w.selections["p1"])
	}
	want = port.AuditRecord{
		Actor: "u3", PackageId: "p1", Code: code(pkg1), Action: port.AuditUpdate, Field: "confirmed",
		OldValue: "true", NewValue: "false", Reason: "Cambió el contenido del PKG.",
	}
	if len(w.audit) != audited+1 || w.audit[audited] != want {
		t.Errorf("auditoría de reabrir: %+v", w.audit[audited:])
	}
	if len(w.items["p1"]) != 3 {
		t.Error("reabrir no toca los TC de la selección")
	}
	// Reabierta, vuelve a admitir ajustes.
	if _, err := adjust(uc, qa, "TC-2026-0003", false, "No aplica"); err != nil {
		t.Errorf("ajustar después de reabrir: %v", err)
	}
}

func TestGetRegressionAndFacts(t *testing.T) {
	uc, w, _ := setup(t)
	facts := func(pkg string) (usecase.RegressionFacts, error) {
		return uc.GetRegressionFacts.Execute(ctx, usecase.GetRegressionFactsInput{PackageCode: pkg})
	}

	for _, missing := range []string{"PKG-2026-9999", "basura"} {
		if _, err := uc.GetRegression.Execute(ctx, usecase.GetRegressionInput{PackageCode: missing}); !errors.Is(err, usecase.ErrPackageNotFound) {
			t.Errorf("leer la regresión del PKG %q: %v", missing, err)
		}
		if _, err := facts(missing); !errors.Is(err, usecase.ErrPackageNotFound) {
			t.Errorf("hechos del PKG %q: %v", missing, err)
		}
	}
	if _, err := uc.GetRegression.Execute(ctx, usecase.GetRegressionInput{PackageCode: pkg1}); !errors.Is(err, domain.ErrNotProposed) ||
		!axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("leer sin proponer: %v", err)
	}
	if got, err := facts(pkg1); err != nil || got.Exists || got.Confirmed || got.Unpassed != nil {
		t.Errorf("hechos sin proponer: %+v %v", got, err)
	}

	propose(t, uc)
	if _, err := adjust(uc, qa, "TC-2026-0003", false, "No aplica"); err != nil {
		t.Fatal(err)
	}
	if _, err := adjust(uc, qa, "TC-2026-0005", true, "Zona frágil"); err != nil {
		t.Fatal(err)
	}
	// El 1 aprobó, el 2 falló, el 3 (quitado) falló y el 5 no se ejecutó.
	w.lastRuns = map[string]string{"t1": "passed", "t2": "failed", "t3": "failed"}

	out, err := uc.GetRegression.Execute(ctx, usecase.GetRegressionInput{PackageCode: pkg1})
	if err != nil {
		t.Fatal(err)
	}
	if shape(out) != "1:critical:PI 2:impact:PI 3:impact:P 5:manual:I" {
		t.Errorf("selección: %s", shape(out))
	}
	if out.Summary != (domain.Summary{Proposed: 3, Included: 3, Added: 1, Removed: 1, Passed: 1, Failed: 1, Pending: 1}) {
		t.Errorf("resumen: %+v", out.Summary)
	}
	if run := out.Items[0].LastRun; run == nil || run.Result != "passed" || run.BuildCode != "BUILD-2026-0001" || out.Items[3].LastRun != nil {
		t.Errorf("último RUN: %+v", out.Items)
	}

	got, err := facts(pkg1)
	if err != nil || !got.Exists || got.Confirmed || !reflect.DeepEqual(got.Unpassed, []string{"TC-2026-0002", "TC-2026-0005"}) {
		t.Errorf("hechos: %+v %v", got, err)
	}
	confirm(t, uc)
	w.lastRuns["t2"], w.lastRuns["t5"] = "passed", "passed"
	if got, err = facts(pkg1); err != nil || !got.Exists || !got.Confirmed || got.Unpassed != nil {
		t.Errorf("hechos con todo aprobado: %+v %v", got, err)
	}
	// La selección de un PKG no es la de otro.
	if got, err = facts("PKG-2026-0009"); err != nil || got.Exists {
		t.Errorf("hechos de otro PKG: %+v %v", got, err)
	}
}
