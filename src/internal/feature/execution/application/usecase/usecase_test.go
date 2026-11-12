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
	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/application/usecase"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	runs         []domain.Run                  // en el orden en que se registraron
	explorations map[string]domain.Exploration // por código
	sequences    map[string]int                // por prefijo
	audit        []port.AuditRecord
	auditErr     error
	lastFilter   port.RunFilter // el último filtro que recibió ListRuns
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	runs, explorations := slices.Clone(w.runs), maps.Clone(w.explorations)
	sequences, audit := maps.Clone(w.sequences), slices.Clone(w.audit)
	if err := fn(ctx); err != nil {
		w.runs, w.explorations, w.sequences, w.audit = runs, explorations, sequences, audit
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

// El REQ-2026-0042 tiene tres TC (uno retirado); el TC-2026-0009 es del REQ-2026-0043. Todos son de SyPago.
var testCases = map[string]domain.TestCaseRef{
	"TC-2026-0001": {Id: "t1", Code: "TC-2026-0001", Title: "Saldo", SystemKey: "SYPAGO"},
	"TC-2026-0002": {Id: "t2", Code: "TC-2026-0002", Title: "Transferencia", SystemKey: "SYPAGO"},
	"TC-2026-0003": {Id: "t3", Code: "TC-2026-0003", Title: "Saldo viejo", Retired: true, SystemKey: "SYPAGO"},
	"TC-2026-0009": {Id: "t9", Code: "TC-2026-0009", Title: "Movimientos", SystemKey: "SYPAGO"},
}

// El catálogo: dos BUILD de SyPago y uno de SIMF; ambientes de SyPago de prueba, de producción y retirado, y uno de SIMF.
var builds = map[string]domain.BuildRef{
	"BUILD-2026-0001": {Id: "b1", Code: "BUILD-2026-0001", SystemKey: "SYPAGO", Version: "1.4.0", Checksum: checksum},
	"BUILD-2026-0002": {Id: "b2", Code: "BUILD-2026-0002", SystemKey: "SYPAGO", Version: "1.5.0", Checksum: strings.Repeat("b", 64)},
	"BUILD-2026-0003": {Id: "b3", Code: "BUILD-2026-0003", SystemKey: "SYPAGO", Version: "1.5.1", Checksum: strings.Repeat("c", 64)},
	"BUILD-2026-0009": {Id: "b9", Code: "BUILD-2026-0009", SystemKey: "SIMF", Version: "4.2.0", Checksum: strings.Repeat("d", 64)},
}

var environments = map[string]domain.EnvironmentRef{
	"ENV-2026-0001": {Id: "e1", Code: "ENV-2026-0001", Name: "QA", SystemKey: "SYPAGO"},
	"ENV-2026-0002": {Id: "e2", Code: "ENV-2026-0002", Name: "Producción", SystemKey: "SYPAGO", Production: true},
	"ENV-2026-0003": {Id: "e3", Code: "ENV-2026-0003", Name: "QA viejo", SystemKey: "SYPAGO", Retired: true},
	"ENV-2026-0009": {Id: "e9", Code: "ENV-2026-0009", Name: "QA SIMF", SystemKey: "SIMF"},
}

// fakeBuilds hace de port.Builds.
type fakeBuilds struct{}

func (fakeBuilds) Find(_ context.Context, code string) (domain.BuildRef, error) {
	build, ok := builds[code]
	if !ok {
		return domain.BuildRef{}, notFound
	}
	return build, nil
}

// fakeEnvironments hace de port.Environments.
type fakeEnvironments struct{}

func (fakeEnvironments) Find(_ context.Context, code string) (domain.EnvironmentRef, error) {
	environment, ok := environments[code]
	if !ok {
		return domain.EnvironmentRef{}, notFound
	}
	return environment, nil
}

func (w *world) NextCode(_ context.Context, prefix string, year int) (shared.Code, error) {
	w.sequences[prefix]++
	return shared.NewCode(prefix, year, w.sequences[prefix]), nil
}

func (w *world) FindTestCase(_ context.Context, code string) (domain.TestCaseRef, error) {
	testCase, ok := testCases[code]
	if !ok {
		return domain.TestCaseRef{}, notFound
	}
	return testCase, nil
}

func (w *world) FindRequirement(_ context.Context, code string) (domain.RequirementRef, error) {
	requirements := map[string]string{"REQ-2026-0042": "r1", "REQ-2026-0043": "r2"}
	id, ok := requirements[code]
	if !ok {
		return domain.RequirementRef{}, notFound
	}
	return domain.RequirementRef{Id: id, Code: code, SystemKey: "SYPAGO"}, nil
}

func (w *world) FindUser(_ context.Context, id shared.UserId) (domain.UserRef, error) {
	if !strings.HasPrefix(string(id), "u") {
		return domain.UserRef{}, notFound
	}
	return domain.UserRef{Id: id, Name: "Usuario " + string(id)}, nil
}

// RequirementTestCases hace de port.TestDesign.
func (w *world) RequirementTestCases(_ context.Context, requirementCode string) ([]domain.CoverageTestCase, error) {
	byRequirement := map[string][]string{
		"REQ-2026-0042": {"TC-2026-0001", "TC-2026-0002", "TC-2026-0003"},
		"REQ-2026-0043": {"TC-2026-0009"},
	}
	codes, ok := byRequirement[requirementCode]
	if !ok {
		return nil, notFound
	}
	var out []domain.CoverageTestCase
	for _, code := range codes {
		tc := testCases[code]
		out = append(out, domain.CoverageTestCase{Id: tc.Id, Code: tc.Code, Retired: tc.Retired})
	}
	return out, nil
}

func (w *world) InsertRun(_ context.Context, run domain.Run) error {
	w.runs = append(w.runs, run)
	return nil
}

func (w *world) FindRun(_ context.Context, code shared.Code) (domain.Run, error) {
	for _, run := range w.runs {
		if run.Snapshot().Code == code {
			return run, nil
		}
	}
	return domain.Run{}, notFound
}

// ListRuns devuelve del más reciente al más antiguo.
func (w *world) ListRuns(_ context.Context, f port.RunFilter) ([]domain.Run, int, error) {
	w.lastFilter = f
	var out []domain.Run
	for _, run := range slices.Backward(w.runs) {
		s := run.Snapshot()
		if (f.FilterTestCases && !slices.Contains(f.TestCaseIds, s.TestCase.Id)) ||
			(f.BuildVersion != "" && s.BuildVersion != f.BuildVersion) {
			continue
		}
		out = append(out, run)
	}
	return out, len(out), nil
}

func (w *world) LatestResults(_ context.Context, testCaseIds []string) (map[string]domain.Result, error) {
	out := map[string]domain.Result{}
	for _, run := range w.runs {
		if s := run.Snapshot(); slices.Contains(testCaseIds, s.TestCase.Id) {
			out[s.TestCase.Id] = s.Result
		}
	}
	return out, nil
}

func (w *world) InsertExploration(_ context.Context, e domain.Exploration) error {
	w.explorations[e.Snapshot().Code.String()] = e
	return nil
}

func (w *world) UpdateExploration(_ context.Context, e domain.Exploration) error {
	w.explorations[e.Snapshot().Code.String()] = e
	return nil
}

func (w *world) FindExploration(_ context.Context, code shared.Code) (domain.Exploration, error) {
	e, ok := w.explorations[code.String()]
	if !ok {
		return domain.Exploration{}, notFound
	}
	return e, nil
}

// ListExplorations devuelve del más reciente al más antiguo.
func (w *world) ListExplorations(_ context.Context, requirementCode string, _, _ int) ([]domain.Exploration, int, error) {
	var out []domain.Exploration
	for _, e := range w.explorations {
		if requirementCode == "" || e.Snapshot().Requirement.Code == requirementCode {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b domain.Exploration) int {
		return strings.Compare(b.Snapshot().Code.String(), a.Snapshot().Code.String())
	})
	return out, len(out), nil
}

func (w *world) Record(_ context.Context, r port.AuditRecord) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	w.audit = append(w.audit, r)
	return nil
}

