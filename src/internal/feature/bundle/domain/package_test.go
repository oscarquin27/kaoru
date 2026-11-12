package domain_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now   = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	later = now.Add(time.Hour)

	sypago     = domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"}
	inAssembly = domain.State{Id: "s1", Key: "in_assembly", Name: "En armado", IsInitial: true}
	inTesting  = domain.State{Id: "s2", Key: "in_testing", Name: "En pruebas"}
)

func params() domain.NewParams {
	return domain.NewParams{
		Id: "p1", Code: shared.NewCode(shared.PrefixPackage, 2026, 3), Title: "  Paquete de noviembre  ",
		Description: "  Límites y firma  ", System: sypago, State: inAssembly, CreatedBy: domain.UserRef{Id: "u1"}, Now: now,
	}
}

func newPackage(t *testing.T) domain.Package {
	t.Helper()
	pkg, err := domain.New(params())
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func fieldsOf(t *testing.T, err error) []string {
	t.Helper()
	var ge *axis.GeneralError
	if !errors.As(err, &ge) || ge.Code() != axis.InvalidRequest {
		t.Fatalf("esperaba INVALID_REQUEST, obtuve %v", err)
	}
	var out []string
	for _, f := range ge.Fields() {
		out = append(out, f.Field+":"+f.Code)
	}
	return out
}

func build(code, system string) domain.Build {
	return domain.Build{Ref: domain.BuildRef{Id: "id-" + code, Code: code, Version: "2.14.0", Checksum: strings.Repeat("a", 64)}, SystemKey: system}
}

func requirement(code, system, state string) domain.Requirement {
	return domain.Requirement{
		Ref:       domain.RequirementRef{Id: "id-" + code, Code: code, Title: "REQ " + code, State: domain.RequirementState{Key: state, Name: state}},
		SystemKey: system,
	}
}

func TestNew(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewParams)
		wantFields []string
	}{
		{"válido: recorta los textos y nace vacío", func(*domain.NewParams) {}, nil},
		{"sin descripción", func(p *domain.NewParams) { p.Description = "" }, nil},
		{"título de 200 caracteres", func(p *domain.NewParams) { p.Title = strings.Repeat("ñ", 200) }, nil},
		{"sin título", func(p *domain.NewParams) { p.Title = "   " }, []string{"title:required"}},
		{"título demasiado largo", func(p *domain.NewParams) { p.Title = strings.Repeat("x", 201) }, []string{"title:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			pkg, err := domain.New(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !reflect.DeepEqual(got, tc.wantFields) {
					t.Errorf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := pkg.Snapshot()
			if s.Title != strings.TrimSpace(p.Title) || s.Description != strings.TrimSpace(p.Description) || s.State != inAssembly ||
				s.System != sypago || len(s.Builds) != 0 || len(s.Requirements) != 0 || !s.CreatedAt.Equal(now) || !s.UpdatedAt.Equal(now) {
				t.Errorf("PKG inesperado: %+v", s)
			}
		})
	}
}

