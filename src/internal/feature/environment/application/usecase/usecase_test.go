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
	"kaoru/internal/feature/environment/application/port"
	"kaoru/internal/feature/environment/application/usecase"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	environments map[string]domain.Environment // por código
	deployments  []domain.Deployment           // en el orden en que se registraron
	sequences    map[string]int                // por prefijo
	audit        []port.AuditRecord
	auditErr     error
	locked       []string // códigos de ENV bloqueados, en orden
	inTx         bool
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	environments, deployments := maps.Clone(w.environments), slices.Clone(w.deployments)
	sequences, audit := maps.Clone(w.sequences), slices.Clone(w.audit)
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.environments, w.deployments, w.sequences, w.audit = environments, deployments, sequences, audit
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

func (w *world) NextCode(_ context.Context, prefix string, year int) (shared.Code, error) {
	w.sequences[prefix]++
	return shared.NewCode(prefix, year, w.sequences[prefix]), nil
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

// nameTaken hace de la restricción única (sistema, nombre) de la base.
func (w *world) nameTaken(e domain.Environment) bool {
	s := e.Snapshot()
	for _, other := range w.environments {
		if o := other.Snapshot(); o.Id != s.Id && o.System.Key == s.System.Key && o.Name == s.Name {
			return true
		}
	}
	return false
}

func (w *world) InsertEnvironment(_ context.Context, e domain.Environment) error {
	if !w.inTx {
		return errors.New("InsertEnvironment fuera de una transacción")
	}
	if w.nameTaken(e) {
		return axis.New(axis.DuplicateResource, "Ya existe.")
	}
	w.environments[e.Snapshot().Code.String()] = e
	return nil
}

func (w *world) UpdateEnvironment(_ context.Context, e domain.Environment) error {
	if !w.inTx || !slices.Contains(w.locked, e.Snapshot().Code.String()) {
		return errors.New("UpdateEnvironment sin bloquear el ambiente en una transacción")
	}
	if w.nameTaken(e) {
		return axis.New(axis.DuplicateResource, "Ya existe.")
	}
	w.environments[e.Snapshot().Code.String()] = e
	return nil
}

func (w *world) LockEnvironment(_ context.Context, code shared.Code) error {
	if _, ok := w.environments[code.String()]; !ok {
		return notFound
	}
	if !w.inTx {
		return errors.New("LockEnvironment fuera de una transacción")
	}
	w.locked = append(w.locked, code.String())
	return nil
}

// view arma el ENV como lo lee la base: con el nombre de quien lo creó y lo que dejó su último despliegue exitoso.
func (w *world) view(e domain.Environment) port.EnvironmentView {
	s := e.Snapshot()
	s.CreatedBy.Name = "Usuario " + string(s.CreatedBy.Id)
	view := port.EnvironmentView{Environment: domain.Rehydrate(s)}
	for _, d := range w.deployments {
		ds := d.Snapshot()
		if ds.Environment.Id != s.Id || ds.Result != domain.ResultSuccess {
			continue
		}
		if view.Installed == nil || !ds.ExecutedAt.Before(view.Installed.DeployedAt) {
			view.Installed = &domain.Installed{Deployment: ds.Code.String(), Build: ds.Build, Package: ds.Package, DeployedAt: ds.ExecutedAt}
		}
	}
	return view
}

func (w *world) FindEnvironment(_ context.Context, code shared.Code) (port.EnvironmentView, error) {
	e, ok := w.environments[code.String()]
	if !ok {
		return port.EnvironmentView{}, notFound
	}
	return w.view(e), nil
}

// ListEnvironments devuelve por código, con los filtros del puerto.
func (w *world) ListEnvironments(_ context.Context, f port.EnvironmentFilter) ([]port.EnvironmentView, int, error) {
	var out []port.EnvironmentView
	for _, code := range slices.Sorted(maps.Keys(w.environments)) {
		s := w.environments[code].Snapshot()
		if (f.SystemKey != "" && s.System.Key != f.SystemKey) || (f.Kind != "" && string(s.Kind) != f.Kind) ||
			(!f.IncludeRetired && s.RetiredAt != nil) {
			continue
		}
		out = append(out, w.view(w.environments[code]))
	}
	return out, len(out), nil
}

// Los PKG del catálogo: uno de SyPago y uno de SIMF.
var packages = map[string]domain.PackageRef{
	"PKG-2026-0001": {Id: "p1", Code: "PKG-2026-0001", Title: "Paquete de noviembre", SystemKey: "SYPAGO"},
	"PKG-2026-0009": {Id: "p9", Code: "PKG-2026-0009", Title: "Mensajería 4.2", SystemKey: "SIMF"},
}

func (w *world) FindPackage(_ context.Context, code string) (domain.PackageRef, error) {
	pkg, ok := packages[code]
	if !ok {
		return domain.PackageRef{}, notFound
	}
	return pkg, nil
}

func (w *world) InsertDeployment(_ context.Context, d domain.Deployment) error {
	if !w.inTx || !slices.Contains(w.locked, d.Snapshot().Environment.Code) {
		return errors.New("InsertDeployment sin bloquear el ambiente en una transacción")
	}
	w.deployments = append(w.deployments, d)
	return nil
}

// FindDeployment devuelve el DEP con el nombre de quien lo registró, como lo hace la base.
func (w *world) FindDeployment(_ context.Context, code shared.Code) (domain.Deployment, error) {
	for _, d := range w.deployments {
		if s := d.Snapshot(); s.Code == code {
			s.ExecutedBy.Name = "Usuario " + string(s.ExecutedBy.Id)
			return domain.RehydrateDeployment(s), nil
		}
	}
	return domain.Deployment{}, notFound
}

// ListDeployments devuelve del más reciente al más antiguo, con los filtros del puerto.
func (w *world) ListDeployments(_ context.Context, f port.DeploymentFilter) ([]domain.Deployment, int, error) {
	var out []domain.Deployment
	for _, d := range slices.Backward(w.deployments) {
		s := d.Snapshot()
		if (f.EnvironmentCode != "" && s.Environment.Code != f.EnvironmentCode) ||
			(f.BuildCode != "" && (s.Build == nil || s.Build.Code != f.BuildCode)) ||
			(f.PackageCode != "" && (s.Package == nil || s.Package.Code != f.PackageCode)) {
			continue
		}
		out = append(out, d)
	}
	return out, len(out), nil
}

func (w *world) HasProductionDeployment(_ context.Context, packageCode string) (bool, error) {
	for _, d := range w.deployments {
		s := d.Snapshot()
		if s.Package != nil && s.Package.Code == packageCode && s.Result == domain.ResultSuccess && s.Environment.Kind == domain.KindProduction {
			return true, nil
		}
	}
	return false, nil
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

// fakeBuilds: un BUILD de SyPago y uno de SIMF.
type fakeBuilds struct{}

var builds = map[string]domain.BuildRef{
	"BUILD-2026-0001": {Id: "b1", Code: "BUILD-2026-0001", Version: "2.14.0", Checksum: strings.Repeat("a", 64), SystemKey: "SYPAGO"},
	"BUILD-2026-0009": {Id: "b9", Code: "BUILD-2026-0009", Version: "4.2.0", Checksum: strings.Repeat("d", 64), SystemKey: "SIMF"},
}

func (fakeBuilds) Find(_ context.Context, code string) (domain.BuildRef, error) {
	build, ok := builds[code]
	if !ok {
		return domain.BuildRef{}, notFound
	}
	return build, nil
}

// fakePermissions: qa mantiene ambientes y despliega en los de prueba; change-manager solo despliega
// en producción; keeper solo mantiene ambientes.
type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	granted := map[string][]string{
		"qa":             {usecase.PermissionManage, usecase.PermissionDeploy},
		"change-manager": {usecase.PermissionDeployProduction},
		"keeper":         {usecase.PermissionManage},
	}
	for _, role := range roles {
		if slices.Contains(granted[role], permission) {
			return nil
		}
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return t0 }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	t0      = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	qa      = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	dev     = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	keeper  = shared.Actor{Id: "u6", Roles: []string{"keeper"}}
	ctx     = context.Background()
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{environments: map[string]domain.Environment{}, sequences: map[string]int{}}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Builds: fakeBuilds{}, Permissions: fakePermissions{}, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

var baseEnvironment = usecase.CreateEnvironmentInput{
	Actor: qa, SystemKey: "SYPAGO", Name: "QA-1", Kind: "baseline", OS: "linux", Notes: "Línea base",
}

// environment registra un ambiente de ese sistema, con ese nombre y tipo, y devuelve su código.
func environment(t *testing.T, uc *usecase.UseCases, system, name, kind string) string {
	t.Helper()
	in := baseEnvironment
	in.SystemKey, in.Name, in.Kind = system, name, kind
	out, err := uc.CreateEnvironment.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	return out.Environment.Code.String()
}

// deploy registra un despliegue como ese actor y devuelve su código.
func deploy(t *testing.T, uc *usecase.UseCases, actor shared.Actor, env, build, pkg, result string, ago time.Duration) string {
	t.Helper()
	out, err := uc.RegisterDeployment.Execute(ctx, usecase.RegisterDeploymentInput{
		Actor: actor, EnvironmentCode: env, BuildCode: build, PackageCode: pkg, Result: result, ExecutedAt: t0.Add(-ago),
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.Code.String()
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

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestCreateEnvironment(t *testing.T) {
	uc, w := setup(t)
	in := baseEnvironment
	in.Actor, in.Name = keeper, " QA-1 "
	out, err := uc.CreateEnvironment.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	saved := w.environments["ENV-2026-0001"].Snapshot()
	if out.Environment.Code.String() != "ENV-2026-0001" || saved.Id != "id-1" || saved.System.Key != "SYPAGO" || saved.Name != "QA-1" ||
		saved.Kind != domain.KindBaseline || saved.OS != domain.OSLinux || saved.Notes != "Línea base" || saved.RetiredAt != nil ||
		!saved.CreatedAt.Equal(t0) || !saved.UpdatedAt.Equal(t0) {
		t.Errorf("ENV guardado inesperado: %+v", saved)
	}
	// Quién lo creó sale de la sesión; un ambiente nuevo no tiene nada instalado.
	if saved.CreatedBy.Id != "u6" || out.Environment.CreatedBy != (domain.UserRef{Id: "u6", Name: "Usuario u6"}) || out.Installed != nil {
		t.Errorf("salida inesperada: %+v", out)
	}
	want := port.AuditRecord{Actor: "u6", EntityType: port.EntityEnvironment, EntityId: "id-1", Code: saved.Code, Action: port.AuditCreate, NewValue: "QA-1"}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Errorf("auditoría del alta: %+v", w.audit)
	}

	// El mismo nombre vale en otro sistema.
	if other := environment(t, uc, "SIMF", "QA-1", "clean"); other != "ENV-2026-0002" {
		t.Errorf("segundo ENV = %s", other)
	}
}

func TestCreateEnvironmentErrors(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(in *usecase.CreateEnvironmentInput)
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", func(in *usecase.CreateEnvironmentInput) { in.Actor = dev }, axis.PermissionDenied, nil},
		{"desplegar en producción no da permiso de mantener", func(in *usecase.CreateEnvironmentInput) { in.Actor = manager }, axis.PermissionDenied, nil},
		{"sin sistema", func(in *usecase.CreateEnvironmentInput) { in.SystemKey = "" }, axis.InvalidRequest, []string{"system:required"}},
		{"sistema desconocido", func(in *usecase.CreateEnvironmentInput) { in.SystemKey = "OTRO" }, axis.InvalidRequest, []string{"system:invalid"}},
		{"campos vacíos", func(in *usecase.CreateEnvironmentInput) { in.Name, in.Kind, in.OS = "", "", "" },
			axis.InvalidRequest, []string{"name:required", "kind:required", "os:required"}},
		{"tipo y sistema operativo desconocidos", func(in *usecase.CreateEnvironmentInput) { in.Kind, in.OS = "staging", "macos" },
			axis.InvalidRequest, []string{"kind:invalid", "os:invalid"}},
		{"nombre repetido en el sistema", func(in *usecase.CreateEnvironmentInput) { in.Name = "Existente" }, axis.InvalidRequest, []string{"name:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			environment(t, uc, "SYPAGO", "Existente", "clean")
			in := baseEnvironment
			tc.mutate(&in)
			_, err := uc.CreateEnvironment.Execute(ctx, in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.environments) != 1 || len(w.audit) != 1 || w.sequences[shared.PrefixEnvironment] != 1 {
				t.Error("un ENV rechazado no debe dejar nada, ni gastar un código")
			}
		})
	}
}

func TestUpdateEnvironment(t *testing.T) {
	uc, w := setup(t)
	code := environment(t, uc, "SYPAGO", "QA-1", "baseline")

	out, err := uc.UpdateEnvironment.Execute(ctx, usecase.UpdateEnvironmentInput{
		Actor: keeper, Code: code, Edit: domain.Edit{Name: ptr(" QA-2 "), Kind: ptr("clean"), Retired: ptr(true)},
	})
	if err != nil {
		t.Fatal(err)
	}
	saved := w.environments[code].Snapshot()
	if out.Environment.Name != "QA-2" || saved.Name != "QA-2" || saved.Kind != domain.KindClean || saved.RetiredAt == nil ||
		!saved.RetiredAt.Equal(t0) || saved.System.Key != "SYPAGO" {
		t.Errorf("edición no guardada: %+v / %+v", out, saved)
	}
	record := func(field, old, new string) port.AuditRecord {
		return port.AuditRecord{Actor: "u6", EntityType: port.EntityEnvironment, EntityId: saved.Id, Code: saved.Code,
			Action: port.AuditUpdate, Field: field, OldValue: old, NewValue: new}
	}
	want := []port.AuditRecord{record("name", "QA-1", "QA-2"), record("kind", "baseline", "clean"), record("retired", "false", "true")}
	if !slices.Equal(w.audit[1:], want) {
		t.Errorf("auditoría de la edición: %+v", w.audit[1:])
	}
	// Escribir sobre un ambiente existente lo bloquea primero.
	if !slices.Equal(w.locked, []string{code}) {
		t.Errorf("bloqueos = %v", w.locked)
	}

	// Editar con lo mismo, o sin campos, no escribe ni audita.
	if _, err := uc.UpdateEnvironment.Execute(ctx, usecase.UpdateEnvironmentInput{
		Actor: qa, Code: code, Edit: domain.Edit{Name: ptr("QA-2"), Retired: ptr(true)},
	}); err != nil || len(w.audit) != 4 {
		t.Errorf("una edición sin cambios no se audita: %v, %d registros", err, len(w.audit))
	}
	if out, err := uc.UpdateEnvironment.Execute(ctx, usecase.UpdateEnvironmentInput{Actor: qa, Code: code}); err != nil ||
		out.Environment.Name != "QA-2" || len(w.audit) != 4 {
		t.Errorf("una edición vacía devuelve el ENV tal cual: %+v, %v", out, err)
	}
}

func TestUpdateEnvironmentErrors(t *testing.T) {
	cases := []struct {
		name       string
		actor      shared.Actor
		code       string
		edit       domain.Edit
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", dev, "ENV-2026-0001", domain.Edit{Name: ptr("Otro")}, axis.PermissionDenied, nil},
		{"desplegar en producción no da permiso de mantener", manager, "ENV-2026-0001", domain.Edit{Retired: ptr(true)}, axis.PermissionDenied, nil},
		{"ENV inexistente", qa, "ENV-2026-0099", domain.Edit{Name: ptr("Otro")}, axis.NotFoundResource, nil},
		{"código mal formado", qa, "env 1", domain.Edit{Name: ptr("Otro")}, axis.NotFoundResource, nil},
		{"valores que no valen", qa, "ENV-2026-0001", domain.Edit{Name: ptr(" "), OS: ptr("macos")}, axis.InvalidRequest, []string{"name:required", "os:invalid"}},
		{"nombre de otro ambiente del sistema", qa, "ENV-2026-0001", domain.Edit{Name: ptr("QA-2")}, axis.InvalidRequest, []string{"name:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			environment(t, uc, "SYPAGO", "QA-1", "baseline")
			environment(t, uc, "SYPAGO", "QA-2", "clean")
			_, err := uc.UpdateEnvironment.Execute(ctx, usecase.UpdateEnvironmentInput{Actor: tc.actor, Code: tc.code, Edit: tc.edit})
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if s := w.environments["ENV-2026-0001"].Snapshot(); s.Name != "QA-1" || s.RetiredAt != nil || len(w.audit) != 2 {
				t.Error("una edición rechazada no cambia el ENV ni se audita")
			}
		})
	}
	uc, _ := setup(t)
	if _, err := uc.UpdateEnvironment.Execute(ctx, usecase.UpdateEnvironmentInput{Actor: qa, Code: "ENV-2026-0099"}); !errors.Is(err, usecase.ErrEnvironmentNotFound) {
		t.Errorf("ENV inexistente: %v", err)
	}
}

func TestRegisterDeployment(t *testing.T) {
	uc, w := setup(t)
	test := environment(t, uc, "SYPAGO", "QA-1", "baseline")
	production := environment(t, uc, "SYPAGO", "Producción", "production")
	w.locked = nil

	// En un ambiente de prueba basta dep.create.
	out, err := uc.RegisterDeployment.Execute(ctx, usecase.RegisterDeploymentInput{
		Actor: qa, EnvironmentCode: test, BuildCode: "BUILD-2026-0001", Result: "success", Notes: " Sin novedades ", ExecutedAt: t0.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	saved := w.deployments[0].Snapshot()
	wantEnv := domain.EnvironmentRef{Id: "id-1", Code: test, Name: "QA-1", Kind: domain.KindBaseline}
	if out.Code.String() != "DEP-2026-0001" || saved.Id != "id-3" || saved.Environment != wantEnv || saved.Build == nil ||
		saved.Build.Id != "b1" || saved.Package != nil || saved.Result != domain.ResultSuccess || saved.Notes != "Sin novedades" ||
		!saved.ExecutedAt.Equal(t0.Add(-time.Hour)) || !saved.CreatedAt.Equal(t0) {
		t.Errorf("DEP guardado inesperado: %+v", saved)
	}
	// Quién lo registra sale de la sesión; la respuesta trae su nombre.
	if saved.ExecutedBy.Id != "u2" || out.ExecutedBy != (domain.UserRef{Id: "u2", Name: "Usuario u2"}) {
		t.Errorf("executedBy = %+v / %+v", saved.ExecutedBy, out.ExecutedBy)
	}
	want := port.AuditRecord{Actor: "u2", EntityType: port.EntityDeployment, EntityId: "id-3", Code: out.Code,
		Action: port.AuditCreate, Field: "result", NewValue: "success"}
	if len(w.audit) != 3 || w.audit[2] != want {
		t.Errorf("auditoría del despliegue: %+v", w.audit)
	}
	if !slices.Equal(w.locked, []string{test}) {
		t.Errorf("el despliegue debe bloquear su ambiente: %v", w.locked)
	}

	// En producción exige dep.create_production, y también vale desplegar un PKG.
	out, err = uc.RegisterDeployment.Execute(ctx, usecase.RegisterDeploymentInput{
		Actor: manager, EnvironmentCode: production, PackageCode: "PKG-2026-0001", Result: "failed", ExecutedAt: t0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Code.String() != "DEP-2026-0002" || out.Build != nil || out.Package == nil || out.Package.Id != "p1" ||
		out.Environment.Kind != domain.KindProduction || out.Result != domain.ResultFailed {
		t.Errorf("DEP de un PKG en producción: %+v", out)
	}
}

func TestRegisterDeploymentErrors(t *testing.T) {
	const test, production, retired = "ENV-2026-0001", "ENV-2026-0002", "ENV-2026-0003"
	cases := []struct {
		name       string
		mutate     func(in *usecase.RegisterDeploymentInput)
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", func(in *usecase.RegisterDeploymentInput) { in.Actor = dev }, axis.PermissionDenied, nil},
		{"mantener ambientes no da permiso de desplegar", func(in *usecase.RegisterDeploymentInput) { in.Actor = keeper }, axis.PermissionDenied, nil},
		{"dep.create no basta en producción", func(in *usecase.RegisterDeploymentInput) { in.EnvironmentCode = production }, axis.PermissionDenied, nil},
		{"dep.create_production no basta en un ambiente de prueba", func(in *usecase.RegisterDeploymentInput) { in.Actor = manager }, axis.PermissionDenied, nil},
		{"sin permiso y sin ambiente", func(in *usecase.RegisterDeploymentInput) { in.Actor, in.EnvironmentCode = dev, "ENV-2026-0099" }, axis.PermissionDenied, nil},
		{"sin ambiente", func(in *usecase.RegisterDeploymentInput) { in.EnvironmentCode = "" }, axis.InvalidRequest, []string{"environment:required"}},
		{"ambiente inexistente", func(in *usecase.RegisterDeploymentInput) { in.EnvironmentCode = "ENV-2026-0099" }, axis.InvalidRequest, []string{"environment:invalid"}},
		{"ambiente retirado", func(in *usecase.RegisterDeploymentInput) { in.EnvironmentCode = retired }, axis.InvalidRequest, []string{"environment:invalid"}},
		{"ni BUILD ni PKG", func(in *usecase.RegisterDeploymentInput) { in.BuildCode = "" }, axis.InvalidRequest, []string{"build:required"}},
		{"BUILD y PKG", func(in *usecase.RegisterDeploymentInput) { in.PackageCode = "PKG-2026-0001" }, axis.InvalidRequest, []string{"package:invalid"}},
		{"BUILD inexistente", func(in *usecase.RegisterDeploymentInput) { in.BuildCode = "BUILD-2026-0099" }, axis.InvalidRequest, []string{"build:invalid"}},
		{"PKG inexistente", func(in *usecase.RegisterDeploymentInput) { in.BuildCode, in.PackageCode = "", "PKG-2026-0099" }, axis.InvalidRequest, []string{"package:invalid"}},
		{"BUILD de otro sistema", func(in *usecase.RegisterDeploymentInput) { in.BuildCode = "BUILD-2026-0009" }, axis.InvalidRequest, []string{"build:invalid"}},
		{"PKG de otro sistema", func(in *usecase.RegisterDeploymentInput) { in.BuildCode, in.PackageCode = "", "PKG-2026-0009" }, axis.InvalidRequest, []string{"package:invalid"}},
		{"sin resultado ni fecha", func(in *usecase.RegisterDeploymentInput) { in.Result, in.ExecutedAt = "", time.Time{} },
			axis.InvalidRequest, []string{"result:required", "executedAt:required"}},
		{"resultado desconocido", func(in *usecase.RegisterDeploymentInput) { in.Result = "ok" }, axis.InvalidRequest, []string{"result:invalid"}},
		{"fecha futura", func(in *usecase.RegisterDeploymentInput) { in.ExecutedAt = t0.Add(time.Minute) }, axis.InvalidRequest, []string{"executedAt:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			environment(t, uc, "SYPAGO", "QA-1", "baseline")
			environment(t, uc, "SYPAGO", "Producción", "production")
			environment(t, uc, "SYPAGO", "QA viejo", "clean")
			if _, err := uc.UpdateEnvironment.Execute(ctx, usecase.UpdateEnvironmentInput{Actor: qa, Code: retired, Edit: domain.Edit{Retired: ptr(true)}}); err != nil {
				t.Fatal(err)
			}
			in := usecase.RegisterDeploymentInput{
				Actor: qa, EnvironmentCode: test, BuildCode: "BUILD-2026-0001", Result: "success", ExecutedAt: t0.Add(-time.Hour),
			}
			tc.mutate(&in)
			_, err := uc.RegisterDeployment.Execute(ctx, in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.deployments) != 0 || len(w.audit) != 4 || w.sequences[shared.PrefixDeployment] != 0 {
				t.Error("un despliegue rechazado no debe dejar nada, ni gastar un código")
			}
		})
	}
}

// Lo instalado es lo que dejó el último despliegue exitoso; los fallidos y los revertidos no cuentan.
func TestEnvironmentInstalled(t *testing.T) {
	uc, _ := setup(t)
	code := environment(t, uc, "SYPAGO", "QA-1", "baseline")
	installed := func() *domain.Installed {
		t.Helper()
		out, err := uc.GetEnvironment.Execute(ctx, usecase.GetEnvironmentInput{Code: code})
		if err != nil {
			t.Fatal(err)
		}
		return out.Installed
	}
	if installed() != nil {
		t.Error("sin despliegues no hay nada instalado")
	}
	deploy(t, uc, qa, code, "BUILD-2026-0001", "", "failed", 5*time.Hour)
	if installed() != nil {
		t.Error("un despliegue fallido no instala nada")
	}
	first := deploy(t, uc, qa, code, "BUILD-2026-0001", "", "success", 4*time.Hour)
	if i := installed(); i == nil || i.Deployment != first || i.Build == nil || i.Build.Code != "BUILD-2026-0001" || i.Package != nil ||
		!i.DeployedAt.Equal(t0.Add(-4*time.Hour)) {
		t.Errorf("instalado tras el primer éxito: %+v", i)
	}
	second := deploy(t, uc, qa, code, "", "PKG-2026-0001", "success", 2*time.Hour)
	deploy(t, uc, qa, code, "BUILD-2026-0001", "", "rolled_back", time.Hour)
	if i := installed(); i == nil || i.Deployment != second || i.Build != nil || i.Package == nil || i.Package.Code != "PKG-2026-0001" {
		t.Errorf("instalado tras el segundo éxito: %+v", i)
	}
	// La lista trae lo mismo.
	list, err := uc.ListEnvironments.Execute(ctx, usecase.ListEnvironmentsInput{Page: 1, Size: 25})
	if err != nil || len(list.Environments) != 1 || list.Environments[0].Installed == nil || list.Environments[0].Installed.Deployment != second {
		t.Errorf("lista con lo instalado: %+v, %v", list, err)
	}
}

func TestGetFindAndListEnvironments(t *testing.T) {
	uc, _ := setup(t)
	first := environment(t, uc, "SYPAGO", "QA-1", "baseline")
	second := environment(t, uc, "SYPAGO", "Producción", "production")
	third := environment(t, uc, "SIMF", "QA SIMF", "baseline")
	if _, err := uc.UpdateEnvironment.Execute(ctx, usecase.UpdateEnvironmentInput{Actor: qa, Code: first, Edit: domain.Edit{Retired: ptr(true)}}); err != nil {
		t.Fatal(err)
	}

	// Leer no exige permiso: ni siquiera recibe un actor.
	out, err := uc.GetEnvironment.Execute(ctx, usecase.GetEnvironmentInput{Code: second})
	if err != nil || out.Environment.Name != "Producción" || out.Environment.CreatedBy.Name != "Usuario u2" {
		t.Errorf("GetEnvironment = %+v, %v", out, err)
	}
	ref, err := uc.FindEnvironment.Execute(ctx, usecase.FindEnvironmentInput{Code: first})
	want := usecase.EnvironmentRef{Id: "id-1", Code: first, Name: "QA-1", Kind: "baseline", SystemKey: "SYPAGO", Retired: true}
	if err != nil || ref != want {
		t.Errorf("FindEnvironment = %+v, %v", ref, err)
	}
	if ref, err := uc.FindEnvironment.Execute(ctx, usecase.FindEnvironmentInput{Code: second}); err != nil || ref.Kind != "production" || ref.Retired {
		t.Errorf("FindEnvironment de producción = %+v, %v", ref, err)
	}
	for _, code := range []string{"ENV-2026-0099", "no es un código", ""} {
		if _, err := uc.GetEnvironment.Execute(ctx, usecase.GetEnvironmentInput{Code: code}); !errors.Is(err, usecase.ErrEnvironmentNotFound) {
			t.Errorf("GetEnvironment(%q): %v", code, err)
		}
		if _, err := uc.FindEnvironment.Execute(ctx, usecase.FindEnvironmentInput{Code: code}); !errors.Is(err, usecase.ErrEnvironmentNotFound) {
			t.Errorf("FindEnvironment(%q): %v", code, err)
		}
	}

	cases := []struct {
		name string
		in   usecase.ListEnvironmentsInput
		want []string
	}{
		{"sin filtro: los retirados no salen", usecase.ListEnvironmentsInput{}, []string{second, third}},
		{"con retirados", usecase.ListEnvironmentsInput{IncludeRetired: true}, []string{first, second, third}},
		{"por sistema", usecase.ListEnvironmentsInput{SystemKey: "SIMF"}, []string{third}},
		{"por tipo", usecase.ListEnvironmentsInput{Kind: "baseline", IncludeRetired: true}, []string{first, third}},
		{"sin resultados", usecase.ListEnvironmentsInput{SystemKey: "SIMF", Kind: "production"}, nil},
	}
	for _, tc := range cases {
		list, err := uc.ListEnvironments.Execute(ctx, tc.in)
		var got []string
		for _, e := range list.Environments {
			got = append(got, e.Environment.Code.String())
		}
		if err != nil || list.Total != len(tc.want) || !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v (total %d), %v; quiero %v", tc.name, got, list.Total, err, tc.want)
		}
	}
}

func TestGetAndListDeployments(t *testing.T) {
	uc, _ := setup(t)
	test := environment(t, uc, "SYPAGO", "QA-1", "baseline")
	production := environment(t, uc, "SYPAGO", "Producción", "production")
	first := deploy(t, uc, qa, test, "BUILD-2026-0001", "", "success", 3*time.Hour)
	second := deploy(t, uc, qa, test, "", "PKG-2026-0001", "failed", 2*time.Hour)
	third := deploy(t, uc, manager, production, "", "PKG-2026-0001", "success", time.Hour)

	out, err := uc.GetDeployment.Execute(ctx, usecase.GetDeploymentInput{Code: third})
	if err != nil || out.Code.String() != third || out.Environment.Code != production || out.ExecutedBy.Name != "Usuario u1" {
		t.Errorf("GetDeployment = %+v, %v", out, err)
	}
	for _, code := range []string{"DEP-2026-0099", "no es un código", ""} {
		if _, err := uc.GetDeployment.Execute(ctx, usecase.GetDeploymentInput{Code: code}); !errors.Is(err, usecase.ErrDeploymentNotFound) {
			t.Errorf("GetDeployment(%q): %v", code, err)
		}
	}

	cases := []struct {
		name string
		in   usecase.ListDeploymentsInput
		want []string
	}{
		{"sin filtro", usecase.ListDeploymentsInput{}, []string{third, second, first}},
		{"por ambiente", usecase.ListDeploymentsInput{EnvironmentCode: test}, []string{second, first}},
		{"por BUILD", usecase.ListDeploymentsInput{BuildCode: "BUILD-2026-0001"}, []string{first}},
		{"por PKG", usecase.ListDeploymentsInput{PackageCode: "PKG-2026-0001"}, []string{third, second}},
		{"por PKG y ambiente", usecase.ListDeploymentsInput{PackageCode: "PKG-2026-0001", EnvironmentCode: production}, []string{third}},
		{"sin resultados", usecase.ListDeploymentsInput{BuildCode: "BUILD-2026-0009"}, nil},
	}
	for _, tc := range cases {
		list, err := uc.ListDeployments.Execute(ctx, tc.in)
		var got []string
		for _, d := range list.Deployments {
			got = append(got, d.Code.String())
		}
		if err != nil || list.Total != len(tc.want) || !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v (total %d), %v; quiero %v", tc.name, got, list.Total, err, tc.want)
		}
	}
}

// Solo cuenta un despliegue exitoso del PKG en un ambiente de producción.
func TestHasProductionDeployment(t *testing.T) {
	uc, _ := setup(t)
	test := environment(t, uc, "SYPAGO", "QA-1", "baseline")
	production := environment(t, uc, "SYPAGO", "Producción", "production")
	has := func() bool {
		t.Helper()
		out, err := uc.HasProductionDeployment.Execute(ctx, usecase.HasProductionDeploymentInput{PackageCode: "PKG-2026-0001"})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	if has() {
		t.Error("sin despliegues")
	}
	deploy(t, uc, qa, test, "", "PKG-2026-0001", "success", 3*time.Hour)
	if has() {
		t.Error("un despliegue exitoso en un ambiente de prueba no cuenta")
	}
	deploy(t, uc, manager, production, "", "PKG-2026-0001", "failed", 2*time.Hour)
	deploy(t, uc, manager, production, "BUILD-2026-0001", "", "success", 2*time.Hour)
	if has() {
		t.Error("ni un despliegue fallido en producción, ni el de un BUILD suelto")
	}
	deploy(t, uc, manager, production, "", "PKG-2026-0001", "success", time.Hour)
	if !has() {
		t.Error("un despliegue exitoso del PKG en producción sí cuenta")
	}
	if out, err := uc.HasProductionDeployment.Execute(ctx, usecase.HasProductionDeploymentInput{PackageCode: "PKG-2026-0099"}); err != nil || out {
		t.Errorf("PKG inexistente: %v, %v", out, err)
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	code := environment(t, uc, "SYPAGO", "QA-1", "baseline")
	deploy(t, uc, qa, code, "BUILD-2026-0001", "", "success", time.Hour)
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")

	in := baseEnvironment
	in.Name = "QA-2"
	if _, err := uc.CreateEnvironment.Execute(ctx, in); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("crear ENV: %v", err)
	}
	if _, err := uc.UpdateEnvironment.Execute(ctx, usecase.UpdateEnvironmentInput{
		Actor: qa, Code: code, Edit: domain.Edit{Name: ptr("Otro"), Retired: ptr(true)},
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("editar ENV: %v", err)
	}
	if _, err := uc.RegisterDeployment.Execute(ctx, usecase.RegisterDeploymentInput{
		Actor: qa, EnvironmentCode: code, BuildCode: "BUILD-2026-0001", Result: "success", ExecutedAt: t0,
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("registrar DEP: %v", err)
	}

	if s := w.environments[code].Snapshot(); len(w.environments) != 1 || s.Name != "QA-1" || s.RetiredAt != nil || len(w.deployments) != 1 ||
		w.sequences[shared.PrefixEnvironment] != 1 || w.sequences[shared.PrefixDeployment] != 1 || len(w.audit) != 2 {
		t.Error("si la auditoría falla, nada debe quedar guardado")
	}
}