// fakePermissions: qa registra RUN y EXP; tester solo RUN; explorer solo EXP.
type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	granted := map[string][]string{
		"qa":       {usecase.PermissionCreateRun, usecase.PermissionCreateExploration},
		"tester":   {usecase.PermissionCreateRun},
		"explorer": {usecase.PermissionCreateExploration},
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
	t0       = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	qa       = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	dev      = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	tester   = shared.Actor{Id: "u4", Roles: []string{"tester"}}
	explorer = shared.Actor{Id: "u5", Roles: []string{"explorer"}}
	ghost    = shared.Actor{Id: "x1", Roles: []string{"qa"}} // con permiso, pero sin usuario
	ctx      = context.Background()
	checksum = strings.Repeat("a", 64)
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{explorations: map[string]domain.Exploration{}, sequences: map[string]int{}}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, TestDesign: w, Builds: fakeBuilds{}, Environments: fakeEnvironments{},
		Permissions: fakePermissions{}, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

var baseRun = usecase.CreateRunInput{
	Actor: qa, TestCaseCode: "TC-2026-0001", BuildCode: "BUILD-2026-0001", EnvironmentCode: "ENV-2026-0001",
	Result: "passed", ExecutedAt: t0.Add(-time.Hour),
}

var baseExploration = usecase.CreateExplorationInput{
	Actor: qa, RequirementCode: "REQ-2026-0042", BuildCode: "BUILD-2026-0001", EnvironmentCode: "ENV-2026-0001",
	ManualChanges: "Parche a mano", Findings: "Nada raro", ExecutedAt: t0.Add(-time.Hour),
}

// run registra un RUN del TC con ese resultado sobre el BUILD de esa versión y devuelve su código.
func run(t *testing.T, uc *usecase.UseCases, testCaseCode, result, buildVersion string) string {
	t.Helper()
	in := baseRun
	in.TestCaseCode, in.Result = testCaseCode, result
	for code, build := range builds {
		if build.Version == buildVersion {
			in.BuildCode = code
		}
	}
	out, err := uc.CreateRun.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	return out.Code.String()
}

// explore registra un EXP del REQ y devuelve su código.
func explore(t *testing.T, uc *usecase.UseCases, requirementCode string) string {
	t.Helper()
	in := baseExploration
	in.RequirementCode = requirementCode
	out, err := uc.CreateExploration.Execute(ctx, in)
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

func TestCreateRun(t *testing.T) {
	uc, w := setup(t)
	in := baseRun
	in.Actor, in.Result, in.Notes = tester, "failed", " Saldo negativo "
	out, err := uc.CreateRun.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.runs) != 1 {
		t.Fatalf("se guardaron %d RUN", len(w.runs))
	}
	saved := w.runs[0].Snapshot()
	if out.Code.String() != "RUN-2026-0001" || saved.Id != "id-1" || saved.TestCase.Id != "t1" || saved.BuildVersion != "1.4.0" ||
		saved.BuildChecksum != checksum || saved.EnvironmentName != "QA" || saved.Result != domain.ResultFailed ||
		saved.Notes != "Saldo negativo" || !saved.ExecutedAt.Equal(t0.Add(-time.Hour)) || !saved.CreatedAt.Equal(t0) {
		t.Errorf("RUN guardado inesperado: %+v", saved)
	}
	// Queda atado al BUILD y al ENV del catálogo; versión, checksum y nombre son copias.
	if saved.BuildId != "b1" || saved.BuildCode != "BUILD-2026-0001" || saved.EnvironmentId != "e1" || saved.EnvironmentCode != "ENV-2026-0001" ||
		out.BuildCode != "BUILD-2026-0001" || out.EnvironmentCode != "ENV-2026-0001" {
		t.Errorf("BUILD y ENV del RUN: %+v", saved)
	}
	// Quién lo registra sale de la sesión.
	if saved.ExecutedBy != (domain.UserRef{Id: "u4", Name: "Usuario u4"}) {
		t.Errorf("ejecutor = %+v", saved.ExecutedBy)
	}
	want := port.AuditRecord{
		Actor: "u4", EntityType: port.EntityRun, EntityId: "id-1", Code: out.Code,
		Action: port.AuditCreate, Field: "result", NewValue: "failed",
	}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Errorf("auditoría del alta: %+v", w.audit)
	}
}

