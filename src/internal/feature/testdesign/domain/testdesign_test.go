package domain_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	uc1 = domain.FunctionalCaseRef{Id: "f1", Code: "UC-2026-0001", SystemKey: "SYPAGO"}
	uc2 = domain.FunctionalCaseRef{Id: "f2", Code: "UC-2026-0002", SystemKey: "SYPAGO"}
	ucX = domain.FunctionalCaseRef{Id: "f9", Code: "UC-2026-0009", SystemKey: "SIMF"}
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

func params() domain.NewTestCaseParams {
	return domain.NewTestCaseParams{
		Id: "t1", Code: shared.NewCode(shared.PrefixTestCase, 2026, 110), SystemKey: "SYPAGO", SystemName: "SyPago",
		Title: " Rechaza sobre el límite ", Steps: " 1. Pagar ", ExpectedResult: " Rechazado ",
		FunctionalCases: []domain.FunctionalCaseRef{uc2, uc1, uc2}, CreatedBy: "u1", Now: now,
	}
}

func TestNewTestCase(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewTestCaseParams)
		wantFields []string
	}{
		{"válido", func(*domain.NewTestCaseParams) {}, nil},
		{"junta lo que falta", func(p *domain.NewTestCaseParams) {
			p.Title, p.Steps, p.ExpectedResult, p.FunctionalCases = "", " ", "", nil
		},
			[]string{"title:required", "steps:required", "expectedResult:required", "functionalCases:required"}},
		{"UC de otro sistema", func(p *domain.NewTestCaseParams) { p.FunctionalCases = []domain.FunctionalCaseRef{uc1, ucX} }, []string{"functionalCases:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			testCase, err := domain.NewTestCase(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := testCase.Snapshot()
			if s.Title != "Rechaza sobre el límite" || s.Steps != "1. Pagar" || s.RetiredAt != nil ||
				!slices.Equal(s.FunctionalCases, []domain.FunctionalCaseRef{uc1, uc2}) {
				t.Errorf("TC inesperado (los UC van sin repetir y en orden): %+v", s)
			}
		})
	}
}

func TestTestCaseApply(t *testing.T) {
	testCase, err := domain.NewTestCase(params())
	if err != nil {
		t.Fatal(err)
	}
	later := now.Add(time.Hour)
	yes, no, title := true, false, "Otro"

	changes, err := testCase.Apply(domain.TestCaseEdit{Title: &title, IsCritical: &yes, Retired: &yes,
		FunctionalCases: &[]domain.FunctionalCaseRef{uc1}}, later)
	if err != nil {
		t.Fatal(err)
	}
	var fields []string
	for _, c := range changes {
		fields = append(fields, c.Field+":"+c.Old+"→"+c.New)
	}
	want := []string{"title:Rechaza sobre el límite→Otro", "isCritical:false→true",
		"functionalCases:UC-2026-0001,UC-2026-0002→UC-2026-0001", "retired:false→true"}
	if !slices.Equal(fields, want) {
		t.Errorf("cambios = %v", fields)
	}
	if s := testCase.Snapshot(); s.RetiredAt == nil || *s.RetiredAt != later || s.UpdatedAt != later {
		t.Errorf("retirar guarda cuándo: %+v", s)
	}
	if changes, _ := testCase.Apply(domain.TestCaseEdit{Retired: &yes}, later.Add(time.Hour)); len(changes) != 0 {
		t.Error("retirar un TC ya retirado no cambia nada")
	}
	if changes, _ := testCase.Apply(domain.TestCaseEdit{Retired: &no}, later); len(changes) != 1 || testCase.Snapshot().RetiredAt != nil {
		t.Error("un TC retirado se puede volver a activar")
	}
	if _, err := testCase.Apply(domain.TestCaseEdit{FunctionalCases: &[]domain.FunctionalCaseRef{}}, later); !slices.Equal(fieldsOf(t, err), []string{"functionalCases:required"}) {
		t.Error("un TC no puede quedarse sin UC")
	}
}

func TestFunctionalCase(t *testing.T) {
	requirement := domain.RequirementRef{Id: "r1", Code: "REQ-2026-0042", SystemKey: "SYPAGO"}
	code := shared.NewCode(shared.PrefixFunctionalCase, 2026, 1)
	if _, err := domain.NewFunctionalCase("f1", code, requirement, " ", "", "u1", now); !slices.Equal(fieldsOf(t, err), []string{"title:required"}) {
		t.Error("un UC necesita título")
	}
	functionalCase, err := domain.NewFunctionalCase("f1", code, requirement, " Pago sobre el límite ", " x ", "u1", now)
	if err != nil || functionalCase.Snapshot().Title != "Pago sobre el límite" || functionalCase.Snapshot().TestCases == nil {
		t.Fatalf("UC inesperado: %+v, %v", functionalCase.Snapshot(), err)
	}
	same, other := "Pago sobre el límite", "Otro"
	if changes, _ := functionalCase.Apply(&same, nil, now); len(changes) != 0 {
		t.Error("el mismo título no es un cambio")
	}
	if changes, _ := functionalCase.Apply(&other, nil, now.Add(time.Hour)); len(changes) != 1 || changes[0].Field != "title" {
		t.Errorf("cambio de título: %+v", changes)
	}
}
