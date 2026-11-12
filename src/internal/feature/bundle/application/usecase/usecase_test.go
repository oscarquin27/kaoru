package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/application/usecase"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	packages     map[string]domain.Snapshot    // por código
	requirements map[string]domain.Requirement // el feature requirement, por código
	sequence     int
	audit        []port.AuditRecord
	auditErr     error
	locked       []string // códigos de PKG bloqueados, en orden
	inTx         bool

	moved     []string // REQ pasados a «En paquete», con el actor: "REQ-…/u3"
	reopened  []string // avisos a regression: "PKG-…/motivo"
	reopenErr error

	// Hechos de otros features.
	regression domain.RegressionFact
	defects    map[string][]domain.DefectFact // por código de REQ
	deployed   bool

	// rules reemplaza las reglas de una transición: "desde>hasta".
	rules map[string][]domain.Rule
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	packages, requirements := maps.Clone(w.packages), maps.Clone(w.requirements)
	sequence, audit, moved, reopened := w.sequence, slices.Clone(w.audit), slices.Clone(w.moved), slices.Clone(w.reopened)
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.packages, w.requirements, w.sequence, w.audit, w.moved, w.reopened = packages, requirements, sequence, audit, moved, reopened
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

// --- Repository ---

func (w *world) NextCode(_ context.Context, year int) (shared.Code, error) {
	w.sequence++
	return shared.NewCode(shared.PrefixPackage, year, w.sequence), nil
}

