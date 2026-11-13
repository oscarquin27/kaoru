package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/production/application/port"
	"kaoru/internal/feature/production/application/usecase"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	templates   []domain.Template            // en el orden en que se publicaron
	checklists  map[string]domain.Checklist  // por código
	diagnostics map[string]domain.Diagnostic // por código
	smokeTests  map[string]domain.SmokeTest  // por código
	sequences   map[string]int               // por prefijo
	audit       []port.AuditRecord
	auditErr    error
	locked      []string // ids de PKG bloqueados, en orden
	inTx        bool
	// versionTaken simula que otra publicación simultánea tomó la versión.
	versionTaken bool
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	templates, checklists, diagnostics := slices.Clone(w.templates), maps.Clone(w.checklists), maps.Clone(w.diagnostics)
	smokeTests, sequences, audit, locked := maps.Clone(w.smokeTests), maps.Clone(w.sequences), slices.Clone(w.audit), slices.Clone(w.locked)
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.templates, w.checklists, w.diagnostics = templates, checklists, diagnostics
		w.smokeTests, w.sequences, w.audit, w.locked = smokeTests, sequences, audit, locked
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

func named(u domain.UserRef) domain.UserRef {
	u.Name = "Usuario " + string(u.Id)
	return u
}

func (w *world) NextCode(_ context.Context, prefix string, year int) (shared.Code, error) {
	w.sequences[prefix]++
	return shared.NewCode(prefix, year, w.sequences[prefix]), nil
}