func TestCreateRunErrors(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(in *usecase.CreateRunInput)
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", func(in *usecase.CreateRunInput) { in.Actor = dev }, axis.PermissionDenied, nil},
		{"el permiso de EXP no basta", func(in *usecase.CreateRunInput) { in.Actor = explorer }, axis.PermissionDenied, nil},
		{"sin TC", func(in *usecase.CreateRunInput) { in.TestCaseCode = "" }, axis.InvalidRequest, []string{"testCase:required"}},
		{"TC inexistente", func(in *usecase.CreateRunInput) { in.TestCaseCode = "TC-2026-0099" }, axis.InvalidRequest, []string{"testCase:invalid"}},
		{"TC retirado", func(in *usecase.CreateRunInput) { in.TestCaseCode = "TC-2026-0003" }, axis.InvalidAction, nil},
		{"actor sin usuario", func(in *usecase.CreateRunInput) { in.Actor = ghost }, axis.NotFoundResource, nil},
		{"sin TC, BUILD ni ENV", func(in *usecase.CreateRunInput) { in.TestCaseCode, in.BuildCode, in.EnvironmentCode = "", "", "" },
			axis.InvalidRequest, []string{"testCase:required", "build:required", "environment:required"}},
		{"BUILD inexistente", func(in *usecase.CreateRunInput) { in.BuildCode = "BUILD-2026-0099" }, axis.InvalidRequest, []string{"build:invalid"}},
		{"ENV inexistente", func(in *usecase.CreateRunInput) { in.EnvironmentCode = "ENV-2026-0099" }, axis.InvalidRequest, []string{"environment:invalid"}},
		{"BUILD y ENV inexistentes", func(in *usecase.CreateRunInput) { in.BuildCode, in.EnvironmentCode = "x", "y" },
			axis.InvalidRequest, []string{"build:invalid", "environment:invalid"}},
		{"BUILD de otro sistema", func(in *usecase.CreateRunInput) { in.BuildCode = "BUILD-2026-0009" }, axis.InvalidRequest, []string{"build:invalid"}},
		{"ENV de otro sistema", func(in *usecase.CreateRunInput) { in.EnvironmentCode = "ENV-2026-0009" }, axis.InvalidRequest, []string{"environment:invalid"}},
		{"ENV retirado", func(in *usecase.CreateRunInput) { in.EnvironmentCode = "ENV-2026-0003" }, axis.InvalidRequest, []string{"environment:invalid"}},
		{"ENV de producción", func(in *usecase.CreateRunInput) { in.EnvironmentCode = "ENV-2026-0002" }, axis.InvalidRequest, []string{"environment:invalid"}},
		{"campos vacíos", func(in *usecase.CreateRunInput) { in.Result, in.ExecutedAt = "", time.Time{} },
			axis.InvalidRequest, []string{"result:required", "executedAt:required"}},
		{"resultado desconocido", func(in *usecase.CreateRunInput) { in.Result = "aprobado" }, axis.InvalidRequest, []string{"result:invalid"}},
		{"fecha futura", func(in *usecase.CreateRunInput) { in.ExecutedAt = t0.Add(time.Minute) }, axis.InvalidRequest, []string{"executedAt:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			in := baseRun
			tc.mutate(&in)
			_, err := uc.CreateRun.Execute(ctx, in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.runs) != 0 || len(w.audit) != 0 || w.sequences[shared.PrefixRun] != 0 {
				t.Error("un RUN rechazado no debe dejar nada, ni gastar un código")
			}
		})
	}

	uc, _ := setup(t)
	in := baseRun
	in.TestCaseCode = "TC-2026-0003"
	if _, err := uc.CreateRun.Execute(ctx, in); !errors.Is(err, domain.ErrTestCaseRetired) {
		t.Errorf("TC retirado: %v", err)
	}
}