func (w *world) FindSystem(_ context.Context, key string) (domain.System, error) {
	systems := map[string]domain.System{
		"SYPAGO": {Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"},
		"SIMF":   {Key: "SIMF", Name: "SIMF", DefaultRoute: "high"},
	}
	system, ok := systems[key]
	if !ok {
		return domain.System{}, notFound
	}
	return system, nil
}

func (w *world) Insert(_ context.Context, pkg domain.Package) error {
	if !w.inTx {
		return errors.New("Insert fuera de una transacción")
	}
	w.packages[pkg.Snapshot().Code.String()] = pkg.Snapshot()
	return nil
}

func (w *world) mustBeLocked(code, what string) error {
	if !w.inTx || !slices.Contains(w.locked, code) {
		return errors.New(what + " sin bloquear el PKG en una transacción")
	}
	return nil
}

// Update guarda la cabecera; el contenido solo cambia por AddBuild, RemoveBuild y sus pares.
func (w *world) Update(_ context.Context, pkg domain.Package) error {
	s := pkg.Snapshot()
	if err := w.mustBeLocked(s.Code.String(), "Update"); err != nil {
		return err
	}
	stored := w.packages[s.Code.String()]
	stored.Title, stored.Description, stored.State, stored.UpdatedAt = s.Title, s.Description, s.State, s.UpdatedAt
	w.packages[s.Code.String()] = stored
	return nil
}

func (w *world) Lock(_ context.Context, code shared.Code) error {
	if _, ok := w.packages[code.String()]; !ok {
		return notFound
	}
	if !w.inTx {
		return errors.New("Lock fuera de una transacción")
	}
	w.locked = append(w.locked, code.String())
	return nil
}

// read arma el PKG como lo lee la base: con el nombre de quien lo creó y el estado actual de sus REQ.
func (w *world) read(s domain.Snapshot) domain.Package {
	s.CreatedBy.Name = "Usuario " + string(s.CreatedBy.Id)
	s.Builds = slices.Clone(s.Builds)
	s.Requirements = slices.Clone(s.Requirements)
	for i, r := range s.Requirements {
		s.Requirements[i] = w.requirements[r.Code].Ref
	}
	return domain.Rehydrate(s)
}

func (w *world) FindByCode(_ context.Context, code shared.Code) (domain.Package, error) {
	s, ok := w.packages[code.String()]
	if !ok {
		return domain.Package{}, notFound
	}
	return w.read(s), nil
}

func (w *world) List(_ context.Context, f port.Filter) ([]domain.Package, int, error) {
	var out []domain.Package
	for _, code := range slices.Backward(slices.Sorted(maps.Keys(w.packages))) {
		s := w.packages[code]
		if (f.SystemKey != "" && s.System.Key != f.SystemKey) || (f.StateKey != "" && s.State.Key != f.StateKey) ||
			(f.Query != "" && !strings.Contains(s.Title, f.Query)) {
			continue
		}
		out = append(out, w.read(s))
	}
	return out, len(out), nil
}

func (w *world) content(packageId, what string, change func(s *domain.Snapshot)) error {
	for code, s := range w.packages {
		if s.Id != packageId {
			continue
		}
		if err := w.mustBeLocked(code, what); err != nil {
			return err
		}
		change(&s)
		w.packages[code] = s
		return nil
	}
	return notFound
}

func (w *world) AddBuild(_ context.Context, packageId, buildId string, _ shared.UserId, _ time.Time) error {
	return w.content(packageId, "AddBuild", func(s *domain.Snapshot) {
		for _, b := range builds {
			if b.Ref.Id == buildId {
				s.Builds = append(slices.Clone(s.Builds), b.Ref)
			}
		}
		slices.SortFunc(s.Builds, func(a, b domain.BuildRef) int { return strings.Compare(a.Code, b.Code) })
	})
}

func (w *world) RemoveBuild(_ context.Context, packageId, buildId string) error {
	return w.content(packageId, "RemoveBuild", func(s *domain.Snapshot) {
		s.Builds = slices.DeleteFunc(slices.Clone(s.Builds), func(b domain.BuildRef) bool { return b.Id == buildId })
	})
}

func (w *world) AddRequirement(_ context.Context, packageId, requirementId string, _ shared.UserId, _ time.Time) error {
	return w.content(packageId, "AddRequirement", func(s *domain.Snapshot) {
		for _, r := range w.requirements {
			if r.Ref.Id == requirementId {
				s.Requirements = append(slices.Clone(s.Requirements), r.Ref)
			}
		}
		slices.SortFunc(s.Requirements, func(a, b domain.RequirementRef) int { return strings.Compare(a.Code, b.Code) })
	})
}

func (w *world) RemoveRequirement(_ context.Context, packageId, requirementId string) error {
	return w.content(packageId, "RemoveRequirement", func(s *domain.Snapshot) {
		s.Requirements = slices.DeleteFunc(slices.Clone(s.Requirements), func(r domain.RequirementRef) bool { return r.Id == requirementId })
	})
}

// --- Builds ---

// Los BUILD del catálogo: dos de SyPago y uno de SIMF.
var builds = map[string]domain.Build{
	"BUILD-2026-0001": {Ref: domain.BuildRef{Id: "b1", Code: "BUILD-2026-0001", Version: "2.14.0", Checksum: strings.Repeat("a", 64)}, SystemKey: "SYPAGO"},
	"BUILD-2026-0002": {Ref: domain.BuildRef{Id: "b2", Code: "BUILD-2026-0002", Version: "2.14.1", Checksum: strings.Repeat("b", 64)}, SystemKey: "SYPAGO"},
	"BUILD-2026-0009": {Ref: domain.BuildRef{Id: "b9", Code: "BUILD-2026-0009", Version: "4.2.0", Checksum: strings.Repeat("d", 64)}, SystemKey: "SIMF"},
}

type fakeBuilds struct{}

func (fakeBuilds) Find(_ context.Context, code string) (domain.Build, error) {
	build, ok := builds[code]
	if !ok {
		return domain.Build{}, notFound
	}
	return build, nil
}

// --- Requirements ---

func requirement(n, system, state string) domain.Requirement {
	names := map[string]string{"approved_qa": "Aprobado QA", "in_package": "En paquete", "in_testing": "En pruebas"}
	return domain.Requirement{
		Ref: domain.RequirementRef{
			Id: "r" + n, Code: "REQ-2026-000" + n, Title: "Requerimiento " + n, State: domain.RequirementState{Key: state, Name: names[state]},
		},
		SystemKey: system,
	}
}

func (w *world) Find(_ context.Context, code string) (domain.Requirement, error) {
	r, ok := w.requirements[code]
	if !ok {
		return domain.Requirement{}, notFound
	}
	return r, nil
}

// MoveToPackage hace de la transición del REQ: exige su permiso (req.package, que tiene dev) y cambia su estado.
func (w *world) MoveToPackage(_ context.Context, actor shared.Actor, code string) error {
	if !w.inTx {
		return errors.New("MoveToPackage fuera de una transacción")
	}
	if !slices.Contains(actor.Roles, "dev") {
		return axis.New(axis.PermissionDenied, "Sin permiso para pasar el REQ a «En paquete».")
	}
	r := w.requirements[code]
	r.Ref.State = domain.RequirementState{Key: "in_package", Name: "En paquete"}
	w.requirements[code] = r
	w.moved = append(w.moved, code+"/"+string(actor.Id))
	return nil
}

// --- Regression ---

func (w *world) Reopen(_ context.Context, actor shared.UserId, code shared.Code, reason string) error {
	if w.reopenErr != nil {
		return w.reopenErr
	}
	if !w.inTx {
		return errors.New("Reopen fuera de una transacción")
	}
	w.reopened = append(w.reopened, fmt.Sprintf("%s/%s/%s", code, actor, reason))
	return nil
}

// --- Facts ---

type fakeFacts struct{ w *world }

func (f fakeFacts) Regression(context.Context, shared.Code) (domain.RegressionFact, error) {
	return f.w.regression, nil
}

func (f fakeFacts) OpenDefects(_ context.Context, requirementCodes []string) ([]domain.DefectFact, error) {
	var out []domain.DefectFact
	for _, code := range requirementCodes {
		out = append(out, f.w.defects[code]...)
	}
	return out, nil
}

func (f fakeFacts) DeployedToProduction(context.Context, shared.Code) (bool, error) {
	return f.w.deployed, nil
}

// --- AuditTrail ---

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

// --- Workflow ---

// fakeWorkflow reproduce el flujo sembrado del PKG, con sus permisos y sus reglas.
type fakeWorkflow struct{ w *world }

var states = map[string]domain.State{
	"in_assembly":             {Id: "s1", Key: "in_assembly", Name: "En armado", IsInitial: true},
	"in_testing":              {Id: "s2", Key: "in_testing", Name: "En pruebas"},
	"ready_to_release":        {Id: "s3", Key: "ready_to_release", Name: "Listo para liberar"},
	"approved_for_production": {Id: "s4", Key: "approved_for_production", Name: "Aprobado para producción"},
	"deployed":                {Id: "s5", Key: "deployed", Name: "Desplegado"},
	"verified":                {Id: "s6", Key: "verified", Name: "Verificado"},
}

type flowStep struct {
	from, to, permission string
	requiresReason       bool
	rules                []domain.Rule
}

func blocking(key string) domain.Rule { return domain.Rule{Key: key, Blocks: true} }

var flow = []flowStep{
	{"in_assembly", "in_testing", "pkg.start_testing", false, []domain.Rule{blocking(domain.RuleHasContent), blocking(domain.RuleRegressionConfirmed)}},
	{"in_testing", "in_assembly", "pkg.reopen", true, nil},
	{"in_testing", "ready_to_release", "pkg.mark_ready", false, []domain.Rule{
		blocking(domain.RuleRegressionPassed), {Key: domain.RuleNoOpenDefect, Params: map[string]any{"severity": "high"}, Blocks: true},
	}},
	{"ready_to_release", "approved_for_production", "pkg.approve_production", false, nil},
	{"approved_for_production", "deployed", "pkg.deploy", false, []domain.Rule{blocking(domain.RuleDeployedToProduction)}},
	{"deployed", "verified", "pkg.verify", false, nil},
}

var granted = map[string][]string{
	"dev":            {usecase.PermissionBuild},
	"builder":        {usecase.PermissionBuild}, // arma paquetes pero no puede pasar un REQ a «En paquete»
	"qa":             {"pkg.start_testing", "pkg.reopen", "pkg.mark_ready", "pkg.verify"},
	"change-manager": {"pkg.approve_production", "pkg.deploy"},
}

func (fakeWorkflow) Require(_ context.Context, roles []string, permission string) error {
	for _, role := range roles {
		if slices.Contains(granted[role], permission) {
			return nil
		}
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

func (fakeWorkflow) InitialState(context.Context) (domain.State, error) {
	return states["in_assembly"], nil
}

func (f fakeWorkflow) transition(step flowStep) port.Transition {
	rules := step.rules
	if override, ok := f.w.rules[step.from+">"+step.to]; ok {
		rules = override
	}
	return port.Transition{To: states[step.to], RequiresReason: step.requiresReason, Rules: rules}
}

func (f fakeWorkflow) ResolveTransition(ctx context.Context, from, to string, roles []string) (port.Transition, error) {
	for _, step := range flow {
		if step.from != from || step.to != to {
			continue
		}
		if err := f.Require(ctx, roles, step.permission); err != nil {
			return port.Transition{}, err
		}
		return f.transition(step), nil
	}
	return port.Transition{}, axis.New(axis.InvalidAction, "El flujo no permite ese paso.")
}

func (f fakeWorkflow) Outgoing(_ context.Context, from string) ([]port.Transition, error) {
	var out []port.Transition
	for _, step := range flow {
		if step.from == from {
			out = append(out, f.transition(step))
		}
	}
	return out, nil
}

// --- Arranque ---

type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	t0      = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	t1      = t0.Add(time.Hour)
	dev     = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	builder = shared.Actor{Id: "u7", Roles: []string{"builder"}}
	qa      = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	ctx     = context.Background()
)

func setup(t *testing.T) (*usecase.UseCases, *world, *fixedClock) {
	t.Helper()
	w := &world{
		packages: map[string]domain.Snapshot{},
		requirements: map[string]domain.Requirement{
			"REQ-2026-0001": requirement("1", "SYPAGO", "approved_qa"),
			"REQ-2026-0002": requirement("2", "SYPAGO", "in_package"),
			"REQ-2026-0003": requirement("3", "SYPAGO", "in_testing"),
			"REQ-2026-0009": requirement("9", "SIMF", "approved_qa"),
		},
		defects: map[string][]domain.DefectFact{},
		rules:   map[string][]domain.Rule{},
	}
	clock := &fixedClock{now: t0}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Workflow: fakeWorkflow{w}, Builds: fakeBuilds{}, Requirements: w, Regression: w, Facts: fakeFacts{w},
		AuditTrail: w, UnitOfWork: w, Clock: clock, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w, clock
}

// create crea un PKG de SyPago como dev y devuelve su código.
func create(t *testing.T, uc *usecase.UseCases) string {
	t.Helper()
	out, err := uc.CreatePackage.Execute(ctx, usecase.CreatePackageInput{Actor: dev, Title: "Paquete de noviembre", SystemKey: "SYPAGO"})
	if err != nil {
		t.Fatal(err)
	}
	return out.Package.Code.String()
}

func addBuild(uc *usecase.UseCases, actor shared.Actor, pkg, build string) (usecase.PackageOutput, error) {
	return uc.AddPackageBuild.Execute(ctx, usecase.AddPackageBuildInput{Actor: actor, Code: pkg, BuildCode: build})
}

func removeBuild(uc *usecase.UseCases, actor shared.Actor, pkg, build string) (usecase.PackageOutput, error) {
	return uc.RemovePackageBuild.Execute(ctx, usecase.RemovePackageBuildInput{Actor: actor, Code: pkg, BuildCode: build})
}

func addRequirement(uc *usecase.UseCases, actor shared.Actor, pkg, req string) (usecase.PackageOutput, error) {
	return uc.AddPackageRequirement.Execute(ctx, usecase.AddPackageRequirementInput{Actor: actor, Code: pkg, RequirementCode: req})
}

func removeRequirement(uc *usecase.UseCases, actor shared.Actor, pkg, req string) (usecase.PackageOutput, error) {
	return uc.RemovePackageRequirement.Execute(ctx, usecase.RemovePackageRequirementInput{Actor: actor, Code: pkg, RequirementCode: req})
}

func move(uc *usecase.UseCases, actor shared.Actor, pkg, to, reason string) (usecase.PackageOutput, error) {
	return uc.TransitionPackage.Execute(ctx, usecase.TransitionPackageInput{Actor: actor, Code: pkg, To: to, Reason: reason})
}

// filled crea un PKG con un BUILD y un REQ y lo deja listo para pasar a pruebas.
func filled(t *testing.T, uc *usecase.UseCases, w *world) string {
	t.Helper()
	pkg := create(t, uc)
	if _, err := addBuild(uc, dev, pkg, "BUILD-2026-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := addRequirement(uc, dev, pkg, "REQ-2026-0001"); err != nil {
		t.Fatal(err)
	}
	w.regression = domain.RegressionFact{Exists: true, Confirmed: true}
	return pkg
}

// content resume el contenido de un PKG: sus BUILD y sus REQ con su estado.
func content(out usecase.PackageOutput) string {
	var parts []string
	for _, b := range out.Package.Builds {
		parts = append(parts, b.Code)
	}
	for _, r := range out.Package.Requirements {
		parts = append(parts, r.Code+"="+r.State.Key)
	}
	return strings.Join(parts, " ")
}

// missing resume «qué falta»: por transición, las reglas que no se cumplen (con ! si bloquean).
func missing(out usecase.PackageOutput) string {
	var parts []string
	for _, m := range out.Missing {
		var keys []string
		for _, c := range m.Conditions {
			key := c.RuleKey
			if c.Blocks {
				key += "!"
			}
			keys = append(keys, key)
		}
		parts = append(parts, m.To.Key+"["+strings.Join(keys, ",")+"]")
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

func ptr[T any](v T) *T { return &v }

func code(raw string) shared.Code {
	c, err := shared.ParseCode(raw)
	if err != nil {
		panic(err)
	}
	return c
}

// --- Pruebas ---

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestCreatePackage(t *testing.T) {
	uc, w, _ := setup(t)
	in := usecase.CreatePackageInput{Actor: dev, Title: "  Paquete de noviembre  ", Description: " Límites ", SystemKey: "SYPAGO"}

	denied := in
	denied.Actor = qa
	if _, err := uc.CreatePackage.Execute(ctx, denied); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("crear sin pkg.build: %v", err)
	}
	for name, tc := range map[string]struct {
		mutate func(in *usecase.CreatePackageInput)
		want   string
	}{
		"sin sistema":         {func(in *usecase.CreatePackageInput) { in.SystemKey = "" }, "system:required"},
		"sistema desconocido": {func(in *usecase.CreatePackageInput) { in.SystemKey = "OTRO" }, "system:invalid"},
		"sin título":          {func(in *usecase.CreatePackageInput) { in.Title = " " }, "title:required"},
		"título largo":        {func(in *usecase.CreatePackageInput) { in.Title = strings.Repeat("x", 201) }, "title:invalid"},
	} {
		bad := in
		tc.mutate(&bad)
		if _, err := uc.CreatePackage.Execute(ctx, bad); !reflect.DeepEqual(fieldsOf(err), []string{tc.want}) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(w.packages) != 0 || len(w.audit) != 0 || w.sequence != 0 {
		t.Fatalf("un alta rechazada no guarda, no audita ni gasta un código: %+v", w.packages)
	}

	w.auditErr = errors.New("sin auditoría")
	if _, err := uc.CreatePackage.Execute(ctx, in); err == nil || len(w.packages) != 0 || w.sequence != 0 {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	out, err := uc.CreatePackage.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	s := out.Package
	if s.Code.String() != "PKG-2026-0001" || s.Title != "Paquete de noviembre" || s.Description != "Límites" || s.System.Key != "SYPAGO" ||
		s.State != states["in_assembly"] || s.CreatedBy != (domain.UserRef{Id: "u3", Name: "Usuario u3"}) || !s.CreatedAt.Equal(t0) ||
		len(s.Builds) != 0 || len(s.Requirements) != 0 {
		t.Errorf("PKG creado: %+v", s)
	}
	if got, want := missing(out), "in_testing[package_has_content!,regression_confirmed!]"; got != want {
		t.Errorf("qué falta = %s, quiero %s", got, want)
	}
	want := port.AuditRecord{Actor: "u3", PackageId: s.Id, Code: s.Code, Action: port.AuditCreate, NewValue: "Paquete de noviembre"}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Errorf("auditoría: %+v", w.audit)
	}
}

func TestGetFindAndListPackages(t *testing.T) {
	uc, w, _ := setup(t)
	first := filled(t, uc, w)
	second, err := uc.CreatePackage.Execute(ctx, usecase.CreatePackageInput{Actor: dev, Title: "Mensajería 4.2", SystemKey: "SIMF"})
	if err != nil {
		t.Fatal(err)
	}

	for _, missing := range []string{"PKG-2026-9999", "basura", ""} {
		if _, err := uc.GetPackage.Execute(ctx, usecase.GetPackageInput{Code: missing}); !errors.Is(err, usecase.ErrPackageNotFound) {
			t.Errorf("ficha de %q: %v", missing, err)
		}
		if _, err := uc.FindPackage.Execute(ctx, usecase.FindPackageInput{Code: missing}); !errors.Is(err, usecase.ErrPackageNotFound) {
			t.Errorf("buscar %q: %v", missing, err)
		}
	}

	// La ficha trae el contenido, con el estado actual de cada REQ, y qué falta.
	out, err := uc.GetPackage.Execute(ctx, usecase.GetPackageInput{Code: first})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := content(out), "BUILD-2026-0001 REQ-2026-0001=in_package"; got != want {
		t.Errorf("contenido = %s, quiero %s", got, want)
	}
	if got, want := missing(out), "in_testing[]"; got != want {
		t.Errorf("qué falta = %s, quiero %s", got, want)
	}

	ref, err := uc.FindPackage.Execute(ctx, usecase.FindPackageInput{Code: first})
	if err != nil || ref != (usecase.PackageRef{Id: out.Package.Id, Code: first, Title: "Paquete de noviembre", SystemKey: "SYPAGO", StateKey: "in_assembly"}) {
		t.Errorf("referencia: %+v %v", ref, err)
	}

	// La lista trae el contenido pero no calcula qué falta.
	page, err := uc.ListPackages.Execute(ctx, usecase.ListPackagesInput{Page: 1, Size: 25})
	if err != nil || page.Total != 2 || page.Packages[0].Package.Code != second.Package.Code || page.Packages[1].Package.Code.String() != first {
		t.Fatalf("lista: %+v %v", page, err)
	}
	if content(page.Packages[1]) != "BUILD-2026-0001 REQ-2026-0001=in_package" || page.Packages[0].Missing != nil || page.Packages[1].Missing != nil {
		t.Errorf("lista: contenido sí, qué falta no: %+v", page.Packages)
	}
	page, err = uc.ListPackages.Execute(ctx, usecase.ListPackagesInput{SystemKey: "SIMF", StateKey: "in_assembly", Query: "Mensajería", Page: 1, Size: 25})
	if err != nil || page.Total != 1 || page.Packages[0].Package.Code != second.Package.Code {
		t.Errorf("lista filtrada: %+v %v", page, err)
	}
}

func TestUpdatePackage(t *testing.T) {
	uc, w, clock := setup(t)
	pkg := create(t, uc)
	audited := len(w.audit)
	update := func(actor shared.Actor, code string, edit domain.Edit) (usecase.PackageOutput, error) {
		return uc.UpdatePackage.Execute(ctx, usecase.UpdatePackageInput{Actor: actor, Code: code, Edit: edit})
	}
	clock.now = t1

	if _, err := update(qa, pkg, domain.Edit{Title: ptr("Otro")}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("editar sin pkg.build: %v", err)
	}
	if _, err := update(dev, "PKG-2026-9999", domain.Edit{Title: ptr("Otro")}); !errors.Is(err, usecase.ErrPackageNotFound) {
		t.Errorf("editar un PKG inexistente: %v", err)
	}
	if _, err := update(dev, pkg, domain.Edit{Title: ptr(" ")}); !reflect.DeepEqual(fieldsOf(err), []string{"title:required"}) {
		t.Errorf("título vacío: %v", err)
	}
	out, err := update(dev, pkg, domain.Edit{Title: ptr(" Paquete de noviembre ")})
	if err != nil || !out.Package.UpdatedAt.Equal(t0) || len(w.audit) != audited {
		t.Errorf("editar sin cambios no guarda ni audita: %v %+v", err, out.Package)
	}

	w.auditErr = errors.New("sin auditoría")
	if _, err := update(dev, pkg, domain.Edit{Title: ptr("Otro")}); err == nil || w.packages[pkg].Title != "Paquete de noviembre" {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	out, err = update(dev, pkg, domain.Edit{Title: ptr("Paquete de diciembre"), Description: ptr("Con firma")})
	if err != nil {
		t.Fatal(err)
	}
	if s := out.Package; s.Title != "Paquete de diciembre" || s.Description != "Con firma" || !s.UpdatedAt.Equal(t1) || s.System.Key != "SYPAGO" ||
		len(out.Missing) != 1 {
		t.Errorf("PKG editado: %+v", out)
	}
	want := []port.AuditRecord{
		{Actor: "u3", PackageId: out.Package.Id, Code: code(pkg), Action: port.AuditUpdate, Field: "title", OldValue: "Paquete de noviembre", NewValue: "Paquete de diciembre"},
		{Actor: "u3", PackageId: out.Package.Id, Code: code(pkg), Action: port.AuditUpdate, Field: "description", NewValue: "Con firma"},
	}
	if !reflect.DeepEqual(w.audit[audited:], want) {
		t.Errorf("auditoría: %+v", w.audit[audited:])
	}
	if !slices.Contains(w.locked, pkg) {
		t.Error("editar debe bloquear el PKG")
	}
}

func TestPackageBuilds(t *testing.T) {
	uc, w, clock := setup(t)
	pkg := create(t, uc)
	audited := len(w.audit)
	clock.now = t1

	// Errores que no guardan nada.
	if _, err := addBuild(uc, qa, pkg, "BUILD-2026-0001"); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("agregar sin pkg.build: %v", err)
	}
	if _, err := removeBuild(uc, qa, pkg, "BUILD-2026-0001"); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("quitar sin pkg.build: %v", err)
	}
	if _, err := addBuild(uc, dev, "PKG-2026-9999", "BUILD-2026-0001"); !errors.Is(err, usecase.ErrPackageNotFound) {
		t.Errorf("PKG inexistente: %v", err)
	}
	if _, err := addBuild(uc, dev, pkg, "BUILD-2026-9999"); !errors.Is(err, usecase.ErrBuildNotFound) || !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("BUILD inexistente: %v", err)
	}
	if _, err := removeBuild(uc, dev, pkg, "BUILD-2026-9999"); !errors.Is(err, usecase.ErrBuildNotFound) {
		t.Errorf("quitar un BUILD inexistente: %v", err)
	}
	if _, err := addBuild(uc, dev, pkg, "BUILD-2026-0009"); !errors.Is(err, domain.ErrBuildOtherSystem) || !axis.IsCode(err, axis.InvalidAction) {
		t.Errorf("BUILD de otro sistema: %v", err)
	}
	// Quitar uno que no está no cambia nada.
	if out, err := removeBuild(uc, dev, pkg, "BUILD-2026-0001"); err != nil || content(out) != "" {
		t.Errorf("quitar un BUILD que no está: %v %s", err, content(out))
	}
	if len(w.audit) != audited || len(w.reopened) != 0 || !w.packages[pkg].UpdatedAt.Equal(t0) {
		t.Fatalf("nada de lo anterior guarda, audita ni reabre la regresión: %+v %v", w.audit[audited:], w.reopened)
	}

	// Si la auditoría falla o la regresión no se puede reabrir, el BUILD no entra.
	w.auditErr = errors.New("sin auditoría")
	if _, err := addBuild(uc, dev, pkg, "BUILD-2026-0001"); err == nil || len(w.packages[pkg].Builds) != 0 {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr, w.reopenErr = nil, errors.New("regresión caída")
	if _, err := addBuild(uc, dev, pkg, "BUILD-2026-0001"); err == nil || len(w.packages[pkg].Builds) != 0 || len(w.audit) != audited {
		t.Fatalf("nada se guarda si la regresión no se reabre: %v", err)
	}
	w.reopenErr = nil

	// Agregar: guarda, audita y avisa a la regresión.
	out, err := addBuild(uc, dev, pkg, "BUILD-2026-0002")
	if err != nil {
		t.Fatal(err)
	}
	if content(out) != "BUILD-2026-0002" || !out.Package.UpdatedAt.Equal(t1) || out.Package.Builds[0].Version != "2.14.1" {
		t.Errorf("PKG con el BUILD: %+v", out.Package)
	}
	if got, want := missing(out), "in_testing[package_has_content!,regression_confirmed!]"; got != want {
		t.Errorf("qué falta = %s, quiero %s", got, want)
	}
	want := port.AuditRecord{Actor: "u3", PackageId: out.Package.Id, Code: code(pkg), Action: port.AuditUpdate, Field: "builds", NewValue: "BUILD-2026-0002"}
	if len(w.audit) != audited+1 || w.audit[audited] != want {
		t.Errorf("auditoría: %+v", w.audit[audited:])
	}
	if !reflect.DeepEqual(w.reopened, []string{pkg + "/u3/Cambió el contenido del PKG."}) {
		t.Errorf("aviso a la regresión: %v", w.reopened)
	}

	// Agregar uno que ya está no cambia nada.
	if out, err = addBuild(uc, dev, pkg, "BUILD-2026-0002"); err != nil || content(out) != "BUILD-2026-0002" || len(w.audit) != audited+1 || len(w.reopened) != 1 {
		t.Errorf("agregar un BUILD que ya está: %v %s", err, content(out))
	}
	if out, err = addBuild(uc, dev, pkg, "BUILD-2026-0001"); err != nil || content(out) != "BUILD-2026-0001 BUILD-2026-0002" {
		t.Errorf("segundo BUILD: %v %s", err, content(out))
	}

	// Quitar: guarda, audita y avisa a la regresión.
	if out, err = removeBuild(uc, dev, pkg, "BUILD-2026-0002"); err != nil || content(out) != "BUILD-2026-0001" {
		t.Fatalf("quitar: %v %s", err, content(out))
	}
	want = port.AuditRecord{Actor: "u3", PackageId: out.Package.Id, Code: code(pkg), Action: port.AuditUpdate, Field: "builds", OldValue: "BUILD-2026-0002"}
	if w.audit[len(w.audit)-1] != want || len(w.reopened) != 3 {
		t.Errorf("auditoría de quitar: %+v %v", w.audit[len(w.audit)-1], w.reopened)
	}
	w.auditErr = errors.New("sin auditoría")
	if _, err := removeBuild(uc, dev, pkg, "BUILD-2026-0001"); err == nil || len(w.packages[pkg].Builds) != 1 {
		t.Fatalf("nada se quita si la auditoría falla: %v", err)
	}
	w.auditErr = nil
}

func TestPackageRequirements(t *testing.T) {
	uc, w, _ := setup(t)
	pkg := create(t, uc)
	audited := len(w.audit)

	// Errores que no guardan nada.
	if _, err := addRequirement(uc, qa, pkg, "REQ-2026-0001"); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("agregar sin pkg.build: %v", err)
	}
	if _, err := removeRequirement(uc, qa, pkg, "REQ-2026-0001"); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("quitar sin pkg.build: %v", err)
	}
	if _, err := addRequirement(uc, dev, "PKG-2026-9999", "REQ-2026-0001"); !errors.Is(err, usecase.ErrPackageNotFound) {
		t.Errorf("PKG inexistente: %v", err)
	}
	if _, err := addRequirement(uc, dev, pkg, "REQ-2026-9999"); !errors.Is(err, usecase.ErrRequirementNotFound) || !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("REQ inexistente: %v", err)
	}
	if _, err := removeRequirement(uc, dev, pkg, "REQ-2026-9999"); !errors.Is(err, usecase.ErrRequirementNotFound) {
		t.Errorf("quitar un REQ inexistente: %v", err)
	}
	if _, err := addRequirement(uc, dev, pkg, "REQ-2026-0009"); !errors.Is(err, domain.ErrRequirementOtherSystem) {
		t.Errorf("REQ de otro sistema: %v", err)
	}
	if _, err := addRequirement(uc, dev, pkg, "REQ-2026-0003"); !errors.Is(err, domain.ErrRequirementNotApproved) || !axis.IsCode(err, axis.InvalidAction) {
		t.Errorf("REQ sin aprobar: %v", err)
	}
	// Quien arma el PKG pero no puede pasar el REQ a «En paquete» no lo agrega.
	if _, err := addRequirement(uc, builder, pkg, "REQ-2026-0001"); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("agregar un REQ «Aprobado QA» sin el permiso de su transición: %v", err)
	}
	if out, err := removeRequirement(uc, dev, pkg, "REQ-2026-0001"); err != nil || content(out) != "" {
		t.Errorf("quitar un REQ que no está: %v %s", err, content(out))
	}
	if len(w.audit) != audited || len(w.reopened) != 0 || len(w.moved) != 0 || len(w.packages[pkg].Requirements) != 0 ||
		w.requirements["REQ-2026-0001"].Ref.State.Key != "approved_qa" {
		t.Fatalf("nada de lo anterior guarda, audita, mueve el REQ ni reabre la regresión: %+v %v %v", w.audit[audited:], w.moved, w.reopened)
	}

	// Si la auditoría falla, ni entra el REQ ni cambia su estado.
	w.auditErr = errors.New("sin auditoría")
	if _, err := addRequirement(uc, dev, pkg, "REQ-2026-0001"); err == nil || len(w.packages[pkg].Requirements) != 0 ||
		w.requirements["REQ-2026-0001"].Ref.State.Key != "approved_qa" || len(w.moved) != 0 {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	// Un REQ «Aprobado QA» entra y pasa a «En paquete» con el mismo actor.
	out, err := addRequirement(uc, dev, pkg, "REQ-2026-0001")
	if err != nil {
		t.Fatal(err)
	}
	if content(out) != "REQ-2026-0001=in_package" || out.Package.Requirements[0].Title != "Requerimiento 1" || out.Package.Requirements[0].State.Name != "En paquete" {
		t.Errorf("PKG con el REQ: %+v", out.Package.Requirements)
	}
	if !reflect.DeepEqual(w.moved, []string{"REQ-2026-0001/u3"}) {
		t.Errorf("el REQ debía pasar a «En paquete» con el mismo actor: %v", w.moved)
	}
	want := port.AuditRecord{Actor: "u3", PackageId: out.Package.Id, Code: code(pkg), Action: port.AuditUpdate, Field: "requirements", NewValue: "REQ-2026-0001"}
	if len(w.audit) != audited+1 || w.audit[audited] != want || !reflect.DeepEqual(w.reopened, []string{pkg + "/u3/Cambió el contenido del PKG."}) {
		t.Errorf("auditoría y aviso: %+v %v", w.audit[audited:], w.reopened)
	}

	// Uno que ya está «En paquete» entra sin transición: no hace falta su permiso.
	if out, err = addRequirement(uc, builder, pkg, "REQ-2026-0002"); err != nil || content(out) != "REQ-2026-0001=in_package REQ-2026-0002=in_package" || len(w.moved) != 1 {
		t.Errorf("REQ ya en paquete: %v %s %v", err, content(out), w.moved)
	}
	// Agregar uno que ya está no cambia nada.
	if out, err = addRequirement(uc, dev, pkg, "REQ-2026-0001"); err != nil || len(out.Package.Requirements) != 2 || len(w.audit) != audited+2 || len(w.reopened) != 2 {
		t.Errorf("agregar un REQ que ya está: %v", err)
	}

	// Quitar no cambia el estado del REQ.
	if out, err = removeRequirement(uc, dev, pkg, "REQ-2026-0001"); err != nil || content(out) != "REQ-2026-0002=in_package" {
		t.Fatalf("quitar: %v %s", err, content(out))
	}
	if w.requirements["REQ-2026-0001"].Ref.State.Key != "in_package" || len(w.moved) != 1 {
		t.Errorf("quitar un REQ no cambia su estado: %+v", w.requirements["REQ-2026-0001"].Ref.State)
	}
	want = port.AuditRecord{Actor: "u3", PackageId: out.Package.Id, Code: code(pkg), Action: port.AuditUpdate, Field: "requirements", OldValue: "REQ-2026-0001"}
	if w.audit[len(w.audit)-1] != want || len(w.reopened) != 3 {
		t.Errorf("auditoría de quitar: %+v %v", w.audit[len(w.audit)-1], w.reopened)
	}
}

