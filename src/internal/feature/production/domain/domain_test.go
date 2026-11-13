package domain_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

var now = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)

// fieldsOf devuelve los errores de campo como "campo:código".
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

func TestNewTemplate(t *testing.T) {
	system := domain.System{Key: "SIMF", Name: "SIMF", DefaultRoute: "high"}
	long := make([]byte, 201)
	for i := range long {
		long[i] = 'a'
	}
	cases := []struct {
		name  string
		steps []domain.StepInput
		want  []string
	}{
		{"sin pasos", nil, []string{"steps:required"}},
		{"paso sin título", []domain.StepInput{{Title: "Respaldar la base"}, {Title: "  "}}, []string{"steps.1.title:required"}},
		{"título demasiado largo", []domain.StepInput{{Title: string(long)}}, []string{"steps.0.title:invalid"}},
		{"válida", []domain.StepInput{{Title: " Respaldar la base ", Description: " Con pg_dump. "}, {Title: "Desplegar"}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			template, err := domain.NewTemplate(domain.NewTemplateParams{Id: "t1", System: system, Version: 3, Steps: tc.steps, CreatedBy: "u1", Now: now})
			if got := fieldsOf(err); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("errores = %v, quiero %v (%v)", got, tc.want, err)
			}
			if tc.want != nil {
				return
			}
			want := domain.TemplateSnapshot{
				Id: "t1", System: system, Version: 3, CreatedBy: domain.UserRef{Id: "u1"}, CreatedAt: now,
				Steps: []domain.TemplateStep{
					{Position: 1, Title: "Respaldar la base", Description: "Con pg_dump."}, {Position: 2, Title: "Desplegar"},
				},
			}
			if got := template.Snapshot(); !reflect.DeepEqual(got, want) {
				t.Errorf("plantilla:\n got %+v\nwant %+v", got, want)
			}
		})
	}
}

func newChecklist(t *testing.T) domain.Checklist {
	t.Helper()
	checklist, err := domain.NewChecklist(domain.NewChecklistParams{
		Id: "c1", Code: shared.NewCode("CHK", 2026, 1),
		Package:     domain.PackageRef{Id: "p1", Code: "PKG-2026-0001", Title: "Paquete", SystemKey: "SIMF"},
		Environment: domain.EnvironmentRef{Id: "e1", Code: "ENV-2026-0001", Name: "Producción", Kind: "production", SystemKey: "SIMF"},
		Template: domain.TemplateSnapshot{Id: "t1", Version: 2, Steps: []domain.TemplateStep{
			{Position: 1, Title: "Respaldar"}, {Position: 2, Title: "Desplegar"},
		}},
		StartedBy: "u1", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return checklist
}

func TestNewChecklist(t *testing.T) {
	pkg := domain.PackageRef{Id: "p1", Code: "PKG-2026-0001", SystemKey: "SIMF"}
	cases := []struct {
		name string
		env  domain.EnvironmentRef
		want []string
	}{
		{"ambiente de prueba", domain.EnvironmentRef{Kind: "clean", SystemKey: "SIMF"}, []string{"environment:invalid"}},
		{"ambiente retirado", domain.EnvironmentRef{Kind: "production", SystemKey: "SIMF", Retired: true}, []string{"environment:invalid"}},
		{"ambiente de otro sistema", domain.EnvironmentRef{Kind: "production", SystemKey: "SYPAGO"}, []string{"environment:invalid"}},
		{"producción del mismo sistema", domain.EnvironmentRef{Kind: "production", SystemKey: "SIMF"}, nil},
	}
	for _, tc := range cases {
		_, err := domain.NewChecklist(domain.NewChecklistParams{Package: pkg, Environment: tc.env})
		if got := fieldsOf(err); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: errores = %v, quiero %v", tc.name, got, tc.want)
		}
	}

	checklist := newChecklist(t)
	s := checklist.Snapshot()
	if s.TemplateId != "t1" || s.TemplateVersion != 2 || len(s.Steps) != 2 || s.Steps[1].Title != "Desplegar" || s.Steps[0].Signature != nil {
		t.Errorf("el checklist nace con los pasos de la plantilla, sin firmar: %+v", s)
	}
	if checklist.SignedSteps() != 0 || checklist.Complete() {
		t.Errorf("recién abierto no está completo")
	}
}

func TestSign(t *testing.T) {
	checklist := newChecklist(t)
	if _, err := checklist.Sign(3, "u1", "", now); !errors.Is(err, domain.ErrStepNotFound) {
		t.Errorf("paso que no existe: %v", err)
	}
	// El orden entre pasos es libre.
	signature, err := checklist.Sign(2, "u5", "  Desplegado sin novedad.  ", now)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Signature{Position: 2, Comment: "Desplegado sin novedad.", SignedBy: domain.UserRef{Id: "u5"}, SignedAt: now}
	if signature != want || *checklist.Snapshot().Steps[1].Signature != want {
		t.Errorf("firma = %+v", signature)
	}
	if _, err := checklist.Sign(2, "u1", "otra", now.Add(time.Minute)); !errors.Is(err, domain.ErrStepAlreadySigned) {
		t.Errorf("un paso se firma una sola vez: %v", err)
	}
	if *checklist.Snapshot().Steps[1].Signature != want {
		t.Errorf("la firma no debía cambiar")
	}
	if checklist.SignedSteps() != 1 || checklist.Complete() {
		t.Errorf("con un paso de dos no está completo")
	}
	if _, err := checklist.Sign(1, "u1", "", now); err != nil || !checklist.Complete() {
		t.Errorf("con todos los pasos firmados está completo: %v", err)
	}
}

var (
	production = domain.EnvironmentRef{Id: "e1", Code: "ENV-2026-0001", Name: "Producción", Kind: "production"}
	pkg        = domain.PackageRef{Id: "p1", Code: "PKG-2026-0001", Title: "Paquete"}
	deployed   = domain.DeploymentRef{Id: "d1", Code: "DEP-2026-0001", Environment: production, Package: &pkg, ExecutedAt: now}
)

func TestDeploymentOfADiagnosticOrASmokeTest(t *testing.T) {
	cases := []struct {
		name       string
		deployment domain.DeploymentRef
		want       []string
	}{
		{"de un BUILD suelto", domain.DeploymentRef{Environment: production}, []string{"deployment:invalid"}},
		{"en un ambiente de prueba", domain.DeploymentRef{Environment: domain.EnvironmentRef{Kind: "clean"}, Package: &pkg}, []string{"deployment:invalid"}},
		{"de un PKG en producción", deployed, nil},
	}
	for _, tc := range cases {
		_, err := domain.NewDiagnostic(domain.NewDiagnosticParams{Deployment: tc.deployment})
		if got := fieldsOf(err); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("DIAG %s: errores = %v, quiero %v", tc.name, got, tc.want)
		}
		_, err = domain.NewSmokeTest(domain.NewSmokeTestParams{Deployment: tc.deployment, Result: "passed"})
		if got := fieldsOf(err); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("SMOKE %s: errores = %v, quiero %v", tc.name, got, tc.want)
		}
	}
}

