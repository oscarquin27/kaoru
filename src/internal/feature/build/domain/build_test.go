package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/build/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now      = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	checksum = strings.Repeat("ab12", 16)
	sypago   = domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"}
)

func params() domain.NewParams {
	return domain.NewParams{
		Id: "b1", Code: shared.NewCode(shared.PrefixBuild, 2026, 7), System: sypago,
		Version: " 2.14.0 ", Checksum: " " + strings.ToUpper(checksum) + " ",
		Requirements: []domain.RequirementRef{{Id: "r1", Code: "REQ-2026-0042", Title: "Límite diario", SystemKey: "SYPAGO"}},
		Migrations:   " 0042_limite.sql\n", Dependencies: " libpagos 3.1 ", Notes: " Compilado en CI ",
		BuiltBy: "u1", Now: now,
	}
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

func TestNew(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewParams)
		wantFields []string
	}{
		{"válido: normaliza el checksum y recorta los textos", func(*domain.NewParams) {}, nil},
		{"sin manifiesto", func(p *domain.NewParams) { p.Requirements, p.Migrations, p.Dependencies, p.Notes = nil, "", "", "" }, nil},
		{"versión de 80 caracteres", func(p *domain.NewParams) { p.Version = strings.Repeat("v", 80) }, nil},
		{"sin versión ni checksum", func(p *domain.NewParams) { p.Version, p.Checksum = " ", "" }, []string{"version:required", "checksum:required"}},
		{"versión demasiado larga", func(p *domain.NewParams) { p.Version = strings.Repeat("v", 81) }, []string{"version:invalid"}},
		{"checksum corto", func(p *domain.NewParams) { p.Checksum = "abc123" }, []string{"checksum:invalid"}},
		{"checksum que no es hexadecimal", func(p *domain.NewParams) { p.Checksum = strings.Repeat("zz", 32) }, []string{"checksum:invalid"}},
		{"REQ de otro sistema", func(p *domain.NewParams) {
			p.Requirements = append(p.Requirements,
				domain.RequirementRef{Id: "r2", Code: "REQ-2026-0050", SystemKey: "SIMF"},
				domain.RequirementRef{Id: "r3", Code: "REQ-2026-0051", SystemKey: "SIMF"})
		}, []string{"requirements:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			build, err := domain.New(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := build.Snapshot()
			if s.Checksum != checksum || s.Version != strings.TrimSpace(p.Version) || s.System != sypago ||
				s.BuiltBy.Id != "u1" || s.CreatedAt != now || s.Code.String() != "BUILD-2026-0007" {
				t.Errorf("BUILD inesperado: %+v", s)
			}
		})
	}

	build, err := domain.New(params())
	if err != nil {
		t.Fatal(err)
	}
	if s := build.Snapshot(); s.Migrations != "0042_limite.sql" || s.Dependencies != "libpagos 3.1" || s.Notes != "Compilado en CI" ||
		len(s.Requirements) != 1 || s.Requirements[0].Code != "REQ-2026-0042" {
		t.Errorf("manifiesto inesperado: %+v", s)
	}
}

func TestRehydrate(t *testing.T) {
	s := domain.Snapshot{Id: "b1", Version: "1.0", Checksum: checksum, BuiltBy: domain.UserRef{Id: "u1", Name: "Ana"}}
	if got := domain.Rehydrate(s).Snapshot(); got.Id != "b1" || got.BuiltBy.Name != "Ana" || got.Checksum != checksum {
		t.Errorf("Rehydrate = %+v", got)
	}
}