func TestApply(t *testing.T) {
	text := func(s string) *string { return &s }
	cases := []struct {
		name       string
		edit       domain.Edit
		want       []domain.FieldChange
		wantFields []string
	}{
		{"sin cambios", domain.Edit{}, nil, nil},
		{"lo mismo, con espacios, no es un cambio", domain.Edit{Title: text(" Paquete de noviembre "), Description: text("Límites y firma")}, nil, nil},
		{"título y descripción", domain.Edit{Title: text("Paquete de diciembre"), Description: text("")}, []domain.FieldChange{
			{Field: "title", Old: "Paquete de noviembre", New: "Paquete de diciembre"},
			{Field: "description", Old: "Límites y firma", New: ""},
		}, nil},
		{"título vacío", domain.Edit{Title: text(" "), Description: text("otra")}, nil, []string{"title:required"}},
		{"título demasiado largo", domain.Edit{Title: text(strings.Repeat("x", 201))}, nil, []string{"title:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkg := newPackage(t)
			before := pkg.Snapshot()
			changes, err := pkg.Apply(tc.edit, later)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !reflect.DeepEqual(got, tc.wantFields) {
					t.Errorf("campos = %v, quiero %v", got, tc.wantFields)
				}
				if !reflect.DeepEqual(pkg.Snapshot(), before) {
					t.Errorf("una edición rechazada no cambia el PKG: %+v", pkg.Snapshot())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(changes, tc.want) {
				t.Errorf("cambios = %+v, quiero %+v", changes, tc.want)
			}
			wantUpdated := now
			if len(tc.want) > 0 {
				wantUpdated = later
			}
			if !pkg.Snapshot().UpdatedAt.Equal(wantUpdated) {
				t.Errorf("updatedAt = %v, quiero %v", pkg.Snapshot().UpdatedAt, wantUpdated)
			}
		})
	}
}

func TestMoveTo(t *testing.T) {
	pkg := newPackage(t)
	change := pkg.MoveTo(inTesting, later)
	if change != (domain.FieldChange{Field: "state", Old: "in_assembly", New: "in_testing"}) {
		t.Errorf("cambio = %+v", change)
	}
	if s := pkg.Snapshot(); s.State != inTesting || !s.UpdatedAt.Equal(later) {
		t.Errorf("PKG = %+v", s)
	}
}

func codes(s domain.Snapshot) string {
	var out []string
	for _, b := range s.Builds {
		out = append(out, b.Code)
	}
	for _, r := range s.Requirements {
		out = append(out, r.Code+"="+r.State.Key)
	}
	return strings.Join(out, ",")
}

func TestContentChangesOnlyInAssembly(t *testing.T) {
	pkg := newPackage(t)

	// BUILD: del mismo sistema, sin repetir, por código.
	if added, err := pkg.AddBuild(build("BUILD-2026-0002", "SYPAGO"), later); err != nil || !added {
		t.Fatalf("agregar BUILD: %v %v", added, err)
	}
	if added, err := pkg.AddBuild(build("BUILD-2026-0001", "SYPAGO"), later); err != nil || !added {
		t.Fatalf("agregar otro BUILD: %v %v", added, err)
	}
	if !pkg.Snapshot().UpdatedAt.Equal(later) {
		t.Errorf("cambiar el contenido cambia updatedAt: %v", pkg.Snapshot().UpdatedAt)
	}
	stamp := later.Add(time.Hour)
	if added, err := pkg.AddBuild(build("BUILD-2026-0002", "SYPAGO"), stamp); err != nil || added {
		t.Errorf("agregar un BUILD que ya está no cambia nada: %v %v", added, err)
	}
	if added, err := pkg.AddBuild(build("BUILD-2026-0009", "SIMF"), stamp); !errors.Is(err, domain.ErrBuildOtherSystem) || added {
		t.Errorf("BUILD de otro sistema: %v %v", added, err)
	}

	// REQ: del mismo sistema, «Aprobado QA» o «En paquete».
	for _, tc := range []struct {
		req     domain.Requirement
		wantErr error
	}{
		{requirement("REQ-2026-0002", "SYPAGO", domain.RequirementInPackage), nil},
		{requirement("REQ-2026-0001", "SYPAGO", domain.RequirementApprovedQA), nil},
		{requirement("REQ-2026-0003", "SIMF", domain.RequirementApprovedQA), domain.ErrRequirementOtherSystem},
		{requirement("REQ-2026-0004", "SYPAGO", "in_testing"), domain.ErrRequirementNotApproved},
		{requirement("REQ-2026-0005", "SYPAGO", "released"), domain.ErrRequirementNotApproved},
	} {
		added, err := pkg.AddRequirement(tc.req, stamp)
		if !errors.Is(err, tc.wantErr) || added != (tc.wantErr == nil) {
			t.Errorf("agregar %s (%s): %v %v", tc.req.Ref.Code, tc.req.Ref.State.Key, added, err)
		}
		if tc.wantErr != nil && axis.CodeOf(err) != axis.InvalidAction {
			t.Errorf("%s: código %s", tc.req.Ref.Code, axis.CodeOf(err))
		}
	}
	// Uno que ya está no se vuelve a mirar: aunque haya cambiado de estado, no cambia nada.
	if added, err := pkg.AddRequirement(requirement("REQ-2026-0001", "SYPAGO", "released"), stamp); err != nil || added {
		t.Errorf("agregar un REQ que ya está: %v %v", added, err)
	}
	if got, want := codes(pkg.Snapshot()), "BUILD-2026-0001,BUILD-2026-0002,REQ-2026-0001=approved_qa,REQ-2026-0002=in_package"; got != want {
		t.Errorf("contenido = %s, quiero %s", got, want)
	}

	// Quitar: lo que no está no cambia nada.
	stamp = stamp.Add(time.Hour)
	if removed, err := pkg.RemoveBuild("BUILD-2026-0007", stamp); err != nil || removed {
		t.Errorf("quitar un BUILD que no está: %v %v", removed, err)
	}
	if removed, err := pkg.RemoveRequirement("REQ-2026-0007", stamp); err != nil || removed {
		t.Errorf("quitar un REQ que no está: %v %v", removed, err)
	}
	if pkg.Snapshot().UpdatedAt.Equal(stamp) {
		t.Error("lo que no cambia nada no toca updatedAt")
	}
	if removed, err := pkg.RemoveBuild("BUILD-2026-0001", stamp); err != nil || !removed {
		t.Errorf("quitar un BUILD: %v %v", removed, err)
	}
	if removed, err := pkg.RemoveRequirement("REQ-2026-0002", stamp); err != nil || !removed {
		t.Errorf("quitar un REQ: %v %v", removed, err)
	}
	if got, want := codes(pkg.Snapshot()), "BUILD-2026-0002,REQ-2026-0001=approved_qa"; got != want || !pkg.Snapshot().UpdatedAt.Equal(stamp) {
		t.Errorf("contenido = %s, quiero %s", got, want)
	}

	// Fuera de armado el contenido no cambia, ni siquiera para lo que ya está o no está.
	pkg.MoveTo(inTesting, stamp)
	before := pkg.Snapshot()
	if _, err := pkg.AddBuild(build("BUILD-2026-0003", "SYPAGO"), later); !errors.Is(err, domain.ErrNotInAssembly) {
		t.Errorf("agregar BUILD fuera de armado: %v", err)
	}
	if _, err := pkg.AddBuild(build("BUILD-2026-0002", "SYPAGO"), later); !errors.Is(err, domain.ErrNotInAssembly) {
		t.Errorf("agregar un BUILD que ya está, fuera de armado: %v", err)
	}
	if _, err := pkg.RemoveBuild("BUILD-2026-0002", later); !errors.Is(err, domain.ErrNotInAssembly) {
		t.Errorf("quitar BUILD fuera de armado: %v", err)
	}
	if _, err := pkg.AddRequirement(requirement("REQ-2026-0009", "SYPAGO", domain.RequirementApprovedQA), later); !errors.Is(err, domain.ErrNotInAssembly) {
		t.Errorf("agregar REQ fuera de armado: %v", err)
	}
	if _, err := pkg.RemoveRequirement("REQ-2026-0001", later); !errors.Is(err, domain.ErrNotInAssembly) || axis.CodeOf(err) != axis.InvalidAction {
		t.Errorf("quitar REQ fuera de armado: %v", err)
	}
	if !reflect.DeepEqual(pkg.Snapshot(), before) {
		t.Errorf("fuera de armado nada cambia: %+v", pkg.Snapshot())
	}
}

func TestRehydrateDoesNotShareItsContent(t *testing.T) {
	stored := domain.Snapshot{State: inAssembly, System: sypago, Builds: []domain.BuildRef{{Code: "BUILD-2026-0001"}}}
	pkg := domain.Rehydrate(stored)
	if _, err := pkg.RemoveBuild("BUILD-2026-0001", later); err != nil {
		t.Fatal(err)
	}
	if len(stored.Builds) != 1 || stored.Builds[0].Code != "BUILD-2026-0001" {
		t.Errorf("cambiar el PKG no debe tocar la lista con la que se armó: %+v", stored.Builds)
	}
}
