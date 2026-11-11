package domain_test

import (
	"math"
	"testing"
	"time"

	"kaoru/internal/feature/traceability/domain"
)

var (
	t0     = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	states = []domain.StateRef{
		{Key: "registered", Name: "Registrado"}, {Key: "risk_assessed", Name: "Riesgo evaluado"},
		{Key: "in_testing", Name: "En pruebas"}, {Key: "approved_qa", Name: "Aprobado QA"},
	}
)

func day(n float64) time.Time { return t0.Add(time.Duration(n * 24 * float64(time.Hour))) }

func TestLifecycle(t *testing.T) {
	type step struct {
		status string
		days   float64
	}
	cases := []struct {
		name        string
		transitions []domain.Transition
		current     string
		blockedNext bool
		now         time.Time
		want        []step
	}{
		{"recién registrado", nil, "registered", false, day(0.5),
			[]step{{"current", 0.5}, {"pending", 0}, {"pending", 0}, {"pending", 0}}},
		{"la siguiente etapa está bloqueada", nil, "registered", true, day(2),
			[]step{{"current", 2}, {"blocked", 0}, {"pending", 0}, {"pending", 0}}},
		{"avanzó dos etapas", []domain.Transition{
			{At: day(1), From: "registered", To: "risk_assessed"}, {At: day(3), From: "risk_assessed", To: "in_testing"},
		}, "in_testing", true, day(7), []step{{"done", 1}, {"done", 2}, {"current", 4}, {"blocked", 0}}},
		{"fue y volvió de un estado lateral: el tiempo se acumula", []domain.Transition{
			{At: day(1), From: "registered", To: "risk_assessed"}, {At: day(2), From: "risk_assessed", To: "in_testing"},
			{At: day(3), From: "in_testing", To: "with_defects"}, {At: day(5), From: "with_defects", To: "in_testing"},
		}, "in_testing", false, day(6), []step{{"done", 1}, {"done", 1}, {"current", 2}, {"pending", 0}}},
		{"está en un estado lateral: ninguna etapa es la actual", []domain.Transition{
			{At: day(1), From: "registered", To: "risk_assessed"}, {At: day(2), From: "risk_assessed", To: "in_testing"},
			{At: day(3), From: "in_testing", To: "with_defects"},
		}, "with_defects", true, day(9), []step{{"done", 1}, {"done", 1}, {"done", 1}, {"pending", 0}}},
		{"llegó a la última etapa", []domain.Transition{
			{At: day(1), From: "registered", To: "risk_assessed"}, {At: day(2), From: "risk_assessed", To: "in_testing"},
			{At: day(4), From: "in_testing", To: "approved_qa"},
		}, "approved_qa", true, day(4), []step{{"done", 1}, {"done", 1}, {"done", 2}, {"current", 0}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := domain.Lifecycle(states, "registered", t0, tc.transitions, tc.current, tc.blockedNext, tc.now)
			if len(steps) != len(tc.want) {
				t.Fatalf("etapas = %d", len(steps))
			}
			for i, want := range tc.want {
				if steps[i].Status != want.status || math.Abs(steps[i].Days-want.days) > 1e-9 || steps[i].State != states[i] {
					t.Errorf("%s: %s %.2f días; quiero %s %.2f", states[i].Name, steps[i].Status, steps[i].Days, want.status, want.days)
				}
				if (steps[i].EnteredAt != nil) != (want.status == "done" || want.status == "current") {
					t.Errorf("%s: enteredAt solo existe en etapas visitadas", states[i].Name)
				}
			}
		})
	}
}

func TestNewIndicators(t *testing.T) {
	cases := []struct{ required, passed, want int }{{0, 0, 0}, {27, 21, 77}, {3, 3, 100}, {3, 0, 0}}
	for _, tc := range cases {
		if got := domain.NewIndicators(tc.required, tc.passed, 0, nil).CoveragePercent; got != tc.want {
			t.Errorf("cobertura de %d/%d = %d %%, quiero %d %%", tc.passed, tc.required, got, tc.want)
		}
	}
}