func TestContentOnlyChangesInAssembly(t *testing.T) {
	uc, w, _ := setup(t)
	pkg := filled(t, uc, w)
	if _, err := move(uc, qa, pkg, "in_testing", ""); err != nil {
		t.Fatal(err)
	}
	audited, reopened := len(w.audit), len(w.reopened)
	calls := map[string]func() error{
		"agregar BUILD":            func() error { _, err := addBuild(uc, dev, pkg, "BUILD-2026-0002"); return err },
		"agregar BUILD que está":   func() error { _, err := addBuild(uc, dev, pkg, "BUILD-2026-0001"); return err },
		"quitar BUILD":             func() error { _, err := removeBuild(uc, dev, pkg, "BUILD-2026-0001"); return err },
		"agregar REQ":              func() error { _, err := addRequirement(uc, dev, pkg, "REQ-2026-0002"); return err },
		"quitar REQ":               func() error { _, err := removeRequirement(uc, dev, pkg, "REQ-2026-0001"); return err },
		"quitar REQ que no está":   func() error { _, err := removeRequirement(uc, dev, pkg, "REQ-2026-0002"); return err },
		"quitar BUILD que no está": func() error { _, err := removeBuild(uc, dev, pkg, "BUILD-2026-0002"); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, domain.ErrNotInAssembly) || !axis.IsCode(err, axis.InvalidAction) {
			t.Errorf("%s fuera de armado: %v", name, err)
		}
	}
	out, err := uc.GetPackage.Execute(ctx, usecase.GetPackageInput{Code: pkg})
	if err != nil || content(out) != "BUILD-2026-0001 REQ-2026-0001=in_package" || len(w.audit) != audited || len(w.reopened) != reopened {
		t.Errorf("fuera de armado nada cambia: %v %s", err, content(out))
	}
}