func TestGetRun(t *testing.T) {
	uc, _ := setup(t)
	code := run(t, uc, "TC-2026-0001", "passed", "1.4.0")

	// Leer no exige permiso: ni siquiera recibe un actor.
	out, err := uc.GetRun.Execute(ctx, usecase.GetRunInput{Code: code})
	if err != nil || out.Code.String() != code || out.TestCase.Code != "TC-2026-0001" || out.Result != domain.ResultPassed {
		t.Errorf("GetRun = %+v, %v", out, err)
	}
	for _, code := range []string{"RUN-2026-0099", "no es un código"} {
		if _, err := uc.GetRun.Execute(ctx, usecase.GetRunInput{Code: code}); !errors.Is(err, usecase.ErrRunNotFound) {
			t.Errorf("RUN %q: %v", code, err)
		}
	}
}

func TestListRuns(t *testing.T) {
	uc, w := setup(t)
	first := run(t, uc, "TC-2026-0001", "failed", "1.4.0")
	second := run(t, uc, "TC-2026-0002", "passed", "1.4.0")
	third := run(t, uc, "TC-2026-0001", "passed", "1.5.0")
	other := run(t, uc, "TC-2026-0009", "passed", "1.5.0")

	cases := []struct {
		name       string
		in         usecase.ListRunsInput
		want       []string
		wantFilter port.RunFilter
	}{
		{"sin filtro", usecase.ListRunsInput{Page: 2, Size: 10}, []string{other, third, second, first}, port.RunFilter{Page: 2, Size: 10}},
		{"por versión", usecase.ListRunsInput{BuildVersion: "1.4.0"}, []string{second, first}, port.RunFilter{BuildVersion: "1.4.0"}},
		{"por TC", usecase.ListRunsInput{TestCaseCode: "TC-2026-0001"}, []string{third, first},
			port.RunFilter{FilterTestCases: true, TestCaseIds: []string{"t1"}}},
		{"por REQ", usecase.ListRunsInput{RequirementCode: "REQ-2026-0042"}, []string{third, second, first},
			port.RunFilter{FilterTestCases: true, TestCaseIds: []string{"t1", "t2", "t3"}}},
		{"por TC y su REQ", usecase.ListRunsInput{TestCaseCode: "TC-2026-0002", RequirementCode: "REQ-2026-0042"}, []string{second},
			port.RunFilter{FilterTestCases: true, TestCaseIds: []string{"t2"}}},
		{"por TC y un REQ ajeno", usecase.ListRunsInput{TestCaseCode: "TC-2026-0009", RequirementCode: "REQ-2026-0042"}, nil,
			port.RunFilter{FilterTestCases: true}},
		{"TC inexistente", usecase.ListRunsInput{TestCaseCode: "TC-2026-0099"}, nil, port.RunFilter{FilterTestCases: true}},
		{"REQ inexistente", usecase.ListRunsInput{RequirementCode: "REQ-2026-0099"}, nil, port.RunFilter{FilterTestCases: true}},
		{"TC de un REQ inexistente", usecase.ListRunsInput{TestCaseCode: "TC-2026-0001", RequirementCode: "REQ-2026-0099"}, nil,
			port.RunFilter{FilterTestCases: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := uc.ListRuns.Execute(ctx, tc.in)
			var got []string
			for _, s := range out.Runs {
				got = append(got, s.Code.String())
			}
			if err != nil || out.Total != len(tc.want) || !slices.Equal(got, tc.want) {
				t.Errorf("RUN = %v (total %d), %v; quiero %v", got, out.Total, err, tc.want)
			}
			f := w.lastFilter
			if f.FilterTestCases != tc.wantFilter.FilterTestCases || !slices.Equal(f.TestCaseIds, tc.wantFilter.TestCaseIds) ||
				f.BuildVersion != tc.wantFilter.BuildVersion || f.Page != tc.wantFilter.Page || f.Size != tc.wantFilter.Size {
				t.Errorf("filtro = %+v; quiero %+v", f, tc.wantFilter)
			}
		})
	}
}

