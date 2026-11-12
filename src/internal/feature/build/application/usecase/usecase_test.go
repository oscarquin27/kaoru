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
	"kaoru/internal/feature/build/application/port"
	"kaoru/internal/feature/build/application/usecase"
	"kaoru/internal/feature/build/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	builds     map[string]domain.Build // por código
	sequence   int
	audit      []port.AuditRecord
	auditErr   error
	insertErr  error
	lastFilter port.Filter
	inTx       bool
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	builds, sequence, audit := maps.Clone(w.builds), w.sequence, slices.Clone(w.audit)
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.builds, w.sequence, w.audit = builds, sequence, audit
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

func (w *world) NextCode(_ context.Context, year int) (shared.Code, error) {
	w.sequence++
	return shared.NewCode(shared.PrefixBuild, year, w.sequence), nil
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

func (w *world) Insert(_ context.Context, b domain.Build) error {
	if !w.inTx {
		return errors.New("Insert fuera de una transacción")
	}
	if w.insertErr != nil {
		return w.insertErr
	}
	w.builds[b.Snapshot().Code.String()] = b
	return nil
}

// FindByCode devuelve el BUILD con el nombre de quien lo construyó, como lo hace la base.
func (w *world) FindByCode(_ context.Context, code shared.Code) (domain.Build, error) {
	b, ok := w.builds[code.String()]
	if !ok {
		return domain.Build{}, notFound
	}
	s := b.Snapshot()
	s.BuiltBy.Name = "Usuario " + string(s.BuiltBy.Id)
	return domain.Rehydrate(s), nil
}

func (w *world) FindCodeByChecksum(_ context.Context, checksum string) (shared.Code, error) {
	for _, b := range w.builds {
		if s := b.Snapshot(); s.Checksum == checksum {
			return s.Code, nil
		}
	}
	return shared.Code{}, notFound
}

// List devuelve del más reciente al más antiguo, con los filtros del puerto.
func (w *world) List(_ context.Context, f port.Filter) ([]domain.Build, int, error) {
	w.lastFilter = f
	var out []domain.Build
	for _, b := range w.builds {
		s := b.Snapshot()
		includes := slices.ContainsFunc(s.Requirements, func(r domain.RequirementRef) bool { return r.Code == f.RequirementCode })
		if (f.SystemKey != "" && s.System.Key != f.SystemKey) || (f.Query != "" && !strings.Contains(s.Version, f.Query)) ||
			(f.RequirementCode != "" && !includes) {
			continue
		}
		out = append(out, b)
	}
	slices.SortFunc(out, func(a, b domain.Build) int {
		return strings.Compare(b.Snapshot().Code.String(), a.Snapshot().Code.String())
	})
	return out, len(out), nil
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

// fakeRequirements: dos REQ de SyPago y uno de SIMF.
type fakeRequirements struct{}

func (fakeRequirements) Find(_ context.Context, code string) (domain.RequirementRef, error) {
	requirements := map[string]domain.RequirementRef{
		"REQ-2026-0042": {Id: "r1", Code: "REQ-2026-0042", Title: "Límite diario", SystemKey: "SYPAGO"},
		"REQ-2026-0043": {Id: "r2", Code: "REQ-2026-0043", Title: "Aviso de límite", SystemKey: "SYPAGO"},
		"REQ-2026-0050": {Id: "r3", Code: "REQ-2026-0050", Title: "Firma de mensajes", SystemKey: "SIMF"},
	}
	requirement, ok := requirements[code]
	if !ok {
		return domain.RequirementRef{}, notFound
	}
	return requirement, nil
}

// fakePermissions: solo dev registra BUILD.
type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	if permission == usecase.PermissionRegister && slices.Contains(roles, "dev") {
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
	t0       = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	dev      = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	qa       = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	ctx      = context.Background()
	checksum = strings.Repeat("a", 64)
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{builds: map[string]domain.Build{}}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Requirements: fakeRequirements{}, Permissions: fakePermissions{}, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

var base = usecase.RegisterBuildInput{
	Actor: dev, SystemKey: "SYPAGO", Version: "2.14.0", Checksum: checksum,
	RequirementCodes: []string{"REQ-2026-0042", "REQ-2026-0043"},
	Migrations:       "0042_limite.sql", Dependencies: "libpagos 3.1", Notes: "Compilado en CI",
}

// register registra un BUILD de ese sistema con esa versión, un checksum propio y esos REQ.
func register(t *testing.T, uc *usecase.UseCases, system, version, digit string, requirements ...string) string {
	t.Helper()
	in := base
	in.SystemKey, in.Version, in.Checksum, in.RequirementCodes = system, version, strings.Repeat(digit, 64), requirements
	out, err := uc.RegisterBuild.Execute(ctx, in)
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

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestRegisterBuild(t *testing.T) {
	uc, w := setup(t)
	in := base
	in.Checksum = " " + strings.ToUpper(checksum) + " "
	in.RequirementCodes = []string{"REQ-2026-0042", " REQ-2026-0043 ", "REQ-2026-0042"} // el repetido cuenta una vez
	out, err := uc.RegisterBuild.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	saved := w.builds["BUILD-2026-0001"].Snapshot()
	if out.Code.String() != "BUILD-2026-0001" || saved.Id != "id-1" || saved.System.Key != "SYPAGO" || saved.Version != "2.14.0" ||
		saved.Checksum != checksum || saved.Migrations != "0042_limite.sql" || saved.Dependencies != "libpagos 3.1" ||
		saved.Notes != "Compilado en CI" || !saved.CreatedAt.Equal(t0) {
		t.Errorf("BUILD guardado inesperado: %+v", saved)
	}
	var codes []string
	for _, r := range saved.Requirements {
		codes = append(codes, r.Id+"/"+r.Code)
	}
	if !slices.Equal(codes, []string{"r1/REQ-2026-0042", "r2/REQ-2026-0043"}) {
		t.Errorf("REQ del manifiesto = %v", codes)
	}
	// Quién lo construyó sale de la sesión; la respuesta trae su nombre.
	if saved.BuiltBy.Id != "u3" || out.BuiltBy != (domain.UserRef{Id: "u3", Name: "Usuario u3"}) {
		t.Errorf("builtBy = %+v / %+v", saved.BuiltBy, out.BuiltBy)
	}
	want := port.AuditRecord{Actor: "u3", BuildId: "id-1", Code: out.Code, Field: "checksum", NewValue: checksum}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Errorf("auditoría del alta: %+v", w.audit)
	}

	// Un BUILD sin REQ también vale, y toma el siguiente código.
	if second := register(t, uc, "SIMF", "4.2.0", "b"); second != "BUILD-2026-0002" || len(w.builds[second].Snapshot().Requirements) != 0 {
		t.Errorf("segundo BUILD = %s", second)
	}
}

func TestRegisterBuildErrors(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(in *usecase.RegisterBuildInput)
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", func(in *usecase.RegisterBuildInput) { in.Actor = qa }, axis.PermissionDenied, nil},
		{"sin sistema", func(in *usecase.RegisterBuildInput) { in.SystemKey = "" }, axis.InvalidRequest, []string{"system:required"}},
		{"sistema desconocido", func(in *usecase.RegisterBuildInput) { in.SystemKey = "OTRO" }, axis.InvalidRequest, []string{"system:invalid"}},
		{"REQ inexistente", func(in *usecase.RegisterBuildInput) { in.RequirementCodes = []string{"REQ-2026-0042", "REQ-2026-0099"} },
			axis.InvalidRequest, []string{"requirements:invalid"}},
		{"REQ mal formado", func(in *usecase.RegisterBuildInput) { in.RequirementCodes = []string{""} }, axis.InvalidRequest, []string{"requirements:invalid"}},
		{"REQ de otro sistema", func(in *usecase.RegisterBuildInput) { in.RequirementCodes = []string{"REQ-2026-0050"} },
			axis.InvalidRequest, []string{"requirements:invalid"}},
		{"sin versión ni checksum", func(in *usecase.RegisterBuildInput) { in.Version, in.Checksum = "", " " },
			axis.InvalidRequest, []string{"version:required", "checksum:required"}},
		{"checksum que no es SHA-256", func(in *usecase.RegisterBuildInput) { in.Checksum = "abc123" }, axis.InvalidRequest, []string{"checksum:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			in := base
			tc.mutate(&in)
			_, err := uc.RegisterBuild.Execute(ctx, in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.builds) != 0 || len(w.audit) != 0 || w.sequence != 0 {
				t.Error("un BUILD rechazado no debe dejar nada, ni gastar un código")
			}
		})
	}
}

// El mismo artefacto no se registra dos veces, aunque cambie la versión o el sistema.
func TestRegisterBuildRejectsRepeatedChecksum(t *testing.T) {
	uc, w := setup(t)
	first := register(t, uc, "SYPAGO", "2.14.0", "a")

	in := base
	in.SystemKey, in.Version, in.Checksum = "SIMF", "9.9.9", strings.ToUpper(checksum)
	in.RequirementCodes = nil
	_, err := uc.RegisterBuild.Execute(ctx, in)
	if !slices.Equal(fieldsOf(err), []string{"checksum:invalid"}) {
		t.Fatalf("checksum repetido: %v %v", err, fieldsOf(err))
	}
	var ge *axis.GeneralError
	if errors.As(err, &ge); !strings.Contains(ge.Fields()[0].Message, "ya está registrado como "+first) {
		t.Errorf("el mensaje debe decir qué BUILD lo tiene: %q", ge.Fields()[0].Message)
	}
	if len(w.builds) != 1 || len(w.audit) != 1 || w.sequence != 1 {
		t.Error("el rechazo no debe dejar nada, ni gastar un código")
	}

	// Si otro registro gana la carrera, la restricción única de la base también es un error del campo.
	w.insertErr = axis.New(axis.DuplicateResource, "Ya existe.")
	in.Checksum = strings.Repeat("f", 64)
	if _, err := uc.RegisterBuild.Execute(ctx, in); !slices.Equal(fieldsOf(err), []string{"checksum:invalid"}) || w.sequence != 1 {
		t.Errorf("checksum repetido en la base: %v %v", err, fieldsOf(err))
	}
}

func TestGetAndFindBuild(t *testing.T) {
	uc, _ := setup(t)
	code := register(t, uc, "SYPAGO", "2.14.0", "a", "REQ-2026-0042")

	// Leer no exige permiso: ni siquiera recibe un actor.
	out, err := uc.GetBuild.Execute(ctx, usecase.GetBuildInput{Code: code})
	if err != nil || out.Code.String() != code || out.Version != "2.14.0" || len(out.Requirements) != 1 || out.BuiltBy.Name != "Usuario u3" {
		t.Errorf("GetBuild = %+v, %v", out, err)
	}
	ref, err := uc.FindBuild.Execute(ctx, usecase.FindBuildInput{Code: code})
	want := usecase.BuildRef{Id: "id-1", Code: code, SystemKey: "SYPAGO", Version: "2.14.0", Checksum: checksum}
	if err != nil || ref != want {
		t.Errorf("FindBuild = %+v, %v", ref, err)
	}
	for _, code := range []string{"BUILD-2026-0099", "no es un código", ""} {
		if _, err := uc.GetBuild.Execute(ctx, usecase.GetBuildInput{Code: code}); !errors.Is(err, usecase.ErrBuildNotFound) {
			t.Errorf("GetBuild(%q): %v", code, err)
		}
		if _, err := uc.FindBuild.Execute(ctx, usecase.FindBuildInput{Code: code}); !errors.Is(err, usecase.ErrBuildNotFound) {
			t.Errorf("FindBuild(%q): %v", code, err)
		}
	}
}

func TestListBuilds(t *testing.T) {
	uc, w := setup(t)
	first := register(t, uc, "SYPAGO", "2.14.0", "a", "REQ-2026-0042")
	second := register(t, uc, "SYPAGO", "2.15.0", "b", "REQ-2026-0042", "REQ-2026-0043")
	third := register(t, uc, "SIMF", "4.2.0", "c", "REQ-2026-0050")

	cases := []struct {
		name string
		in   usecase.ListBuildsInput
		want []string
	}{
		{"sin filtro", usecase.ListBuildsInput{Page: 2, Size: 10}, []string{third, second, first}},
		{"por sistema", usecase.ListBuildsInput{SystemKey: "SYPAGO"}, []string{second, first}},
		{"por REQ", usecase.ListBuildsInput{RequirementCode: "REQ-2026-0043"}, []string{second}},
		{"por versión", usecase.ListBuildsInput{Query: "2.1"}, []string{second, first}},
		{"sin resultados", usecase.ListBuildsInput{SystemKey: "SIMF", Query: "2.1"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := uc.ListBuilds.Execute(ctx, tc.in)
			var got []string
			for _, s := range out.Builds {
				got = append(got, s.Code.String())
			}
			if err != nil || out.Total != len(tc.want) || !slices.Equal(got, tc.want) {
				t.Errorf("BUILD = %v (total %d), %v; quiero %v", got, out.Total, err, tc.want)
			}
			want := port.Filter{SystemKey: tc.in.SystemKey, RequirementCode: tc.in.RequirementCode, Query: tc.in.Query, Page: tc.in.Page, Size: tc.in.Size}
			if w.lastFilter != want {
				t.Errorf("filtro = %+v; quiero %+v", w.lastFilter, want)
			}
		})
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	register(t, uc, "SYPAGO", "2.14.0", "a")
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")

	in := base
	in.Checksum = strings.Repeat("b", 64)
	if _, err := uc.RegisterBuild.Execute(ctx, in); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("registrar BUILD: %v", err)
	}
	if len(w.builds) != 1 || w.sequence != 1 || len(w.audit) != 1 {
		t.Error("si la auditoría falla, nada debe quedar guardado")
	}
}