func TestTransitionPackage(t *testing.T) {
	uc, w, clock := setup(t)
	pkg := create(t, uc)
	clock.now = t1
	blocked := func(actor shared.Actor, to string, wantParts ...string) {
		t.Helper()
		before := w.packages[pkg].State
		_, err := move(uc, actor, pkg, to, "")
		if !axis.IsCode(err, axis.InvalidAction) {
			t.Fatalf("pasar a %s debía bloquearse: %v", to, err)
		}
		for _, part := range wantParts {
			if !strings.Contains(err.Error(), part) {
				t.Errorf("pasar a %s: el mensaje %q no dice %q", to, err.Error(), part)
			}
		}
		if w.packages[pkg].State != before {
			t.Errorf("una transición bloqueada no cambia el estado: %+v", w.packages[pkg].State)
		}
	}

	// Lo que no llega a evaluar reglas.
	if _, err := move(uc, qa, pkg, "", ""); !reflect.DeepEqual(fieldsOf(err), []string{"to:required"}) {
		t.Errorf("sin destino: %v", err)
	}
	if _, err := move(uc, qa, "PKG-2026-9999", "in_testing", ""); !errors.Is(err, usecase.ErrPackageNotFound) {
		t.Errorf("PKG inexistente: %v", err)
	}
	if _, err := move(uc, qa, pkg, "verified", ""); !axis.IsCode(err, axis.InvalidAction) {
		t.Errorf("paso que el flujo no permite: %v", err)
	}
	if _, err := move(uc, dev, pkg, "in_testing", ""); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("sin el permiso de la transición: %v", err)
	}

	// A pruebas: contenido y regresión confirmada.
	blocked(qa, "in_testing", "No se puede pasar a «En pruebas».", "El PKG está vacío", "Falta proponer y confirmar la regresión.")
	if _, err := addBuild(uc, dev, pkg, "BUILD-2026-0001"); err != nil {
		t.Fatal(err)
	}
	blocked(qa, "in_testing", "El PKG no tiene ningún REQ.")
	if _, err := addRequirement(uc, dev, pkg, "REQ-2026-0001"); err != nil {
		t.Fatal(err)
	}
	w.regression = domain.RegressionFact{Exists: true, Unpassed: []string{"TC-2026-0001", "TC-2026-0002"}}
	blocked(qa, "in_testing", "No se puede pasar a «En pruebas». Falta confirmar la selección de regresión.")
	w.regression.Confirmed = true
	audited, reopened := len(w.audit), len(w.reopened)

	w.auditErr = errors.New("sin auditoría")
	if _, err := move(uc, qa, pkg, "in_testing", ""); err == nil || w.packages[pkg].State.Key != "in_assembly" {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	out, err := move(uc, qa, pkg, "in_testing", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if out.Package.State != states["in_testing"] || !out.Package.UpdatedAt.Equal(t1) {
		t.Errorf("PKG en pruebas: %+v", out.Package)
	}
	if got, want := missing(out), "in_assembly[] ready_to_release[regression_passed!]"; got != want {
		t.Errorf("qué falta = %s, quiero %s", got, want)
	}
	if related := out.Missing[1].Conditions[0].Related; !reflect.DeepEqual(related, []string{"TC-2026-0001", "TC-2026-0002"}) {
		t.Errorf("TC relacionados: %v", related)
	}
	want := port.AuditRecord{Actor: "u2", PackageId: out.Package.Id, Code: code(pkg), Action: port.AuditTransition, Field: "state", OldValue: "in_assembly", NewValue: "in_testing"}
	if len(w.audit) != audited+1 || w.audit[audited] != want {
		t.Errorf("auditoría: %+v", w.audit[audited:])
	}
	if len(w.reopened) != reopened {
		t.Errorf("avanzar no reabre la regresión: %v", w.reopened)
	}

	// Volver a armado: exige motivo y quita la confirmación de la regresión.
	if _, err := move(uc, qa, pkg, "in_assembly", "  "); !reflect.DeepEqual(fieldsOf(err), []string{"reason:required"}) {
		t.Errorf("volver a armado sin motivo: %v", err)
	}
	w.reopenErr = errors.New("regresión caída")
	if _, err := move(uc, qa, pkg, "in_assembly", "Falta un BUILD"); err == nil || w.packages[pkg].State.Key != "in_testing" {
		t.Fatalf("si la regresión no se reabre, el PKG no vuelve a armado: %v", err)
	}
	w.reopenErr = nil
	if out, err = move(uc, qa, pkg, "in_assembly", "  Falta un BUILD  "); err != nil || out.Package.State != states["in_assembly"] {
		t.Fatalf("volver a armado: %v %+v", err, out.Package.State)
	}
	if last := w.audit[len(w.audit)-1]; last.Action != port.AuditTransition || last.NewValue != "in_assembly" || last.Reason != "Falta un BUILD" {
		t.Errorf("auditoría de volver a armado: %+v", last)
	}
	if !reflect.DeepEqual(w.reopened[reopened:], []string{pkg + "/u2/El PKG volvió a armado."}) {
		t.Errorf("volver a armado debe reabrir la regresión: %v", w.reopened[reopened:])
	}
	if _, err := move(uc, qa, pkg, "in_testing", ""); err != nil {
		t.Fatal(err)
	}

	// A listo para liberar: regresión aprobada y sin DEF alto abierto en los REQ del PKG.
	w.defects["REQ-2026-0001"] = []domain.DefectFact{{Code: "DEF-2026-0091", Severity: "high"}, {Code: "DEF-2026-0092", Severity: "low"}}
	w.defects["REQ-2026-0002"] = []domain.DefectFact{{Code: "DEF-2026-0099", Severity: "high"}} // no está en el PKG
	blocked(qa, "ready_to_release", "No se puede pasar a «Listo para liberar».",
		"Faltan 2 TC sin RUN aprobado sobre un BUILD del PKG: TC-2026-0001, TC-2026-0002.", "Hay 1 DEF abierto de severidad alta: DEF-2026-0091.")
	w.regression.Unpassed = nil
	blocked(qa, "ready_to_release", "Hay 1 DEF abierto de severidad alta: DEF-2026-0091.")
	w.regression = domain.RegressionFact{}
	w.defects = map[string][]domain.DefectFact{"REQ-2026-0001": {{Code: "DEF-2026-0092", Severity: "low"}}}
	blocked(qa, "ready_to_release", "El PKG no tiene selección de regresión.")
	w.regression = domain.RegressionFact{Exists: true, Confirmed: true}
	if _, err := move(uc, qa, pkg, "ready_to_release", ""); err != nil {
		t.Fatalf("a listo para liberar: %v", err)
	}

	// Aprobar para producción es de change management; desplegado exige el despliegue en producción.
	if _, err := move(uc, qa, pkg, "approved_for_production", ""); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("QA no aprueba para producción: %v", err)
	}
	if _, err := move(uc, manager, pkg, "approved_for_production", ""); err != nil {
		t.Fatal(err)
	}
	blocked(manager, "deployed", "No se puede pasar a «Desplegado». Falta un despliegue exitoso del PKG en un ambiente de producción.")
	w.deployed = true
	if _, err := move(uc, manager, pkg, "deployed", ""); err != nil {
		t.Fatal(err)
	}
	if out, err = move(uc, qa, pkg, "verified", ""); err != nil || out.Package.State.Key != "verified" || len(out.Missing) != 0 {
		t.Errorf("verificado: %v %+v", err, out)
	}
}