func TestGetCoverage(t *testing.T) {
	uc, _ := setup(t)

	// Sin RUN, todos los TC activos están pendientes; el retirado no se exige.
	out, err := uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{RequirementCode: "REQ-2026-0042"})
	if err != nil || out.Required != 2 || !slices.Equal(out.Pending, []string{"TC-2026-0001", "TC-2026-0002"}) {
		t.Errorf("cobertura sin RUN = %+v, %v", out, err)
	}

	// Cuenta el último RUN de cada TC, no cualquiera aprobado.
	run(t, uc, "TC-2026-0001", "failed", "1.4.0")
	run(t, uc, "TC-2026-0001", "passed", "1.5.0")
	run(t, uc, "TC-2026-0002", "passed", "1.4.0")
	run(t, uc, "TC-2026-0002", "blocked", "1.5.0")
	out, err = uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{RequirementCode: "REQ-2026-0042"})
	if err != nil || out.Required != 2 || !slices.Equal(out.Pending, []string{"TC-2026-0002"}) {
		t.Errorf("cobertura = %+v, %v", out, err)
	}

	run(t, uc, "TC-2026-0002", "passed", "1.5.1")
	out, err = uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{RequirementCode: "REQ-2026-0042"})
	if err != nil || out.Required != 2 || len(out.Pending) != 0 {
		t.Errorf("cobertura completa = %+v, %v", out, err)
	}

	if _, err := uc.GetCoverage.Execute(ctx, usecase.GetCoverageInput{RequirementCode: "REQ-2026-0099"}); !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("REQ inexistente: %v", err)
	}
}