func TestDiagnosticReview(t *testing.T) {
	submit := func() domain.Diagnostic {
		d, err := domain.NewDiagnostic(domain.NewDiagnosticParams{
			Id: "g1", Code: shared.NewCode("DIAG", 2026, 1), Deployment: deployed, Notes: "  Salida adjunta.  ", SubmittedBy: "u3", Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	d := submit()
	if s := d.Snapshot(); s.Package != pkg || s.Notes != "Salida adjunta." || s.Review != nil || s.SubmittedBy.Id != "u3" {
		t.Errorf("DIAG entregado: %+v", s)
	}
	cases := []struct {
		name, result, notes string
		want                []string
	}{
		{"sin resultado", "", "", []string{"result:required"}},
		{"resultado desconocido", "ok", "", []string{"result:invalid"}},
		{"no conforme sin explicación", "non_conforming", "  ", []string{"notes:required"}},
	}
	for _, tc := range cases {
		d := submit()
		_, err := d.Review(tc.result, tc.notes, "u2", now)
		if got := fieldsOf(err); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: errores = %v, quiero %v", tc.name, got, tc.want)
		}
		if d.Snapshot().Review != nil {
			t.Errorf("%s: una revisión que no vale no se guarda", tc.name)
		}
	}
	later := now.Add(time.Hour)
	review, err := d.Review("conforming", " Sin hallazgos. ", "u2", later)
	want := domain.Review{Result: domain.ReviewConforming, Notes: "Sin hallazgos.", ReviewedBy: domain.UserRef{Id: "u2"}, ReviewedAt: later}
	if err != nil || review != want || *d.Snapshot().Review != want {
		t.Errorf("revisión = %+v, %v", review, err)
	}
	if _, err := d.Review("non_conforming", "Cambié de opinión.", "u2", later); !errors.Is(err, domain.ErrAlreadyReviewed) {
		t.Errorf("un DIAG se revisa una sola vez: %v", err)
	}
	other := submit()
	if _, err := other.Review("non_conforming", "Faltan tablas.", "u2", later); err != nil {
		t.Errorf("no conforme con explicación: %v", err)
	}
}

func TestNewSmokeTest(t *testing.T) {
	cases := []struct {
		name, result, notes string
		want                []string
	}{
		{"sin resultado", "", "", []string{"result:required"}},
		{"resultado desconocido", "blocked", "", []string{"result:invalid"}},
		{"fallida sin explicación", "failed", "", []string{"notes:required"}},
		{"fallida con explicación", "failed", "No responde el login.", nil},
		{"aprobada", "passed", "", nil},
	}
	for _, tc := range cases {
		smoke, err := domain.NewSmokeTest(domain.NewSmokeTestParams{
			Id: "k1", Code: shared.NewCode("SMOKE", 2026, 1), Deployment: deployed, Result: tc.result, Notes: tc.notes, ExecutedBy: "u2", Now: now,
		})
		if got := fieldsOf(err); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: errores = %v, quiero %v", tc.name, got, tc.want)
		}
		if tc.want == nil {
			if s := smoke.Snapshot(); s.Package != pkg || string(s.Result) != tc.result || s.ExecutedBy.Id != "u2" || !s.CreatedAt.Equal(now) {
				t.Errorf("%s: SMOKE = %+v", tc.name, s)
			}
		}
	}
}
