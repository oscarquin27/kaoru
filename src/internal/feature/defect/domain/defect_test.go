package domain_test

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/defect/domain"
	shared "kaoru/internal/shared/domain"
)

var now = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)

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

func params() domain.NewParams {
	return domain.NewParams{
		Id: "d1", Code: shared.NewCode(shared.PrefixDefect, 2026, 91), Title: " Saldo negativo ",
		Description: " Al pagar sobre el límite ", Requirement: domain.RequirementRef{Id: "r1", Code: "REQ-2026-0042"},
		Run: domain.RunRef{Id: "x1", Code: "RUN-2026-0534", Result: "failed"}, Severity: "high",
		State: domain.State{Id: "s1", Key: "open", Name: "Abierto"}, CreatedBy: domain.UserRef{Id: "u1", Name: "Ana"}, Now: now,
	}
}

func TestNew(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewParams)
		wantFields []string
	}{
		{"válido con RUN fallido", func(*domain.NewParams) {}, nil},
		{"válido con RUN bloqueado", func(p *domain.NewParams) { p.Run.Result = "blocked" }, nil},
		{"un RUN aprobado no reproduce nada", func(p *domain.NewParams) { p.Run.Result = "passed" }, []string{"run:invalid"}},
		{"sin datos", func(p *domain.NewParams) { p.Title, p.Description, p.Severity = "", " ", "" },
			[]string{"title:required", "description:required", "severity:required"}},
		{"severidad en español", func(p *domain.NewParams) { p.Severity = "alta" }, []string{"severity:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			defect, err := domain.New(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if s := defect.Snapshot(); s.Title != "Saldo negativo" || s.State.Key != "open" || s.Severity != domain.SeverityHigh {
				t.Errorf("DEF inesperado: %+v", s)
			}
		})
	}
}

func TestApplyAndMove(t *testing.T) {
	defect, err := domain.New(params())
	if err != nil {
		t.Fatal(err)
	}
	severity := "medium"
	dev := &domain.UserRef{Id: "u3", Name: "Juan"}
	changes, err := defect.Apply(domain.Edit{Severity: &severity, Assignee: domain.AssigneeEdit{Set: true, Value: dev}}, now.Add(time.Hour))
	if err != nil || len(changes) != 2 || changes[0] != (domain.FieldChange{Field: "severity", Old: "high", New: "medium"}) ||
		changes[1] != (domain.FieldChange{Field: "assignee", Old: "", New: "u3"}) {
		t.Errorf("cambios = %+v, %v", changes, err)
	}
	bad := "urgente"
	if _, err := defect.Apply(domain.Edit{Severity: &bad}, now); !slices.Equal(fieldsOf(t, err), []string{"severity:invalid"}) {
		t.Error("severidad inválida")
	}
	moved := defect.MoveTo(domain.State{Id: "s2", Key: "triaged", Name: "Triado"}, now.Add(2*time.Hour))
	if moved != (domain.FieldChange{Field: "state", Old: "open", New: "triaged"}) || defect.Snapshot().State.Key != "triaged" {
		t.Errorf("transición = %+v", moved)
	}
}

func TestBlockers(t *testing.T) {
	notHigh := []domain.Rule{{Key: domain.RuleSeverityIsNot, Params: map[string]any{"severity": "high"}, Blocks: true}}
	high, _ := domain.New(params())
	p := params()
	p.Severity = "low"
	low, _ := domain.New(p)

	if got := high.Blockers(notHigh); !slices.Equal(got, []string{"Un DEF de severidad alta no admite este paso."}) {
		t.Errorf("un DEF alto no se difiere: %v", got)
	}
	if got := low.Blockers(notHigh); got != nil {
		t.Errorf("un DEF bajo sí: %v", got)
	}
	if got := low.Blockers([]domain.Rule{{Key: "inventada", Blocks: true}}); len(got) != 1 {
		t.Errorf("una regla desconocida bloquea: %v", got)
	}
	if got := high.Blockers([]domain.Rule{{Key: domain.RuleSeverityIsNot, Params: map[string]any{"severity": "high"}, Blocks: false}}); got != nil {
		t.Errorf("una regla que solo avisa no bloquea: %v", got)
	}
}

func TestDaysInState(t *testing.T) {
	defect, _ := domain.New(params())
	entered := now.Add(36 * time.Hour)
	cases := []struct {
		name    string
		entered *time.Time
		at      time.Time
		want    float64
	}{
		{"sin transiciones cuenta desde que se abrió", nil, now.Add(12 * time.Hour), 0.5},
		{"cuenta desde la última transición", &entered, now.Add(84 * time.Hour), 2},
		{"nunca es negativo", &entered, now, 0},
	}
	for _, tc := range cases {
		if got := defect.DaysInState(tc.entered, tc.at); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%s: %v días, quiero %v", tc.name, got, tc.want)
		}
	}
}