func TestCreateExploration(t *testing.T) {
	uc, w := setup(t)
	in := baseExploration
	in.Actor, in.ManualChanges = explorer, "  Parche a mano "
	out, err := uc.CreateExploration.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	saved := w.explorations["EXP-2026-0001"].Snapshot()
	if out.Code.String() != "EXP-2026-0001" || saved.Id != "id-1" || saved.Requirement != (domain.RequirementRef{Id: "r1", Code: "REQ-2026-0042", SystemKey: "SYPAGO"}) ||
		saved.BuildVersion != "1.4.0" || saved.EnvironmentName != "QA" || saved.ManualChanges != "Parche a mano" ||
		saved.BuildId != "b1" || saved.BuildCode != "BUILD-2026-0001" || saved.EnvironmentId != "e1" || saved.EnvironmentCode != "ENV-2026-0001" ||
		saved.Findings != "Nada raro" || saved.ExecutedBy != (domain.UserRef{Id: "u5", Name: "Usuario u5"}) ||
		!saved.ExecutedAt.Equal(t0.Add(-time.Hour)) || !saved.CreatedAt.Equal(t0) {
		t.Errorf("EXP guardado inesperado: %+v", saved)
	}
	want := port.AuditRecord{Actor: "u5", EntityType: port.EntityExploration, EntityId: "id-1", Code: out.Code, Action: port.AuditCreate}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Errorf("auditoría del alta: %+v", w.audit)
	}
}

// Explorar en producción sí vale: la regla «QA no ejecuta en producción» es del RUN oficial.
func TestCreateExplorationInProduction(t *testing.T) {
	uc, w := setup(t)
	in := baseExploration
	in.EnvironmentCode = "ENV-2026-0002"
	out, err := uc.CreateExploration.Execute(ctx, in)
	if err != nil || out.EnvironmentName != "Producción" || w.explorations["EXP-2026-0001"].Snapshot().EnvironmentCode != "ENV-2026-0002" {
		t.Errorf("EXP en producción = %+v, %v", out, err)
	}
}

