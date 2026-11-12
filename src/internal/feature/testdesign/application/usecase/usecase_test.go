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
	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/application/usecase"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	functionalCases map[string]domain.FunctionalCase // por código
	testCases       map[string]domain.TestCase       // por código
	sequences       map[string]int                   // por prefijo
	components      map[string][]string              // id del TC → códigos de COMP
	audit           []port.AuditRecord
	auditErr        error
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	functionalCases, testCases := maps.Clone(w.functionalCases), maps.Clone(w.testCases)
	sequences, audit, components := maps.Clone(w.sequences), slices.Clone(w.audit), maps.Clone(w.components)
	if err := fn(ctx); err != nil {
		w.functionalCases, w.testCases, w.sequences, w.audit = functionalCases, testCases, sequences, audit
		w.components = components
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

var requirements = map[string]domain.RequirementRef{
	"REQ-2026-0042": {Id: "r1", Code: "REQ-2026-0042", SystemKey: "core"},
	"REQ-2026-0043": {Id: "r2", Code: "REQ-2026-0043", SystemKey: "web"},
}

// Find hace de port.Requirements.
func (w *world) Find(_ context.Context, code string) (domain.RequirementRef, error) {
	requirement, ok := requirements[code]
	if !ok {
		return domain.RequirementRef{}, notFound
	}
	return requirement, nil
}

func (w *world) NextCode(_ context.Context, prefix string, year int) (shared.Code, error) {
	w.sequences[prefix]++
	return shared.NewCode(prefix, year, w.sequences[prefix]), nil
}

func (w *world) FindSystem(_ context.Context, key string) (string, error) {
	systems := map[string]string{"core": "Core bancario", "web": "Banca web"}
	name, ok := systems[key]
	if !ok {
		return "", notFound
	}
	return name, nil
}

func (w *world) InsertFunctionalCase(_ context.Context, f domain.FunctionalCase) error {
	w.functionalCases[f.Snapshot().Code.String()] = f
	return nil
}

func (w *world) UpdateFunctionalCase(_ context.Context, f domain.FunctionalCase) error {
	w.functionalCases[f.Snapshot().Code.String()] = f
	return nil
}

func (w *world) FindFunctionalCase(_ context.Context, code shared.Code) (domain.FunctionalCase, error) {
	f, ok := w.functionalCases[code.String()]
	if !ok {
		return domain.FunctionalCase{}, notFound
	}
	return f, nil
}

func (w *world) ListFunctionalCases(_ context.Context, requirementId string) ([]domain.FunctionalCase, error) {
	var out []domain.FunctionalCase
	for _, f := range w.functionalCases {
		if f.Snapshot().Requirement.Id == requirementId {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(a, b domain.FunctionalCase) int {
		return strings.Compare(a.Snapshot().Code.String(), b.Snapshot().Code.String())
	})
	return out, nil
}

func (w *world) FunctionalCaseRefs(_ context.Context, codes []string) ([]domain.FunctionalCaseRef, error) {
	var out []domain.FunctionalCaseRef
	for _, code := range codes {
		if f, ok := w.functionalCases[code]; ok {
			s := f.Snapshot()
			out = append(out, domain.FunctionalCaseRef{Id: s.Id, Code: code, SystemKey: s.Requirement.SystemKey})
		}
	}
	return out, nil
}

func (w *world) InsertTestCase(_ context.Context, t domain.TestCase) error {
	w.testCases[t.Snapshot().Code.String()] = t
	return nil
}

func (w *world) UpdateTestCase(_ context.Context, t domain.TestCase) error {
	w.testCases[t.Snapshot().Code.String()] = t
	return nil
}

func (w *world) FindTestCase(_ context.Context, code shared.Code) (domain.TestCase, error) {
	t, ok := w.testCases[code.String()]
	if !ok {
		return domain.TestCase{}, notFound
	}
	return t, nil
}

// ListTestCases aplica el filtro en memoria y recuerda el último que recibió.
func (w *world) ListTestCases(_ context.Context, f port.TestCaseFilter) ([]domain.TestCase, int, error) {
	var out []domain.TestCase
	for _, t := range w.testCases {
		s := t.Snapshot()
		validates := slices.ContainsFunc(s.FunctionalCases, func(r domain.FunctionalCaseRef) bool {
			return w.functionalCases[r.Code].Snapshot().Requirement.Id == f.RequirementId
		})
		if (f.SystemKey != "" && s.SystemKey != f.SystemKey) || (f.Critical != nil && s.IsCritical != *f.Critical) ||
			!strings.Contains(s.Title, f.Query) || (f.RequirementId != "" && !validates) {
			continue
		}
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b domain.TestCase) int {
		return strings.Compare(a.Snapshot().Code.String(), b.Snapshot().Code.String())
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

type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	if permission == usecase.PermissionManage && slices.Contains(roles, "qa") {
		return nil
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
	t0  = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	qa  = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	dev = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	ctx = context.Background()
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{
		functionalCases: map[string]domain.FunctionalCase{}, testCases: map[string]domain.TestCase{}, sequences: map[string]int{},
		components: map[string][]string{},
	}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Requirements: w, Components: w, Permissions: fakePermissions{}, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

// functionalCase define un UC del REQ dado y devuelve su código.
func functionalCase(t *testing.T, uc *usecase.UseCases, requirementCode, title string) string {
	t.Helper()
	out, err := uc.CreateFunctionalCase.Execute(ctx, usecase.CreateFunctionalCaseInput{
		Actor: qa, RequirementCode: requirementCode, Title: title, Description: "Detalle",
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.Code.String()
}

// testCase define un TC del sistema core que valida los UC dados y devuelve su código.
func testCase(t *testing.T, uc *usecase.UseCases, title string, critical bool, functionalCases ...string) string {
	t.Helper()
	out, err := uc.CreateTestCase.Execute(ctx, usecase.CreateTestCaseInput{
		Actor: qa, Title: title, SystemKey: "core", Steps: "1. Entrar", ExpectedResult: "Entra",
		IsCritical: critical, FunctionalCases: functionalCases,
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

// changesOf resume los registros de auditoría como «campo:antes→después».
func changesOf(records []port.AuditRecord) []string {
	var out []string
	for _, a := range records {
		out = append(out, a.Field+":"+a.OldValue+"→"+a.NewValue)
	}
	return out
}

func ptr[T any](v T) *T { return &v }

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestCreateFunctionalCase(t *testing.T) {
	uc, w := setup(t)
	out, err := uc.CreateFunctionalCase.Execute(ctx, usecase.CreateFunctionalCaseInput{
		Actor: qa, RequirementCode: "REQ-2026-0042", Title: "  Consultar saldo ", Description: " Desde la app ",
	})
	if err != nil {
		t.Fatal(err)
	}
	saved := w.functionalCases["UC-2026-0001"].Snapshot()
	if out.Code.String() != "UC-2026-0001" || saved.Id != "id-1" || saved.Requirement.Id != "r1" || saved.Title != "Consultar saldo" ||
		saved.Description != "Desde la app" || saved.CreatedBy != "u2" || !saved.CreatedAt.Equal(t0) {
		t.Errorf("UC guardado inesperado: %+v", saved)
	}
	want := port.AuditRecord{
		Actor: "u2", EntityType: port.EntityFunctionalCase, EntityId: "id-1", Code: out.Code,
		Action: port.AuditCreate, NewValue: "Consultar saldo",
	}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Errorf("auditoría del alta: %+v", w.audit)
	}
}

func TestCreateFunctionalCaseErrors(t *testing.T) {
	cases := []struct {
		name       string
		actor      shared.Actor
		code       string
		title      string
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", dev, "REQ-2026-0042", "Consultar saldo", axis.PermissionDenied, nil},
		{"REQ inexistente", qa, "REQ-2026-0099", "Consultar saldo", axis.NotFoundResource, nil},
		{"sin título", qa, "REQ-2026-0042", "  ", axis.InvalidRequest, []string{"title:required"}},
		{"título demasiado largo", qa, "REQ-2026-0042", strings.Repeat("a", 201), axis.InvalidRequest, []string{"title:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			_, err := uc.CreateFunctionalCase.Execute(ctx, usecase.CreateFunctionalCaseInput{
				Actor: tc.actor, RequirementCode: tc.code, Title: tc.title,
			})
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.functionalCases) != 0 || len(w.audit) != 0 || w.sequences[shared.PrefixFunctionalCase] != 0 {
				t.Error("un alta rechazada no debe dejar nada, ni gastar un código")
			}
		})
	}
}

func TestUpdateFunctionalCase(t *testing.T) {
	uc, w := setup(t)
	code := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo")

	out, err := uc.UpdateFunctionalCase.Execute(ctx, usecase.UpdateFunctionalCaseInput{
		Actor: qa, Code: code, Title: ptr(" Consultar saldo disponible "), Description: ptr("Nuevo detalle"),
	})
	if err != nil {
		t.Fatal(err)
	}
	saved := w.functionalCases[code].Snapshot()
	if out.Title != "Consultar saldo disponible" || saved.Title != out.Title || saved.Description != "Nuevo detalle" {
		t.Errorf("edición no guardada: %+v / %+v", out, saved)
	}
	want := []string{"title:Consultar saldo→Consultar saldo disponible", "description:Detalle→Nuevo detalle"}
	if !slices.Equal(changesOf(w.audit[1:]), want) {
		t.Errorf("auditoría de la edición: %v", changesOf(w.audit[1:]))
	}
	last := w.audit[2]
	if last.Actor != "u2" || last.EntityType != port.EntityFunctionalCase || last.EntityId != saved.Id ||
		last.Code != saved.Code || last.Action != port.AuditUpdate {
		t.Errorf("registro de auditoría incompleto: %+v", last)
	}

	// Editar con lo mismo, o sin campos, no escribe ni audita.
	if _, err := uc.UpdateFunctionalCase.Execute(ctx, usecase.UpdateFunctionalCaseInput{
		Actor: qa, Code: code, Title: ptr("Consultar saldo disponible"),
	}); err != nil || len(w.audit) != 3 {
		t.Errorf("una edición sin cambios no se audita: %v, %d registros", err, len(w.audit))
	}
	if out, err := uc.UpdateFunctionalCase.Execute(ctx, usecase.UpdateFunctionalCaseInput{Actor: qa, Code: code}); err != nil ||
		out.Title != "Consultar saldo disponible" || len(w.audit) != 3 {
		t.Errorf("una edición vacía devuelve el UC tal cual: %+v, %v", out, err)
	}
}

func TestUpdateFunctionalCaseErrors(t *testing.T) {
	uc, w := setup(t)
	code := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo")
	cases := []struct {
		name       string
		in         usecase.UpdateFunctionalCaseInput
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", usecase.UpdateFunctionalCaseInput{Actor: dev, Code: code, Title: ptr("Otro")}, axis.PermissionDenied, nil},
		{"UC inexistente", usecase.UpdateFunctionalCaseInput{Actor: qa, Code: "UC-2026-0099", Title: ptr("Otro")}, axis.NotFoundResource, nil},
		{"código mal formado", usecase.UpdateFunctionalCaseInput{Actor: qa, Code: "uc 1", Title: ptr("Otro")}, axis.NotFoundResource, nil},
		{"título vacío", usecase.UpdateFunctionalCaseInput{Actor: qa, Code: code, Title: ptr(" "), Description: ptr("Otro")}, axis.InvalidRequest, []string{"title:required"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.UpdateFunctionalCase.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			saved := w.functionalCases[code].Snapshot()
			if saved.Title != "Consultar saldo" || saved.Description != "Detalle" || len(w.audit) != 1 {
				t.Error("una edición rechazada no cambia el UC ni se audita")
			}
		})
	}
	if _, err := uc.UpdateFunctionalCase.Execute(ctx, usecase.UpdateFunctionalCaseInput{Actor: qa, Code: "UC-2026-0099"}); !errors.Is(err, usecase.ErrFunctionalCaseNotFound) {
		t.Errorf("UC inexistente: %v", err)
	}
}

func TestCreateTestCase(t *testing.T) {
	uc, w := setup(t)
	first, second := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo"), functionalCase(t, uc, "REQ-2026-0042", "Transferir")

	// Los UC repetidos se cuentan una vez y quedan en orden de código.
	out, err := uc.CreateTestCase.Execute(ctx, usecase.CreateTestCaseInput{
		Actor: qa, Title: " Saldo tras transferir ", SystemKey: "core", Preconditions: " Cuenta con fondos ",
		Steps: "1. Transferir", ExpectedResult: "El saldo baja", IsCritical: true,
		FunctionalCases: []string{second, first, second},
	})
	if err != nil {
		t.Fatal(err)
	}
	saved := w.testCases["TC-2026-0001"].Snapshot()
	var validated []string
	for _, ref := range saved.FunctionalCases {
		validated = append(validated, ref.Code)
	}
	if out.Code.String() != "TC-2026-0001" || saved.Id != "id-3" || saved.SystemName != "Core bancario" ||
		saved.Title != "Saldo tras transferir" || saved.Preconditions != "Cuenta con fondos" || !saved.IsCritical ||
		saved.RetiredAt != nil || saved.CreatedBy != "u2" || !saved.CreatedAt.Equal(t0) || !slices.Equal(validated, []string{first, second}) {
		t.Errorf("TC guardado inesperado: %+v", saved)
	}
	want := port.AuditRecord{
		Actor: "u2", EntityType: port.EntityTestCase, EntityId: "id-3", Code: out.Code,
		Action: port.AuditCreate, NewValue: "Saldo tras transferir",
	}
	if len(w.audit) != 3 || w.audit[2] != want {
		t.Errorf("auditoría del alta: %+v", w.audit)
	}
}

func TestCreateTestCaseErrors(t *testing.T) {
	base := usecase.CreateTestCaseInput{
		Actor: qa, Title: "Saldo", SystemKey: "core", Steps: "1. Entrar", ExpectedResult: "Entra",
		FunctionalCases: []string{"UC-2026-0001"},
	}
	cases := []struct {
		name       string
		mutate     func(in *usecase.CreateTestCaseInput)
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", func(in *usecase.CreateTestCaseInput) { in.Actor = dev }, axis.PermissionDenied, nil},
		{"sin sistema", func(in *usecase.CreateTestCaseInput) { in.SystemKey = "" }, axis.InvalidRequest, []string{"system:required"}},
		{"sistema inexistente", func(in *usecase.CreateTestCaseInput) { in.SystemKey = "mainframe" }, axis.InvalidRequest, []string{"system:invalid"}},
		{"UC inexistente", func(in *usecase.CreateTestCaseInput) { in.FunctionalCases = []string{"UC-2026-0001", "UC-2026-0099"} }, axis.InvalidRequest, []string{"functionalCases:invalid"}},
		{"UC de otro sistema", func(in *usecase.CreateTestCaseInput) { in.FunctionalCases = []string{"UC-2026-0002"} }, axis.InvalidRequest, []string{"functionalCases:invalid"}},
		{"sin UC", func(in *usecase.CreateTestCaseInput) { in.FunctionalCases = nil }, axis.InvalidRequest, []string{"functionalCases:required"}},
		{"sin título, pasos ni resultado", func(in *usecase.CreateTestCaseInput) { in.Title, in.Steps, in.ExpectedResult = "", " ", "" }, axis.InvalidRequest, []string{"title:required", "steps:required", "expectedResult:required"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo") // UC-2026-0001, de core
			functionalCase(t, uc, "REQ-2026-0043", "Ver movimientos") // UC-2026-0002, de web
			in := base
			tc.mutate(&in)
			_, err := uc.CreateTestCase.Execute(ctx, in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.testCases) != 0 || len(w.audit) != 2 || w.sequences[shared.PrefixTestCase] != 0 {
				t.Error("un alta rechazada no debe dejar nada, ni gastar un código")
			}
		})
	}
}

func TestUpdateTestCase(t *testing.T) {
	uc, w := setup(t)
	first, second := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo"), functionalCase(t, uc, "REQ-2026-0042", "Transferir")
	code := testCase(t, uc, "Saldo", false, first)
	audited := len(w.audit)

	out, err := uc.UpdateTestCase.Execute(ctx, usecase.UpdateTestCaseInput{
		Actor: qa, Code: code, Title: ptr("Saldo disponible"), Preconditions: ptr("Con fondos"), Steps: ptr("1. Consultar"),
		ExpectedResult: ptr("Muestra el saldo"), IsCritical: ptr(true), FunctionalCases: ptr([]string{second, first}), Retired: ptr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	saved := w.testCases[code].Snapshot()
	if out.Title != "Saldo disponible" || saved.Title != out.Title || !saved.IsCritical || len(saved.FunctionalCases) != 2 ||
		saved.RetiredAt == nil || !saved.RetiredAt.Equal(t0) {
		t.Errorf("edición no guardada: %+v", saved)
	}
	want := []string{
		"title:Saldo→Saldo disponible", "preconditions:→Con fondos", "steps:1. Entrar→1. Consultar",
		"expectedResult:Entra→Muestra el saldo", "isCritical:false→true",
		"functionalCases:" + first + "→" + first + "," + second, "retired:false→true",
	}
	if !slices.Equal(changesOf(w.audit[audited:]), want) {
		t.Errorf("auditoría de la edición: %v", changesOf(w.audit[audited:]))
	}
	last := w.audit[len(w.audit)-1]
	if last.Actor != "u2" || last.EntityType != port.EntityTestCase || last.EntityId != saved.Id ||
		last.Code != saved.Code || last.Action != port.AuditUpdate {
		t.Errorf("registro de auditoría incompleto: %+v", last)
	}
	audited = len(w.audit)

	// Editar con lo mismo no escribe ni audita.
	if _, err := uc.UpdateTestCase.Execute(ctx, usecase.UpdateTestCaseInput{
		Actor: qa, Code: code, Title: ptr("Saldo disponible"), Retired: ptr(true), FunctionalCases: ptr([]string{first, second}),
	}); err != nil || len(w.audit) != audited {
		t.Errorf("una edición sin cambios no se audita: %v, %d registros", err, len(w.audit)-audited)
	}

	// Un TC retirado se puede volver a activar.
	out, err = uc.UpdateTestCase.Execute(ctx, usecase.UpdateTestCaseInput{Actor: qa, Code: code, Retired: ptr(false)})
	if err != nil || out.RetiredAt != nil || w.testCases[code].Snapshot().RetiredAt != nil ||
		!slices.Equal(changesOf(w.audit[audited:]), []string{"retired:true→false"}) {
		t.Errorf("reactivar: %+v, %v, %v", out, err, changesOf(w.audit[audited:]))
	}
}

func TestUpdateTestCaseErrors(t *testing.T) {
	uc, w := setup(t)
	core, web := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo"), functionalCase(t, uc, "REQ-2026-0043", "Ver movimientos")
	code := testCase(t, uc, "Saldo", false, core)
	audited := len(w.audit)
	cases := []struct {
		name       string
		in         usecase.UpdateTestCaseInput
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", usecase.UpdateTestCaseInput{Actor: dev, Code: code, Title: ptr("Otro")}, axis.PermissionDenied, nil},
		{"TC inexistente", usecase.UpdateTestCaseInput{Actor: qa, Code: "TC-2026-0099", Title: ptr("Otro")}, axis.NotFoundResource, nil},
		{"código mal formado", usecase.UpdateTestCaseInput{Actor: qa, Code: "tc 1", Title: ptr("Otro")}, axis.NotFoundResource, nil},
		{"UC inexistente", usecase.UpdateTestCaseInput{Actor: qa, Code: code, FunctionalCases: ptr([]string{"UC-2026-0099"})}, axis.InvalidRequest, []string{"functionalCases:invalid"}},
		{"UC de otro sistema", usecase.UpdateTestCaseInput{Actor: qa, Code: code, FunctionalCases: ptr([]string{web})}, axis.InvalidRequest, []string{"functionalCases:invalid"}},
		{"sin UC", usecase.UpdateTestCaseInput{Actor: qa, Code: code, FunctionalCases: ptr([]string{})}, axis.InvalidRequest, []string{"functionalCases:required"}},
		{"campos vacíos", usecase.UpdateTestCaseInput{Actor: qa, Code: code, Title: ptr(""), Steps: ptr(" "), ExpectedResult: ptr(""), IsCritical: ptr(true)}, axis.InvalidRequest, []string{"title:required", "steps:required", "expectedResult:required"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.UpdateTestCase.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			saved := w.testCases[code].Snapshot()
			if saved.Title != "Saldo" || saved.IsCritical || len(saved.FunctionalCases) != 1 || len(w.audit) != audited {
				t.Error("una edición rechazada no cambia el TC ni se audita")
			}
		})
	}
	if _, err := uc.UpdateTestCase.Execute(ctx, usecase.UpdateTestCaseInput{Actor: qa, Code: "TC-2026-0099"}); !errors.Is(err, usecase.ErrTestCaseNotFound) {
		t.Errorf("TC inexistente: %v", err)
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	functionalCaseCode := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo")
	testCaseCode := testCase(t, uc, "Saldo", false, functionalCaseCode)
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")

	if _, err := uc.CreateFunctionalCase.Execute(ctx, usecase.CreateFunctionalCaseInput{
		Actor: qa, RequirementCode: "REQ-2026-0042", Title: "Transferir",
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("crear UC: %v", err)
	}
	if _, err := uc.UpdateFunctionalCase.Execute(ctx, usecase.UpdateFunctionalCaseInput{
		Actor: qa, Code: functionalCaseCode, Title: ptr("Otro"),
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("editar UC: %v", err)
	}
	if _, err := uc.CreateTestCase.Execute(ctx, usecase.CreateTestCaseInput{
		Actor: qa, Title: "Otro", SystemKey: "core", Steps: "1", ExpectedResult: "2", FunctionalCases: []string{functionalCaseCode},
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("crear TC: %v", err)
	}
	if _, err := uc.UpdateTestCase.Execute(ctx, usecase.UpdateTestCaseInput{
		Actor: qa, Code: testCaseCode, Title: ptr("Otro"), Retired: ptr(true),
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("editar TC: %v", err)
	}

	savedCase, savedTest := w.functionalCases[functionalCaseCode].Snapshot(), w.testCases[testCaseCode].Snapshot()
	if len(w.functionalCases) != 1 || len(w.testCases) != 1 || savedCase.Title != "Consultar saldo" || savedTest.Title != "Saldo" ||
		savedTest.RetiredAt != nil || w.sequences[shared.PrefixFunctionalCase] != 1 || w.sequences[shared.PrefixTestCase] != 1 || len(w.audit) != 2 {
		t.Error("si la auditoría falla, nada debe quedar guardado")
	}
}

func TestGetAndListFunctionalCases(t *testing.T) {
	uc, _ := setup(t)
	first := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo")
	functionalCase(t, uc, "REQ-2026-0043", "Ver movimientos")
	third := functionalCase(t, uc, "REQ-2026-0042", "Transferir")

	// Leer no exige permiso: ni siquiera recibe un actor.
	out, err := uc.GetFunctionalCase.Execute(ctx, usecase.GetFunctionalCaseInput{Code: first})
	if err != nil || out.Title != "Consultar saldo" || out.Requirement.Code != "REQ-2026-0042" {
		t.Errorf("GetFunctionalCase = %+v, %v", out, err)
	}
	for _, code := range []string{"UC-2026-0099", "no es un código"} {
		if _, err := uc.GetFunctionalCase.Execute(ctx, usecase.GetFunctionalCaseInput{Code: code}); !errors.Is(err, usecase.ErrFunctionalCaseNotFound) {
			t.Errorf("UC %q: %v", code, err)
		}
	}

	list, err := uc.ListFunctionalCases.Execute(ctx, usecase.ListFunctionalCasesInput{RequirementCode: "REQ-2026-0042"})
	if err != nil || len(list) != 2 || list[0].Code.String() != first || list[1].Code.String() != third {
		t.Errorf("solo los UC del REQ, en orden de código: %+v, %v", list, err)
	}
	if _, err := uc.ListFunctionalCases.Execute(ctx, usecase.ListFunctionalCasesInput{RequirementCode: "REQ-2026-0099"}); !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("REQ inexistente: %v", err)
	}
}

func TestGetAndListTestCases(t *testing.T) {
	uc, _ := setup(t)
	core, other := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo"), functionalCase(t, uc, "REQ-2026-0043", "Ver movimientos")
	critical, plain := testCase(t, uc, "Saldo crítico", true, core), testCase(t, uc, "Saldo", false, core)
	web, err := uc.CreateTestCase.Execute(ctx, usecase.CreateTestCaseInput{
		Actor: qa, Title: "Movimientos", SystemKey: "web", Steps: "1", ExpectedResult: "2", FunctionalCases: []string{other},
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := uc.GetTestCase.Execute(ctx, usecase.GetTestCaseInput{Code: plain})
	if err != nil || out.Title != "Saldo" || out.SystemKey != "core" {
		t.Errorf("GetTestCase = %+v, %v", out, err)
	}
	for _, code := range []string{"TC-2026-0099", "no es un código"} {
		if _, err := uc.GetTestCase.Execute(ctx, usecase.GetTestCaseInput{Code: code}); !errors.Is(err, usecase.ErrTestCaseNotFound) {
			t.Errorf("TC %q: %v", code, err)
		}
	}

	cases := []struct {
		name string
		in   usecase.ListTestCasesInput
		want []string
	}{
		{"sin filtro", usecase.ListTestCasesInput{}, []string{critical, plain, web.Code.String()}},
		{"por sistema", usecase.ListTestCasesInput{SystemKey: "web"}, []string{web.Code.String()}},
		{"solo críticos", usecase.ListTestCasesInput{Critical: ptr(true)}, []string{critical}},
		{"por texto", usecase.ListTestCasesInput{Query: "Saldo"}, []string{critical, plain}},
		{"por REQ", usecase.ListTestCasesInput{RequirementCode: "REQ-2026-0042"}, []string{critical, plain}},
	}
	for _, tc := range cases {
		list, err := uc.ListTestCases.Execute(ctx, tc.in)
		var got []string
		for _, s := range list.TestCases {
			got = append(got, s.Code.String())
		}
		if err != nil || list.Total != len(tc.want) || !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v (total %d), %v; quiero %v", tc.name, got, list.Total, err, tc.want)
		}
	}
	if _, err := uc.ListTestCases.Execute(ctx, usecase.ListTestCasesInput{RequirementCode: "REQ-2026-0099"}); !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("REQ inexistente: %v", err)
	}
}

func TestListRequirementTestCases(t *testing.T) {
	uc, w := setup(t)
	core, other := functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo"), functionalCase(t, uc, "REQ-2026-0043", "Ver movimientos")
	active, retired := testCase(t, uc, "Saldo", false, core), testCase(t, uc, "Saldo viejo", false, core)
	if _, err := uc.CreateTestCase.Execute(ctx, usecase.CreateTestCaseInput{
		Actor: qa, Title: "Movimientos", SystemKey: "web", Steps: "1", ExpectedResult: "2", FunctionalCases: []string{other},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.UpdateTestCase.Execute(ctx, usecase.UpdateTestCaseInput{Actor: qa, Code: retired, Retired: ptr(true)}); err != nil {
		t.Fatal(err)
	}

	out, err := uc.ListRequirementTestCases.Execute(ctx, usecase.ListRequirementTestCasesInput{RequirementCode: "REQ-2026-0042"})
	want := []usecase.RequirementTestCase{
		{Id: w.testCases[active].Snapshot().Id, Code: active},
		{Id: w.testCases[retired].Snapshot().Id, Code: retired, Retired: true},
	}
	if err != nil || !slices.Equal(out, want) {
		t.Errorf("TC del REQ = %+v, %v; quiero %+v", out, err, want)
	}
	if _, err := uc.ListRequirementTestCases.Execute(ctx, usecase.ListRequirementTestCasesInput{RequirementCode: "REQ-2026-0099"}); !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("REQ inexistente: %v", err)
	}
}