func TestRulesThatOnlyWarnOrAreUnknown(t *testing.T) {
	uc, w, _ := setup(t)
	pkg := create(t, uc)

	// Una regla que solo avisa no impide el paso, pero aparece en «qué falta».
	w.rules["in_assembly>in_testing"] = []domain.Rule{{Key: domain.RuleHasContent, Blocks: false}, {Key: "desconocida", Blocks: false}}
	out, err := uc.GetPackage.Execute(ctx, usecase.GetPackageInput{Code: pkg})
	if err != nil || missing(out) != "in_testing[package_has_content,desconocida]" {
		t.Errorf("qué falta con reglas que solo avisan: %v %s", err, missing(out))
	}
	if _, err := move(uc, qa, pkg, "in_testing", ""); err != nil {
		t.Fatalf("una regla que solo avisa no bloquea: %v", err)
	}

	// Una regla que el dominio no conoce y que bloquea, bloquea.
	w.rules["in_testing>ready_to_release"] = []domain.Rule{{Key: "min_coverage", Blocks: true}}
	_, err = move(uc, qa, pkg, "ready_to_release", "")
	if !axis.IsCode(err, axis.InvalidAction) || !strings.Contains(err.Error(), "La regla «min_coverage» no se puede evaluar.") {
		t.Errorf("regla desconocida: %v", err)
	}
	if w.packages[pkg].State.Key != "in_testing" {
		t.Errorf("el PKG no debía avanzar: %+v", w.packages[pkg].State)
	}
}
