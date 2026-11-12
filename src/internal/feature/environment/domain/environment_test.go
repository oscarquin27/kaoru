package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now    = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	sypago = domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"}
)

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

func ptr[T any](v T) *T { return &v }

func params() domain.NewParams {
	return domain.NewParams{
		Id: "e1", Code: shared.NewCode(shared.PrefixEnvironment, 2026, 3), System: sypago,
		Name: " QA-1 ", Kind: "baseline", OS: "linux", Notes: " Línea base ", CreatedBy: "u2", Now: now,
	}
}

func TestNew(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewParams)
		wantFields []string
	}{
		{"válido", func(*domain.NewParams) {}, nil},
		{"producción en Windows", func(p *domain.NewParams) { p.Kind, p.OS = "production", "windows" }, nil},
		{"limpio", func(p *domain.NewParams) { p.Kind = "clean" }, nil},
		{"offline", func(p *domain.NewParams) { p.Kind = "offline" }, nil},
		{"nombre de 80 caracteres", func(p *domain.NewParams) { p.Name = strings.Repeat("n", 80) }, nil},
		{"sin nada", func(p *domain.NewParams) { p.Name, p.Kind, p.OS = " ", "", "" }, []string{"name:required", "kind:required", "os:required"}},
		{"nombre demasiado largo", func(p *domain.NewParams) { p.Name = strings.Repeat("n", 81) }, []string{"name:invalid"}},
		{"tipo desconocido", func(p *domain.NewParams) { p.Kind = "staging" }, []string{"kind:invalid"}},
		{"sistema operativo desconocido", func(p *domain.NewParams) { p.OS = "macos" }, []string{"os:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			environment, err := domain.New(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := environment.Snapshot()
			if s.Name != strings.TrimSpace(p.Name) || string(s.Kind) != p.Kind || string(s.OS) != p.OS || s.Notes != "Línea base" ||
				s.RetiredAt != nil || s.System != sypago || s.CreatedBy.Id != "u2" || s.CreatedAt != now || s.UpdatedAt != now {
				t.Errorf("ENV inesperado: %+v", s)
			}
		})
	}
}

func TestApply(t *testing.T) {
	later := now.Add(time.Hour)
	cases := []struct {
		name        string
		edit        domain.Edit
		wantFields  []string
		wantChanges []domain.FieldChange
	}{
		{"sin campos", domain.Edit{}, nil, nil},
		{"lo mismo no es un cambio", domain.Edit{Name: ptr("QA-1 "), Kind: ptr("baseline"), Retired: ptr(false)}, nil, nil},
		{"todos los campos", domain.Edit{Name: ptr(" QA-2 "), Kind: ptr("clean"), OS: ptr("windows"), Notes: ptr(" Recién instalado ")}, nil,
			[]domain.FieldChange{{Field: "name", Old: "QA-1", New: "QA-2"}, {Field: "kind", Old: "baseline", New: "clean"},
				{Field: "os", Old: "linux", New: "windows"}, {Field: "notes", Old: "Línea base", New: "Recién instalado"}}},
		{"retirar", domain.Edit{Retired: ptr(true)}, nil, []domain.FieldChange{{Field: "retired", Old: "false", New: "true"}}},
		{"valores que no valen", domain.Edit{Name: ptr(""), Kind: ptr("staging"), OS: ptr("")}, []string{"name:required", "kind:invalid", "os:required"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			environment, err := domain.New(params())
			if err != nil {
				t.Fatal(err)
			}
			before := environment.Snapshot()
			changes, err := environment.Apply(tc.edit, later)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(changes, tc.wantChanges) {
				t.Errorf("cambios = %+v, quiero %+v", changes, tc.wantChanges)
			}
			after := environment.Snapshot()
			if len(tc.wantChanges) == 0 && after != before {
				t.Errorf("sin cambios, el ENV no se toca: %+v", after)
			}
			if len(tc.wantChanges) > 0 && after.UpdatedAt != later {
				t.Errorf("updatedAt = %v", after.UpdatedAt)
			}
		})
	}

	// Retirar guarda cuándo; volver a ponerlo en uso lo borra.
	environment, _ := domain.New(params())
	if _, err := environment.Apply(domain.Edit{Retired: ptr(true)}, later); err != nil || environment.Snapshot().RetiredAt == nil ||
		!environment.Snapshot().RetiredAt.Equal(later) {
		t.Errorf("retirar: %+v, %v", environment.Snapshot(), err)
	}
	if changes, _ := environment.Apply(domain.Edit{Retired: ptr(true)}, later.Add(time.Hour)); len(changes) != 0 {
		t.Error("retirar un ambiente retirado no es un cambio")
	}
	changes, err := environment.Apply(domain.Edit{Retired: ptr(false)}, later.Add(time.Hour))
	if err != nil || len(changes) != 1 || changes[0] != (domain.FieldChange{Field: "retired", Old: "true", New: "false"}) ||
		environment.Snapshot().RetiredAt != nil {
		t.Errorf("volver a usar: %+v, %v", changes, err)
	}
}

