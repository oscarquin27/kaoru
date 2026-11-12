package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now      = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	checksum = strings.Repeat("ab12", 16)
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

var (
	build       = domain.BuildRef{Id: "b1", Code: "BUILD-2026-0007", SystemKey: "SYPAGO", Version: "2.14.0", Checksum: checksum}
	environment = domain.EnvironmentRef{Id: "e1", Code: "ENV-2026-0002", Name: "QA-1", SystemKey: "SYPAGO"}
)

func params() domain.NewRunParams {
	return domain.NewRunParams{
		Id: "r1", Code: shared.NewCode(shared.PrefixRun, 2026, 533),
		TestCase: domain.TestCaseRef{Id: "t1", Code: "TC-2026-0110", Title: "Rechaza sobre el límite", SystemKey: "SYPAGO"},
		Build:    build, Environment: environment,
		Result: "passed", ExecutedBy: domain.UserRef{Id: "u1", Name: "Ana"}, ExecutedAt: now.Add(-time.Hour), Now: now,
	}
}

func TestNewRun(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewRunParams)
		wantFields []string
	}{
		{"válido: copia versión, checksum y ambiente", func(*domain.NewRunParams) {}, nil},
		{"ejecutado ahora mismo", func(p *domain.NewRunParams) { p.ExecutedAt = now }, nil},
		{"sin resultado ni fecha", func(p *domain.NewRunParams) { p.Result, p.ExecutedAt = "", time.Time{} },
			[]string{"result:required", "executedAt:required"}},
		{"BUILD de otro sistema", func(p *domain.NewRunParams) { p.Build.SystemKey = "SIMF" }, []string{"build:invalid"}},
		{"ambiente de otro sistema", func(p *domain.NewRunParams) { p.Environment.SystemKey = "SIMF" }, []string{"environment:invalid"}},
		{"ambiente retirado", func(p *domain.NewRunParams) { p.Environment.Retired = true }, []string{"environment:invalid"}},
		{"ambiente de producción", func(p *domain.NewRunParams) { p.Environment.Production = true }, []string{"environment:invalid"}},
		{"todo de otro sistema", func(p *domain.NewRunParams) { p.TestCase.SystemKey = "SGLPAR" }, []string{"build:invalid", "environment:invalid"}},
		{"resultado desconocido", func(p *domain.NewRunParams) { p.Result = "aprobado" }, []string{"result:invalid"}},
		{"fecha futura", func(p *domain.NewRunParams) { p.ExecutedAt = now.Add(time.Minute) }, []string{"executedAt:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			run, err := domain.NewRun(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := run.Snapshot()
			if s.BuildChecksum != checksum || s.BuildVersion != "2.14.0" || s.EnvironmentName != "QA-1" || s.CreatedAt != now ||
				s.BuildId != "b1" || s.BuildCode != "BUILD-2026-0007" || s.EnvironmentId != "e1" || s.EnvironmentCode != "ENV-2026-0002" {
				t.Errorf("RUN inesperado: %+v", s)
			}
		})
	}
}

func TestNewRunRejectsRetiredTestCase(t *testing.T) {
	p := params()
	p.TestCase.Retired = true
	if _, err := domain.NewRun(p); !errors.Is(err, domain.ErrTestCaseRetired) {
		t.Errorf("un TC retirado no admite RUN: %v", err)
	}
}

func TestCoverage(t *testing.T) {
	testCases := []domain.CoverageTestCase{
		{Id: "t1", Code: "TC-2026-0001"},
		{Id: "t2", Code: "TC-2026-0002"},
		{Id: "t3", Code: "TC-2026-0003"},
		{Id: "t4", Code: "TC-2026-0004", Retired: true},
	}
	cases := []struct {
		name         string
		latest       map[string]domain.Result
		wantRequired int
		wantPending  []string
	}{
		{"sin ejecuciones", nil, 3, []string{"TC-2026-0001", "TC-2026-0002", "TC-2026-0003"}},
		{"solo cuenta el último RUN aprobado", map[string]domain.Result{"t1": domain.ResultPassed, "t2": domain.ResultFailed, "t3": domain.ResultBlocked},
			3, []string{"TC-2026-0002", "TC-2026-0003"}},
		{"todo aprobado; el retirado no se exige", map[string]domain.Result{"t1": domain.ResultPassed, "t2": domain.ResultPassed, "t3": domain.ResultPassed, "t4": domain.ResultFailed}, 3, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			required, pending := domain.Coverage(testCases, tc.latest)
			if required != tc.wantRequired || !slices.Equal(pending, tc.wantPending) {
				t.Errorf("cobertura = %d, %v; quiero %d, %v", required, pending, tc.wantRequired, tc.wantPending)
			}
		})
	}
	if required, pending := domain.Coverage(nil, nil); required != 0 || pending != nil {
		t.Error("un REQ sin TC no exige ninguno")
	}
}

func TestExploration(t *testing.T) {
	p := domain.NewExplorationParams{
		Id: "e1", Code: shared.NewCode(shared.PrefixExploration, 2026, 7), Requirement: domain.RequirementRef{Id: "r1", Code: "REQ-2026-0042", SystemKey: "SYPAGO"},
		Build: build, Environment: environment, ExecutedBy: domain.UserRef{Id: "u1"}, ExecutedAt: now, Now: now,
	}
	exploration, err := domain.NewExploration(p)
	if err != nil {
		t.Fatal(err)
	}
	if s := exploration.Snapshot(); s.BuildVersion != "2.14.0" || s.EnvironmentName != "QA-1" ||
		s.BuildCode != "BUILD-2026-0007" || s.EnvironmentCode != "ENV-2026-0002" || s.BuildId != "b1" || s.EnvironmentId != "e1" {
		t.Errorf("EXP inesperado: %+v", s)
	}
	findings := " El saldo queda negativo "
	changes := exploration.Apply(nil, &findings, now.Add(time.Hour))
	if len(changes) != 1 || changes[0].Field != "findings" || exploration.Snapshot().Findings != "El saldo queda negativo" {
		t.Errorf("completar hallazgos: %+v", changes)
	}
	if changes := exploration.Apply(nil, &findings, now.Add(2*time.Hour)); len(changes) != 0 {
		t.Error("el mismo texto no es un cambio")
	}
	// Un EXP sí se puede registrar en producción, pero no en un ambiente retirado ni de otro sistema.
	p.Environment.Production = true
	if _, err := domain.NewExploration(p); err != nil {
		t.Errorf("EXP en producción: %v", err)
	}
	p.Build.SystemKey, p.Environment.Retired, p.ExecutedAt = "SIMF", true, now.Add(time.Hour)
	if _, err := domain.NewExploration(p); !slices.Equal(fieldsOf(t, err), []string{"build:invalid", "environment:invalid", "executedAt:invalid"}) {
		t.Errorf("EXP inválido: %v", err)
	}
}