var systems = map[string]domain.System{
	"SYPAGO": {Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"},
	"SIMF":   {Key: "SIMF", Name: "SIMF", DefaultRoute: "high"},
}

func (w *world) FindSystem(_ context.Context, key string) (domain.System, error) {
	system, ok := systems[key]
	if !ok {
		return domain.System{}, notFound
	}
	return system, nil
}

func (w *world) LockPackage(_ context.Context, packageId string) error {
	if !w.inTx {
		return errors.New("LockPackage fuera de una transacción")
	}
	w.locked = append(w.locked, packageId)
	return nil
}

func (w *world) NextTemplateVersion(_ context.Context, systemKey string) (int, error) {
	version := 1
	for _, t := range w.templates {
		if s := t.Snapshot(); s.System.Key == systemKey {
			version = s.Version + 1
		}
	}
	return version, nil
}

func (w *world) InsertTemplate(_ context.Context, t domain.Template) error {
	if !w.inTx {
		return errors.New("InsertTemplate fuera de una transacción")
	}
	if w.versionTaken {
		return axis.New(axis.DuplicateResource, "Ya existe.")
	}
	w.templates = append(w.templates, t)
	return nil
}

func (w *world) LatestTemplate(_ context.Context, systemKey string) (domain.Template, error) {
	for _, t := range slices.Backward(w.templates) {
		if s := t.Snapshot(); s.System.Key == systemKey {
			s.CreatedBy = named(s.CreatedBy)
			return domain.RehydrateTemplate(s), nil
		}
	}
	return domain.Template{}, notFound
}

func (w *world) ListTemplates(_ context.Context, systemKey string) ([]domain.Template, error) {
	var out []domain.Template
	for _, t := range slices.Backward(w.templates) {
		if systemKey == "" || t.Snapshot().System.Key == systemKey {
			out = append(out, t)
		}
	}
	return out, nil
}

// Los PKG, los ambientes y los despliegues que otros features ya registraron.
var (
	packages = map[string]domain.PackageRef{
		"PKG-2026-0001": {Id: "p1", Code: "PKG-2026-0001", Title: "Mensajería 4.2", SystemKey: "SIMF"},
		"PKG-2026-0002": {Id: "p2", Code: "PKG-2026-0002", Title: "Paquete de noviembre", SystemKey: "SYPAGO"},
	}
	environments = map[string]domain.EnvironmentRef{
		"ENV-2026-0001": {Id: "e1", Code: "ENV-2026-0001", Name: "Producción SIMF", Kind: "production", SystemKey: "SIMF"},
		"ENV-2026-0002": {Id: "e2", Code: "ENV-2026-0002", Name: "QA limpio", Kind: "clean", SystemKey: "SIMF"},
		"ENV-2026-0003": {Id: "e3", Code: "ENV-2026-0003", Name: "Producción SyPago", Kind: "production", SystemKey: "SYPAGO"},
	}
	simfPackage = domain.PackageRef{Id: "p1", Code: "PKG-2026-0001", Title: "Mensajería 4.2"}
	deployments = map[string]domain.DeploymentRef{
		"DEP-2026-0001": {Id: "d1", Code: "DEP-2026-0001", Package: &simfPackage, ExecutedAt: t0.Add(-time.Hour),
			Environment: domain.EnvironmentRef{Id: "e1", Code: "ENV-2026-0001", Name: "Producción SIMF", Kind: "production"}},
		"DEP-2026-0002": {Id: "d2", Code: "DEP-2026-0002", Package: &simfPackage, ExecutedAt: t0.Add(-2 * time.Hour),
			Environment: domain.EnvironmentRef{Id: "e2", Code: "ENV-2026-0002", Name: "QA limpio", Kind: "clean"}},
		"DEP-2026-0003": {Id: "d3", Code: "DEP-2026-0003", ExecutedAt: t0.Add(-3 * time.Hour),
			Environment: domain.EnvironmentRef{Id: "e1", Code: "ENV-2026-0001", Name: "Producción SIMF", Kind: "production"}},
	}
)

func (w *world) FindPackage(_ context.Context, code string) (domain.PackageRef, error) {
	pkg, ok := packages[code]
	if !ok {
		return domain.PackageRef{}, notFound
	}
	return pkg, nil
}

func (w *world) FindEnvironment(_ context.Context, code string) (domain.EnvironmentRef, error) {
	environment, ok := environments[code]
	if !ok {
		return domain.EnvironmentRef{}, notFound
	}
	return environment, nil
}

func (w *world) FindDeployment(_ context.Context, code string) (domain.DeploymentRef, error) {
	deployment, ok := deployments[code]
	if !ok {
		return domain.DeploymentRef{}, notFound
	}
	return deployment, nil
}

func (w *world) InsertChecklist(_ context.Context, c domain.Checklist) error {
	if !w.inTx {
		return errors.New("InsertChecklist fuera de una transacción")
	}
	w.checklists[c.Snapshot().Code.String()] = c
	return nil
}

func (w *world) InsertSignature(_ context.Context, checklistId string, signature domain.Signature) error {
	for code, c := range w.checklists {
		s := c.Snapshot()
		if s.Id != checklistId {
			continue
		}
		if !w.inTx || !slices.Contains(w.locked, s.Package.Id) {
			return errors.New("InsertSignature sin bloquear el PKG en una transacción")
		}
		s.Steps = slices.Clone(s.Steps)
		s.Steps[signature.Position-1].Signature = &signature
		w.checklists[code] = domain.RehydrateChecklist(s)
		return nil
	}
	return notFound
}

// FindChecklist devuelve el CHK con los nombres de quienes lo abrieron y firmaron, como lo hace la base.
func (w *world) FindChecklist(_ context.Context, code shared.Code) (domain.Checklist, error) {
	c, ok := w.checklists[code.String()]
	if !ok {
		return domain.Checklist{}, notFound
	}
	s := c.Snapshot()
	s.StartedBy, s.Steps = named(s.StartedBy), slices.Clone(s.Steps)
	for i, step := range s.Steps {
		if step.Signature != nil {
			signature := *step.Signature
			signature.SignedBy = named(signature.SignedBy)
			s.Steps[i].Signature = &signature
		}
	}
	return domain.RehydrateChecklist(s), nil
}

func (w *world) ListChecklists(_ context.Context, f port.Filter) ([]domain.Checklist, int, error) {
	var out []domain.Checklist
	for _, code := range slices.Backward(slices.Sorted(maps.Keys(w.checklists))) {
		if c := w.checklists[code]; f.PackageCode == "" || c.Snapshot().Package.Code == f.PackageCode {
			out = append(out, c)
		}
	}
	return out, len(out), nil
}

func (w *world) InsertDiagnostic(_ context.Context, d domain.Diagnostic) error {
	if !w.inTx {
		return errors.New("InsertDiagnostic fuera de una transacción")
	}
	w.diagnostics[d.Snapshot().Code.String()] = d
	return nil
}

func (w *world) InsertReview(_ context.Context, diagnosticId string, review domain.Review) error {
	for code, d := range w.diagnostics {
		s := d.Snapshot()
		if s.Id != diagnosticId {
			continue
		}
		if !w.inTx || !slices.Contains(w.locked, s.Package.Id) {
			return errors.New("InsertReview sin bloquear el PKG en una transacción")
		}
		s.Review = &review
		w.diagnostics[code] = domain.RehydrateDiagnostic(s)
		return nil
	}
	return notFound
}

func (w *world) FindDiagnostic(_ context.Context, code shared.Code) (domain.Diagnostic, error) {
	d, ok := w.diagnostics[code.String()]
	if !ok {
		return domain.Diagnostic{}, notFound
	}
	s := d.Snapshot()
	s.SubmittedBy = named(s.SubmittedBy)
	if s.Review != nil {
		review := *s.Review
		review.ReviewedBy = named(review.ReviewedBy)
		s.Review = &review
	}
	return domain.RehydrateDiagnostic(s), nil
}

func (w *world) ListDiagnostics(_ context.Context, f port.Filter) ([]domain.Diagnostic, int, error) {
	var out []domain.Diagnostic
	for _, code := range slices.Backward(slices.Sorted(maps.Keys(w.diagnostics))) {
		if d := w.diagnostics[code]; f.PackageCode == "" || d.Snapshot().Package.Code == f.PackageCode {
			out = append(out, d)
		}
	}
	return out, len(out), nil
}

func (w *world) InsertSmokeTest(_ context.Context, s domain.SmokeTest) error {
	if !w.inTx {
		return errors.New("InsertSmokeTest fuera de una transacción")
	}
	w.smokeTests[s.Snapshot().Code.String()] = s
	return nil
}

func (w *world) FindSmokeTest(_ context.Context, code shared.Code) (domain.SmokeTest, error) {
	k, ok := w.smokeTests[code.String()]
	if !ok {
		return domain.SmokeTest{}, notFound
	}
	s := k.Snapshot()
	s.ExecutedBy = named(s.ExecutedBy)
	return domain.RehydrateSmokeTest(s), nil
}

func (w *world) ListSmokeTests(_ context.Context, f port.Filter) ([]domain.SmokeTest, int, error) {
	var out []domain.SmokeTest
	for _, code := range slices.Backward(slices.Sorted(maps.Keys(w.smokeTests))) {
		if k := w.smokeTests[code]; f.PackageCode == "" || k.Snapshot().Package.Code == f.PackageCode {
			out = append(out, k)
		}
	}
	return out, len(out), nil
}

func (w *world) PackageFacts(_ context.Context, packageCode string) (domain.PackageFacts, error) {
	if packageCode != "PKG-2026-0001" {
		return domain.PackageFacts{}, nil
	}
	return domain.PackageFacts{ChecklistExists: true, ChecklistPending: 1, Diagnostic: domain.DiagnosticPending, Smoke: "passed"}, nil
}

// --- Permissions ---

var granted = map[string][]string{
	"change-manager": {usecase.PermissionManageTemplate, usecase.PermissionStart, usecase.PermissionSign},
	"ops":            {usecase.PermissionSign, usecase.PermissionSubmitDiagnostic},
	"dev":            {usecase.PermissionSubmitDiagnostic},
	"qa":             {usecase.PermissionReviewDiagnostic, usecase.PermissionSmoke},
}

func (w *world) Require(_ context.Context, roles []string, permission string) error {
	for _, role := range roles {
		if slices.Contains(granted[role], permission) {
			return nil
		}
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
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
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	qa      = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	dev     = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	ops     = shared.Actor{Id: "u5", Roles: []string{"ops"}}
	ctx     = context.Background()
)

func setup(t *testing.T) (*usecase.UseCases, *world, *fixedClock) {
	t.Helper()
	w := &world{
		checklists: map[string]domain.Checklist{}, diagnostics: map[string]domain.Diagnostic{},
		smokeTests: map[string]domain.SmokeTest{}, sequences: map[string]int{},
	}
	clock := &fixedClock{now: t0}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Permissions: w, AuditTrail: w, UnitOfWork: w, Clock: clock, IdGenerator: &seqIDs{},
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

func denied(t *testing.T, what string, err error) {
	t.Helper()
	if !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("%s debía negarse por permiso: %v", what, err)
	}
}

var twoSteps = []domain.StepInput{{Title: "Respaldar la base"}, {Title: "Desplegar el paquete", Description: "Con el instalador."}}

func publish(t *testing.T, uc *usecase.UseCases, system string) domain.TemplateSnapshot {
	t.Helper()
	out, err := uc.PublishChecklistTemplate.Execute(ctx, usecase.PublishChecklistTemplateInput{Actor: manager, SystemKey: system, Steps: twoSteps})
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

func TestPublishChecklistTemplate(t *testing.T) {
	uc, w, _ := setup(t)
	in := usecase.PublishChecklistTemplateInput{Actor: manager, SystemKey: "SIMF", Steps: twoSteps}

	for _, actor := range []shared.Actor{qa, dev, ops} {
		_, err := uc.PublishChecklistTemplate.Execute(ctx, usecase.PublishChecklistTemplateInput{Actor: actor, SystemKey: "SIMF", Steps: twoSteps})
		denied(t, "publicar la plantilla como "+actor.Roles[0], err)
	}
	for system, want := range map[string]string{"": "system:required", "OTRO": "system:invalid"} {
		_, err := uc.PublishChecklistTemplate.Execute(ctx, usecase.PublishChecklistTemplateInput{Actor: manager, SystemKey: system, Steps: twoSteps})
		if !reflect.DeepEqual(fieldsOf(err), []string{want}) {
			t.Errorf("sistema %q: %v", system, err)
		}
	}
	if _, err := uc.PublishChecklistTemplate.Execute(ctx, usecase.PublishChecklistTemplateInput{Actor: manager, SystemKey: "SIMF"}); !reflect.DeepEqual(fieldsOf(err), []string{"steps:required"}) {
		t.Errorf("sin pasos: %v", err)
	}
	w.auditErr = errors.New("sin auditoría")
	if _, err := uc.PublishChecklistTemplate.Execute(ctx, in); err == nil || len(w.templates) != 0 {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil
	w.versionTaken = true
	if _, err := uc.PublishChecklistTemplate.Execute(ctx, in); !errors.Is(err, usecase.ErrVersionTaken) || len(w.audit) != 0 {
		t.Errorf("versión tomada por otra publicación: %v", err)
	}
	w.versionTaken = false

	out, err := uc.PublishChecklistTemplate.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.TemplateSnapshot{
		Id: out.Id, System: systems["SIMF"], Version: 1, CreatedBy: domain.UserRef{Id: "u1", Name: "Usuario u1"}, CreatedAt: t0,
		Steps: []domain.TemplateStep{{Position: 1, Title: "Respaldar la base"}, {Position: 2, Title: "Desplegar el paquete", Description: "Con el instalador."}},
	}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("plantilla:\n got %+v\nwant %+v", out, want)
	}
	wantAudit := []port.AuditRecord{{Actor: "u1", EntityType: port.EntityTemplate, EntityId: out.Id, Action: port.AuditCreate, Field: "version", NewValue: "SIMF v1"}}
	if !reflect.DeepEqual(w.audit, wantAudit) {
		t.Errorf("auditoría: %+v", w.audit)
	}

	// La versión siguiente es por sistema, y la anterior queda.
	if second := publish(t, uc, "SIMF"); second.Version != 2 {
		t.Errorf("segunda versión de SIMF = %d", second.Version)
	}
	if other := publish(t, uc, "SYPAGO"); other.Version != 1 {
		t.Errorf("primera versión de SyPago = %d", other.Version)
	}
	all, err := uc.ListChecklistTemplates.Execute(ctx, usecase.ListChecklistTemplatesInput{})
	if err != nil || len(all) != 3 {
		t.Fatalf("todas las versiones: %v %d", err, len(all))
	}
	simf, err := uc.ListChecklistTemplates.Execute(ctx, usecase.ListChecklistTemplatesInput{SystemKey: "SIMF"})
	if err != nil || len(simf) != 2 || simf[0].Version != 2 || simf[1].Version != 1 {
		t.Errorf("versiones de SIMF, la más reciente primero: %v %+v", err, simf)
	}
}

func start(t *testing.T, uc *usecase.UseCases) string {
	t.Helper()
	out, err := uc.StartChecklist.Execute(ctx, usecase.StartChecklistInput{Actor: manager, PackageCode: "PKG-2026-0001", EnvironmentCode: "ENV-2026-0001"})
	if err != nil {
		t.Fatal(err)
	}
	return out.Checklist.Code.String()
}

func TestStartChecklist(t *testing.T) {
	uc, w, _ := setup(t)
	in := usecase.StartChecklistInput{Actor: manager, PackageCode: "PKG-2026-0001", EnvironmentCode: "ENV-2026-0001"}

	for _, actor := range []shared.Actor{qa, dev, ops} {
		_, err := uc.StartChecklist.Execute(ctx, usecase.StartChecklistInput{Actor: actor, PackageCode: in.PackageCode, EnvironmentCode: in.EnvironmentCode})
		denied(t, "abrir el checklist como "+actor.Roles[0], err)
	}
	cases := []struct {
		name, pkg, env string
		want           []string
	}{
		{"sin nada", "", "", []string{"package:required", "environment:required"}},
		{"no existen", "PKG-2026-9999", "ENV-2026-9999", []string{"package:invalid", "environment:invalid"}},
	}
	for _, tc := range cases {
		_, err := uc.StartChecklist.Execute(ctx, usecase.StartChecklistInput{Actor: manager, PackageCode: tc.pkg, EnvironmentCode: tc.env})
		if !reflect.DeepEqual(fieldsOf(err), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
	if _, err := uc.StartChecklist.Execute(ctx, in); !errors.Is(err, usecase.ErrNoTemplate) {
		t.Errorf("sin plantilla del sistema: %v", err)
	}
	publish(t, uc, "SYPAGO") // la plantilla de otro sistema no sirve
	if _, err := uc.StartChecklist.Execute(ctx, in); !errors.Is(err, usecase.ErrNoTemplate) {
		t.Errorf("con plantilla solo de otro sistema: %v", err)
	}
	publish(t, uc, "SIMF")
	template := publish(t, uc, "SIMF")
	for name, env := range map[string]string{"de prueba": "ENV-2026-0002", "de otro sistema": "ENV-2026-0003"} {
		_, err := uc.StartChecklist.Execute(ctx, usecase.StartChecklistInput{Actor: manager, PackageCode: in.PackageCode, EnvironmentCode: env})
		if !reflect.DeepEqual(fieldsOf(err), []string{"environment:invalid"}) {
			t.Errorf("ambiente %s: %v", name, err)
		}
	}
	audited := len(w.audit)
	w.auditErr = errors.New("sin auditoría")
	if _, err := uc.StartChecklist.Execute(ctx, in); err == nil || len(w.checklists) != 0 || w.sequences["CHK"] != 0 {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	out, err := uc.StartChecklist.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	s := out.Checklist
	if s.Code.String() != "CHK-2026-0001" || s.Package.Code != "PKG-2026-0001" || s.Environment.Code != "ENV-2026-0001" ||
		s.TemplateId != template.Id || s.TemplateVersion != 2 || len(s.Steps) != 2 || s.StartedBy != (domain.UserRef{Id: "u1", Name: "Usuario u1"}) ||
		!s.CreatedAt.Equal(t0) || out.SignedSteps != 0 || out.Complete {
		t.Errorf("checklist abierto con la última versión: %+v", out)
	}
	wantAudit := port.AuditRecord{Actor: "u1", EntityType: port.EntityChecklist, EntityId: s.Id, Code: "CHK-2026-0001", Action: port.AuditCreate, Field: "package", NewValue: "PKG-2026-0001"}
	if len(w.audit) != audited+1 || w.audit[audited] != wantAudit {
		t.Errorf("auditoría: %+v", w.audit[audited:])
	}

	// Una versión posterior de la plantilla no cambia un checklist ya abierto.
	if _, err := uc.PublishChecklistTemplate.Execute(ctx, usecase.PublishChecklistTemplateInput{
		Actor: manager, SystemKey: "SIMF", Steps: []domain.StepInput{{Title: "Un solo paso"}},
	}); err != nil {
		t.Fatal(err)
	}
	if got, err := uc.GetChecklist.Execute(ctx, usecase.GetChecklistInput{Code: "CHK-2026-0001"}); err != nil || got.Checklist.TemplateVersion != 2 || len(got.Checklist.Steps) != 2 {
		t.Errorf("el checklist conserva su versión: %v %+v", err, got.Checklist)
	}
	for _, code := range []string{"CHK-2026-9999", "nada"} {
		if _, err := uc.GetChecklist.Execute(ctx, usecase.GetChecklistInput{Code: code}); !errors.Is(err, usecase.ErrChecklistNotFound) {
			t.Errorf("checklist %q: %v", code, err)
		}
	}
	list, err := uc.ListChecklists.Execute(ctx, usecase.ListInput{PackageCode: "PKG-2026-0001", Page: 1, Size: 25})
	if err != nil || list.Total != 1 || list.Checklists[0].Checklist.Code != s.Code {
		t.Errorf("lista por PKG: %v %+v", err, list)
	}
	if list, err := uc.ListChecklists.Execute(ctx, usecase.ListInput{PackageCode: "PKG-2026-0002", Page: 1, Size: 25}); err != nil || list.Total != 0 {
		t.Errorf("lista de otro PKG: %v %+v", err, list)
	}
}

func TestSignChecklistStep(t *testing.T) {
	uc, w, clock := setup(t)
	publish(t, uc, "SIMF")
	code := start(t, uc)
	clock.now = t1
	sign := func(actor shared.Actor, code string, position int, comment string) (usecase.ChecklistOutput, error) {
		return uc.SignChecklistStep.Execute(ctx, usecase.SignChecklistStepInput{Actor: actor, Code: code, Position: position, Comment: comment})
	}

	// QA no firma pasos en producción (regla de negocio 6); dev tampoco.
	for _, actor := range []shared.Actor{qa, dev} {
		_, err := sign(actor, code, 1, "")
		denied(t, "firmar como "+actor.Roles[0], err)
	}
	if _, err := sign(ops, "CHK-2026-9999", 1, ""); !errors.Is(err, usecase.ErrChecklistNotFound) {
		t.Errorf("checklist inexistente: %v", err)
	}
	for _, position := range []int{0, 3} {
		if _, err := sign(ops, code, position, ""); !errors.Is(err, domain.ErrStepNotFound) {
			t.Errorf("paso %d: %v", position, err)
		}
	}
	audited := len(w.audit)
	w.auditErr = errors.New("sin auditoría")
	if _, err := sign(ops, code, 1, "Respaldo hecho."); err == nil || w.checklists[code].SignedSteps() != 0 {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	out, err := sign(ops, code, 1, "  Respaldo en /backups/1112.  ")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Signature{Position: 1, Comment: "Respaldo en /backups/1112.", SignedBy: domain.UserRef{Id: "u5", Name: "Usuario u5"}, SignedAt: t1}
	if got := out.Checklist.Steps[0].Signature; got == nil || *got != want || out.SignedSteps != 1 || out.Complete {
		t.Errorf("firma: %+v, avance %d/%v", got, out.SignedSteps, out.Complete)
	}
	wantAudit := port.AuditRecord{
		Actor: "u5", EntityType: port.EntityChecklist, EntityId: out.Checklist.Id, Code: code,
		Action: port.AuditDecision, Field: "step.1", NewValue: "signed", Reason: "Respaldo en /backups/1112.",
	}
	if len(w.audit) != audited+1 || w.audit[audited] != wantAudit {
		t.Errorf("auditoría: %+v", w.audit[audited:])
	}
	if !slices.Contains(w.locked, "p1") {
		t.Errorf("firmar bloquea el PKG del checklist: %v", w.locked)
	}

	// Una firma no se cambia ni se quita, ni siquiera por quien la hizo.
	if _, err := sign(ops, code, 1, "Otra"); !errors.Is(err, domain.ErrStepAlreadySigned) || len(w.audit) != audited+1 {
		t.Errorf("firmar dos veces: %v", err)
	}
	if out, err = sign(manager, code, 2, ""); err != nil || out.SignedSteps != 2 || !out.Complete {
		t.Errorf("con todos los pasos firmados queda completo: %v %+v", err, out)
	}
}

func TestSubmitAndReviewDiagnostic(t *testing.T) {
	uc, w, clock := setup(t)
	submit := func(actor shared.Actor, deployment string) (domain.DiagnosticSnapshot, error) {
		return uc.SubmitDiagnostic.Execute(ctx, usecase.SubmitDiagnosticInput{Actor: actor, DeploymentCode: deployment, Notes: " Salida del diagnóstico adjunta. "})
	}
	// QA no entrega el DIAG: lo revisa (regla de negocio 7).
	for _, actor := range []shared.Actor{qa, manager} {
		_, err := submit(actor, "DEP-2026-0001")
		denied(t, "entregar un DIAG como "+actor.Roles[0], err)
	}
	for deployment, want := range map[string]string{
		"": "deployment:required", "DEP-2026-9999": "deployment:invalid",
		"DEP-2026-0002": "deployment:invalid", // en un ambiente de prueba
		"DEP-2026-0003": "deployment:invalid", // de un BUILD suelto
	} {
		if _, err := submit(dev, deployment); !reflect.DeepEqual(fieldsOf(err), []string{want}) {
			t.Errorf("despliegue %q: %v", deployment, err)
		}
	}
	w.auditErr = errors.New("sin auditoría")
	if _, err := submit(dev, "DEP-2026-0001"); err == nil || len(w.diagnostics) != 0 || w.sequences["DIAG"] != 0 {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	out, err := submit(ops, "DEP-2026-0001")
	if err != nil {
		t.Fatal(err)
	}
	if out.Code.String() != "DIAG-2026-0001" || out.Package != simfPackage || out.Deployment.Code != "DEP-2026-0001" ||
		out.Notes != "Salida del diagnóstico adjunta." || out.SubmittedBy != (domain.UserRef{Id: "u5", Name: "Usuario u5"}) ||
		!out.CreatedAt.Equal(t0) || out.Review != nil {
		t.Errorf("DIAG entregado: %+v", out)
	}
	wantAudit := []port.AuditRecord{{Actor: "u5", EntityType: port.EntityDiagnostic, EntityId: out.Id, Code: "DIAG-2026-0001", Action: port.AuditCreate, Field: "deployment", NewValue: "DEP-2026-0001"}}
	if !reflect.DeepEqual(w.audit, wantAudit) {
		t.Errorf("auditoría: %+v", w.audit)
	}

	clock.now = t1
	review := func(actor shared.Actor, code, result, notes string) (domain.DiagnosticSnapshot, error) {
		return uc.ReviewDiagnostic.Execute(ctx, usecase.ReviewDiagnosticInput{Actor: actor, Code: code, Result: result, Notes: notes})
	}
	// Quien entrega el DIAG no lo revisa.
	for _, actor := range []shared.Actor{dev, ops, manager} {
		_, err := review(actor, "DIAG-2026-0001", "conforming", "")
		denied(t, "revisar un DIAG como "+actor.Roles[0], err)
	}
	for _, code := range []string{"DIAG-2026-9999", "nada"} {
		if _, err := review(qa, code, "conforming", ""); !errors.Is(err, usecase.ErrDiagnosticNotFound) {
			t.Errorf("DIAG %q: %v", code, err)
		}
	}
	if _, err := review(qa, "DIAG-2026-0001", "", ""); !reflect.DeepEqual(fieldsOf(err), []string{"result:required"}) {
		t.Errorf("sin resultado: %v", err)
	}
	if _, err := review(qa, "DIAG-2026-0001", "non_conforming", ""); !reflect.DeepEqual(fieldsOf(err), []string{"notes:required"}) {
		t.Errorf("no conforme sin explicación: %v", err)
	}
	w.auditErr = errors.New("sin auditoría")
	if _, err := review(qa, "DIAG-2026-0001", "conforming", ""); err == nil || w.diagnostics["DIAG-2026-0001"].Snapshot().Review != nil {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	out, err = review(qa, "DIAG-2026-0001", "conforming", " Sin hallazgos. ")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Review{Result: domain.ReviewConforming, Notes: "Sin hallazgos.", ReviewedBy: domain.UserRef{Id: "u2", Name: "Usuario u2"}, ReviewedAt: t1}
	if out.Review == nil || *out.Review != want {
		t.Errorf("revisión: %+v", out.Review)
	}
	if last := w.audit[len(w.audit)-1]; last != (port.AuditRecord{
		Actor: "u2", EntityType: port.EntityDiagnostic, EntityId: out.Id, Code: "DIAG-2026-0001",
		Action: port.AuditDecision, Field: "review", NewValue: "conforming", Reason: "Sin hallazgos.",
	}) {
		t.Errorf("auditoría de la revisión: %+v", last)
	}
	if !slices.Contains(w.locked, "p1") {
		t.Errorf("revisar bloquea el PKG del DIAG: %v", w.locked)
	}
	if _, err := review(qa, "DIAG-2026-0001", "non_conforming", "Me equivoqué."); !errors.Is(err, domain.ErrAlreadyReviewed) {
		t.Errorf("revisar dos veces: %v", err)
	}

	got, err := uc.GetDiagnostic.Execute(ctx, usecase.GetDiagnosticInput{Code: "DIAG-2026-0001"})
	if err != nil || !reflect.DeepEqual(got, out) {
		t.Errorf("leer el DIAG: %v %+v", err, got)
	}
	if _, err := uc.GetDiagnostic.Execute(ctx, usecase.GetDiagnosticInput{Code: "DIAG-2026-0404"}); !errors.Is(err, usecase.ErrDiagnosticNotFound) {
		t.Errorf("DIAG inexistente: %v", err)
	}
	if list, err := uc.ListDiagnostics.Execute(ctx, usecase.ListInput{PackageCode: "PKG-2026-0001", Page: 1, Size: 25}); err != nil || list.Total != 1 {
		t.Errorf("lista por PKG: %v %+v", err, list)
	}
	if list, err := uc.ListDiagnostics.Execute(ctx, usecase.ListInput{PackageCode: "PKG-2026-0002", Page: 1, Size: 25}); err != nil || list.Total != 0 {
		t.Errorf("lista de otro PKG: %v %+v", err, list)
	}
}

func TestRegisterSmokeTest(t *testing.T) {
	uc, w, _ := setup(t)
	register := func(actor shared.Actor, deployment, result, notes string) (domain.SmokeTestSnapshot, error) {
		return uc.RegisterSmokeTest.Execute(ctx, usecase.RegisterSmokeTestInput{Actor: actor, DeploymentCode: deployment, Result: result, Notes: notes})
	}
	for _, actor := range []shared.Actor{dev, ops, manager} {
		_, err := register(actor, "DEP-2026-0001", "passed", "")
		denied(t, "registrar un SMOKE como "+actor.Roles[0], err)
	}
	for deployment, want := range map[string]string{"": "deployment:required", "DEP-2026-9999": "deployment:invalid", "DEP-2026-0002": "deployment:invalid"} {
		if _, err := register(qa, deployment, "passed", ""); !reflect.DeepEqual(fieldsOf(err), []string{want}) {
			t.Errorf("despliegue %q: %v", deployment, err)
		}
	}
	if _, err := register(qa, "DEP-2026-0001", "failed", ""); !reflect.DeepEqual(fieldsOf(err), []string{"notes:required"}) {
		t.Errorf("fallida sin explicación: %v", err)
	}
	w.auditErr = errors.New("sin auditoría")
	if _, err := register(qa, "DEP-2026-0001", "passed", ""); err == nil || len(w.smokeTests) != 0 || w.sequences["SMOKE"] != 0 {
		t.Fatalf("nada se guarda si la auditoría falla: %v", err)
	}
	w.auditErr = nil

	out, err := register(qa, "DEP-2026-0001", "passed", " Login y pago de prueba. ")
	if err != nil {
		t.Fatal(err)
	}
	if out.Code.String() != "SMOKE-2026-0001" || out.Package != simfPackage || out.Deployment.Code != "DEP-2026-0001" ||
		out.Result != domain.SmokePassed || out.Notes != "Login y pago de prueba." ||
		out.ExecutedBy != (domain.UserRef{Id: "u2", Name: "Usuario u2"}) || !out.CreatedAt.Equal(t0) {
		t.Errorf("SMOKE registrado: %+v", out)
	}
	wantAudit := []port.AuditRecord{{Actor: "u2", EntityType: port.EntitySmokeTest, EntityId: out.Id, Code: "SMOKE-2026-0001", Action: port.AuditCreate, Field: "result", NewValue: "passed"}}
	if !reflect.DeepEqual(w.audit, wantAudit) {
		t.Errorf("auditoría: %+v", w.audit)
	}
	got, err := uc.GetSmokeTest.Execute(ctx, usecase.GetSmokeTestInput{Code: "SMOKE-2026-0001"})
	if err != nil || !reflect.DeepEqual(got, out) {
		t.Errorf("leer el SMOKE: %v %+v", err, got)
	}
	for _, code := range []string{"SMOKE-2026-0404", "nada"} {
		if _, err := uc.GetSmokeTest.Execute(ctx, usecase.GetSmokeTestInput{Code: code}); !errors.Is(err, usecase.ErrSmokeTestNotFound) {
			t.Errorf("SMOKE %q: %v", code, err)
		}
	}
	if list, err := uc.ListSmokeTests.Execute(ctx, usecase.ListInput{PackageCode: "PKG-2026-0001", Page: 1, Size: 25}); err != nil || list.Total != 1 {
		t.Errorf("lista por PKG: %v %+v", err, list)
	}
}

func TestGetPackageFacts(t *testing.T) {
	uc, _, _ := setup(t)
	facts, err := uc.GetPackageFacts.Execute(ctx, usecase.GetPackageFactsInput{PackageCode: "PKG-2026-0001"})
	want := domain.PackageFacts{ChecklistExists: true, ChecklistPending: 1, Diagnostic: domain.DiagnosticPending, Smoke: "passed"}
	if err != nil || facts != want {
		t.Errorf("hechos del PKG: %v %+v", err, facts)
	}
	if facts, err := uc.GetPackageFacts.Execute(ctx, usecase.GetPackageFactsInput{PackageCode: "PKG-2026-9999"}); err != nil || facts != (domain.PackageFacts{}) {
		t.Errorf("un PKG sin nada en producción: %v %+v", err, facts)
	}
}