func deploymentParams(t *testing.T) domain.NewDeploymentParams {
	t.Helper()
	environment, err := domain.New(params())
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewDeploymentParams{
		Id: "d1", Code: shared.NewCode(shared.PrefixDeployment, 2026, 9), Environment: environment.Snapshot(),
		Build:  &domain.BuildRef{Id: "b1", Code: "BUILD-2026-0007", Version: "2.14.0", Checksum: strings.Repeat("a", 64), SystemKey: "SYPAGO"},
		Result: "success", Notes: " Sin novedades ", ExecutedBy: "u2", ExecutedAt: now.Add(-time.Hour), Now: now,
	}
}

func TestNewDeployment(t *testing.T) {
	pkg := &domain.PackageRef{Id: "p1", Code: "PKG-2026-0001", Title: "Paquete de noviembre", SystemKey: "SYPAGO"}
	retired := now.Add(-24 * time.Hour)
	cases := []struct {
		name       string
		mutate     func(p *domain.NewDeploymentParams)
		wantFields []string
	}{
		{"un BUILD", func(*domain.NewDeploymentParams) {}, nil},
		{"un PKG", func(p *domain.NewDeploymentParams) { p.Build, p.Package = nil, pkg }, nil},
		{"fallido, ahora mismo", func(p *domain.NewDeploymentParams) { p.Result, p.ExecutedAt = "failed", now }, nil},
		{"revertido", func(p *domain.NewDeploymentParams) { p.Result = "rolled_back" }, nil},
		{"ni BUILD ni PKG", func(p *domain.NewDeploymentParams) { p.Build = nil }, []string{"build:required"}},
		{"BUILD y PKG", func(p *domain.NewDeploymentParams) { p.Package = pkg }, []string{"package:invalid"}},
		{"BUILD de otro sistema", func(p *domain.NewDeploymentParams) { p.Build.SystemKey = "SIMF" }, []string{"build:invalid"}},
		{"PKG de otro sistema", func(p *domain.NewDeploymentParams) {
			p.Build, p.Package = nil, &domain.PackageRef{Id: "p2", Code: "PKG-2026-0002", SystemKey: "SIMF"}
		}, []string{"package:invalid"}},
		{"ambiente retirado", func(p *domain.NewDeploymentParams) { p.Environment.RetiredAt = &retired }, []string{"environment:invalid"}},
		{"sin resultado ni fecha", func(p *domain.NewDeploymentParams) { p.Result, p.ExecutedAt = "", time.Time{} },
			[]string{"result:required", "executedAt:required"}},
		{"resultado desconocido", func(p *domain.NewDeploymentParams) { p.Result = "ok" }, []string{"result:invalid"}},
		{"fecha futura", func(p *domain.NewDeploymentParams) { p.ExecutedAt = now.Add(time.Minute) }, []string{"executedAt:invalid"}},
		{"todo mal", func(p *domain.NewDeploymentParams) {
			p.Environment.RetiredAt, p.Build, p.Result, p.ExecutedAt = &retired, nil, "ok", now.Add(time.Minute)
		}, []string{"environment:invalid", "build:required", "result:invalid", "executedAt:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := deploymentParams(t)
			tc.mutate(&p)
			deployment, err := domain.NewDeployment(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := deployment.Snapshot()
			wantEnv := domain.EnvironmentRef{Id: "e1", Code: "ENV-2026-0003", Name: "QA-1", Kind: domain.KindBaseline}
			if s.Environment != wantEnv || s.Build != p.Build || s.Package != p.Package || string(s.Result) != p.Result ||
				s.Notes != "Sin novedades" || s.ExecutedBy.Id != "u2" || !s.ExecutedAt.Equal(p.ExecutedAt) || s.CreatedAt != now ||
				s.Code.String() != "DEP-2026-0009" {
				t.Errorf("DEP inesperado: %+v", s)
			}
		})
	}
}