func TestCreateExplorationErrors(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(in *usecase.CreateExplorationInput)
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", func(in *usecase.CreateExplorationInput) { in.Actor = dev }, axis.PermissionDenied, nil},
		{"el permiso de RUN no basta", func(in *usecase.CreateExplorationInput) { in.Actor = tester }, axis.PermissionDenied, nil},
		{"sin REQ", func(in *usecase.CreateExplorationInput) { in.RequirementCode = "" }, axis.InvalidRequest, []string{"requirement:required"}},
		{"REQ inexistente", func(in *usecase.CreateExplorationInput) { in.RequirementCode = "REQ-2026-0099" }, axis.InvalidRequest, []string{"requirement:invalid"}},
		{"actor sin usuario", func(in *usecase.CreateExplorationInput) { in.Actor = ghost }, axis.NotFoundResource, nil},
		{"sin REQ, BUILD ni ENV", func(in *usecase.CreateExplorationInput) {
			in.RequirementCode, in.BuildCode, in.EnvironmentCode = "", "", ""
		},
			axis.InvalidRequest, []string{"requirement:required", "build:required", "environment:required"}},
		{"BUILD inexistente", func(in *usecase.CreateExplorationInput) { in.BuildCode = "BUILD-2026-0099" }, axis.InvalidRequest, []string{"build:invalid"}},
		{"ENV inexistente", func(in *usecase.CreateExplorationInput) { in.EnvironmentCode = "ENV-2026-0099" }, axis.InvalidRequest, []string{"environment:invalid"}},
		{"BUILD y ENV de otro sistema", func(in *usecase.CreateExplorationInput) {
			in.BuildCode, in.EnvironmentCode = "BUILD-2026-0009", "ENV-2026-0009"
		},
			axis.InvalidRequest, []string{"build:invalid", "environment:invalid"}},
		{"ENV retirado", func(in *usecase.CreateExplorationInput) { in.EnvironmentCode = "ENV-2026-0003" }, axis.InvalidRequest, []string{"environment:invalid"}},
		{"sin fecha", func(in *usecase.CreateExplorationInput) { in.ExecutedAt = time.Time{} }, axis.InvalidRequest, []string{"executedAt:required"}},
		{"fecha futura", func(in *usecase.CreateExplorationInput) { in.ExecutedAt = t0.Add(time.Minute) }, axis.InvalidRequest, []string{"executedAt:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			in := baseExploration
			tc.mutate(&in)
			_, err := uc.CreateExploration.Execute(ctx, in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.explorations) != 0 || len(w.audit) != 0 || w.sequences[shared.PrefixExploration] != 0 {
				t.Error("un EXP rechazado no debe dejar nada, ni gastar un código")
			}
		})
	}
}

func TestUpdateExploration(t *testing.T) {
	uc, w := setup(t)
	code := explore(t, uc, "REQ-2026-0042")

	out, err := uc.UpdateExploration.Execute(ctx, usecase.UpdateExplorationInput{
		Actor: explorer, Code: code, ManualChanges: ptr(" Dos parches "), Findings: ptr("Falla al reintentar"),
	})
	if err != nil {
		t.Fatal(err)
	}
	saved := w.explorations[code].Snapshot()
	if out.ManualChanges != "Dos parches" || saved.ManualChanges != "Dos parches" || saved.Findings != "Falla al reintentar" {
		t.Errorf("edición no guardada: %+v / %+v", out, saved)
	}
	want := []port.AuditRecord{
		{Actor: "u5", EntityType: port.EntityExploration, EntityId: saved.Id, Code: saved.Code, Action: port.AuditUpdate,
			Field: "manualChanges", OldValue: "Parche a mano", NewValue: "Dos parches"},
		{Actor: "u5", EntityType: port.EntityExploration, EntityId: saved.Id, Code: saved.Code, Action: port.AuditUpdate,
			Field: "findings", OldValue: "Nada raro", NewValue: "Falla al reintentar"},
	}
	if !slices.Equal(w.audit[1:], want) {
		t.Errorf("auditoría de la edición: %+v", w.audit[1:])
	}

	// Editar con lo mismo, o sin campos, no escribe ni audita.
	if _, err := uc.UpdateExploration.Execute(ctx, usecase.UpdateExplorationInput{
		Actor: qa, Code: code, Findings: ptr("Falla al reintentar"),
	}); err != nil || len(w.audit) != 3 {
		t.Errorf("una edición sin cambios no se audita: %v, %d registros", err, len(w.audit))
	}
	if out, err := uc.UpdateExploration.Execute(ctx, usecase.UpdateExplorationInput{Actor: qa, Code: code}); err != nil ||
		out.Findings != "Falla al reintentar" || len(w.audit) != 3 {
		t.Errorf("una edición vacía devuelve el EXP tal cual: %+v, %v", out, err)
	}
}

func TestUpdateExplorationErrors(t *testing.T) {
	uc, w := setup(t)
	code := explore(t, uc, "REQ-2026-0042")
	cases := []struct {
		name     string
		actor    shared.Actor
		code     string
		wantCode axis.Code
	}{
		{"sin permiso", dev, code, axis.PermissionDenied},
		{"el permiso de RUN no basta", tester, code, axis.PermissionDenied},
		{"EXP inexistente", qa, "EXP-2026-0099", axis.NotFoundResource},
		{"código mal formado", qa, "exp 1", axis.NotFoundResource},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.UpdateExploration.Execute(ctx, usecase.UpdateExplorationInput{Actor: tc.actor, Code: tc.code, Findings: ptr("Otro")})
			if !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if w.explorations[code].Snapshot().Findings != "Nada raro" || len(w.audit) != 1 {
				t.Error("una edición rechazada no cambia el EXP ni se audita")
			}
		})
	}
	if _, err := uc.UpdateExploration.Execute(ctx, usecase.UpdateExplorationInput{Actor: qa, Code: "EXP-2026-0099"}); !errors.Is(err, usecase.ErrExplorationNotFound) {
		t.Errorf("EXP inexistente: %v", err)
	}
}

func TestGetAndListExplorations(t *testing.T) {
	uc, _ := setup(t)
	first, second, other := explore(t, uc, "REQ-2026-0042"), explore(t, uc, "REQ-2026-0042"), explore(t, uc, "REQ-2026-0043")

	out, err := uc.GetExploration.Execute(ctx, usecase.GetExplorationInput{Code: first})
	if err != nil || out.Code.String() != first || out.Requirement.Code != "REQ-2026-0042" {
		t.Errorf("GetExploration = %+v, %v", out, err)
	}
	for _, code := range []string{"EXP-2026-0099", "no es un código"} {
		if _, err := uc.GetExploration.Execute(ctx, usecase.GetExplorationInput{Code: code}); !errors.Is(err, usecase.ErrExplorationNotFound) {
			t.Errorf("EXP %q: %v", code, err)
		}
	}

	cases := []struct {
		name            string
		requirementCode string
		want            []string
	}{
		{"sin filtro", "", []string{other, second, first}},
		{"por REQ", "REQ-2026-0042", []string{second, first}},
		{"REQ sin EXP", "REQ-2026-0099", nil},
	}
	for _, tc := range cases {
		list, err := uc.ListExplorations.Execute(ctx, usecase.ListExplorationsInput{RequirementCode: tc.requirementCode, Page: 1, Size: 25})
		var got []string
		for _, s := range list.Explorations {
			got = append(got, s.Code.String())
		}
		if err != nil || list.Total != len(tc.want) || !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v (total %d), %v; quiero %v", tc.name, got, list.Total, err, tc.want)
		}
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	run(t, uc, "TC-2026-0001", "passed", "1.4.0")
	code := explore(t, uc, "REQ-2026-0042")
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")

	if _, err := uc.CreateRun.Execute(ctx, baseRun); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("registrar RUN: %v", err)
	}
	if _, err := uc.CreateExploration.Execute(ctx, baseExploration); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("registrar EXP: %v", err)
	}
	if _, err := uc.UpdateExploration.Execute(ctx, usecase.UpdateExplorationInput{
		Actor: qa, Code: code, Findings: ptr("Otro"),
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("editar EXP: %v", err)
	}

	if len(w.runs) != 1 || len(w.explorations) != 1 || w.explorations[code].Snapshot().Findings != "Nada raro" ||
		w.sequences[shared.PrefixRun] != 1 || w.sequences[shared.PrefixExploration] != 1 || len(w.audit) != 2 {
		t.Error("si la auditoría falla, nada debe quedar guardado")
	}
}
