package domain

import (
	"slices"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)

func day(n float64) time.Time { return t0.Add(time.Duration(n * 24 * float64(time.Hour))) }

func ptr(f float64) *float64 { return &f }

func equalPtr(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestMedianAndPercentile(t *testing.T) {
	cases := []struct {
		values      []float64
		median, p90 *float64
	}{
		{nil, nil, nil},
		{[]float64{4}, ptr(4), ptr(4)},
		{[]float64{3, 1}, ptr(2), ptr(3)},
		{[]float64{5, 1, 3}, ptr(3), ptr(5)},
		{[]float64{10, 1, 2, 3, 4, 5, 6, 7, 8, 9}, ptr(5.5), ptr(9)},
	}
	for _, c := range cases {
		if got := Median(c.values); !equalPtr(got, c.median) {
			t.Errorf("Median(%v) = %v, quiero %v", c.values, got, c.median)
		}
		if got := Percentile(c.values, 0.9); !equalPtr(got, c.p90) {
			t.Errorf("Percentile(%v, 0.9) = %v, quiero %v", c.values, got, c.p90)
		}
	}
}

func TestPeriod(t *testing.T) {
	if got := DefaultFrom(time.Date(2026, 3, 15, 23, 0, 0, 0, time.UTC)); !got.Equal(time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("DefaultFrom = %s", got)
	}
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	open := Period{From: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)}
	if got := open.Months(now); !slices.Equal(got, []string{"2026-09", "2026-10", "2026-11"}) {
		t.Errorf("meses sin fin: %v", got)
	}
	// To es exclusivo: hasta el 1 de octubre no incluye octubre.
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	closed := Period{From: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC), To: &end}
	if got := closed.Months(now); !slices.Equal(got, []string{"2026-08", "2026-09"}) {
		t.Errorf("meses con fin: %v", got)
	}
}

func TestCoverageStatusAndOrder(t *testing.T) {
	row := func(code, risk string, required, passed int, minimum *float64) CoverageRow {
		return CoverageRow{Requirement: CodeTitle{Code: code}, RiskLevel: risk, Required: required, Passed: passed, MinimumPercent: minimum}
	}
	rows := []CoverageRow{
		row("REQ-1", "high", 4, 4, ptr(100)),  // cumple
		row("REQ-2", "", 3, 0, nil),           // sin riesgo, aunque tenga TC
		row("REQ-3", "low", 0, 0, ptr(50)),    // sin TC
		row("REQ-4", "medium", 3, 2, ptr(80)), // 66 % < 80 %
		row("REQ-5", "medium", 4, 3, ptr(75)), // justo en el mínimo: cumple
		row("REQ-6", "high", 2, 0, ptr(100)),  // 0 %
		row("REQ-7", "low", 2, 1, nil),        // sin regla activa: cumple
	}
	want := map[string]CoverageStatus{
		"REQ-1": CoverageMeets, "REQ-2": CoverageUnassessed, "REQ-3": CoverageNoTestCases, "REQ-4": CoverageBelow,
		"REQ-5": CoverageMeets, "REQ-6": CoverageBelow, "REQ-7": CoverageMeets,
	}
	for _, r := range rows {
		if got := r.Status(); got != want[r.Requirement.Code] {
			t.Errorf("%s: %s, quiero %s", r.Requirement.Code, got, want[r.Requirement.Code])
		}
	}
	if got := rows[3].Percent(); got != 66 {
		t.Errorf("2 de 3 = %d %%, quiero 66 (hacia abajo)", got)
	}
	SortCoverage(rows)
	var order []string
	for _, r := range rows {
		order = append(order, r.Requirement.Code)
	}
	if want := []string{"REQ-6", "REQ-4", "REQ-3", "REQ-2", "REQ-7", "REQ-5", "REQ-1"}; !slices.Equal(order, want) {
		t.Errorf("orden %v, quiero %v", order, want)
	}
	s := SummarizeCoverage(rows)
	if s.Requirements != 7 || s.ByStatus[CoverageBelow] != 2 || s.ByStatus[CoverageMeets] != 3 || s.Required != 18 || s.Passed != 10 {
		t.Errorf("resumen: %+v", s)
	}
	if !ValidCoverageStatus("below") || ValidCoverageStatus("bajo") {
		t.Error("ValidCoverageStatus")
	}
}

func TestMeasureDefectStates(t *testing.T) {
	states := []WorkflowState{
		{State: StateRef{Key: "open", Name: "Abierto"}},
		{State: StateRef{Key: "in_fix", Name: "En corrección"}},
		{State: StateRef{Key: "closed", Name: "Cerrado"}, IsFinal: true},
		{State: StateRef{Key: "reopened", Name: "Reabierto"}},
	}
	now := day(20)
	histories := []DefectHistory{
		// Abierto 2 días, en corrección 3, cerrado desde el día 5.
		{Id: "a", CreatedAt: day(0), Current: "closed", Transitions: []Transition{
			{At: day(2), From: "open", To: "in_fix"}, {At: day(5), From: "in_fix", To: "closed"},
		}},
		// Abierto 1 día, en corrección 1, reabierto 2, en corrección otra vez hasta hoy (día 20): 1 + 14 = 15.
		{Id: "b", CreatedAt: day(2), Current: "in_fix", Transitions: []Transition{
			{At: day(3), From: "open", To: "in_fix"}, {At: day(4), From: "in_fix", To: "reopened"},
			{At: day(6), From: "reopened", To: "in_fix"},
		}},
		// Nunca cambió: abierto desde el día 16.
		{Id: "c", CreatedAt: day(16), Current: "open"},
		// Pasó por un estado que el flujo ya no tiene.
		{Id: "d", CreatedAt: day(10), Current: "closed", Transitions: []Transition{
			{At: day(11), From: "open", To: "on_hold"}, {At: day(12), From: "on_hold", To: "closed"},
		}},
	}
	m := MeasureDefectStates(states, histories, now)
	byKey := map[string]StateMetric{}
	var keys []string
	for _, s := range m.States {
		byKey[s.State.Key] = s
		keys = append(keys, s.State.Key)
	}
	if !slices.Equal(keys, []string{"open", "in_fix", "closed", "reopened", "on_hold"}) || byKey["on_hold"].State.Name != "on_hold" {
		t.Fatalf("estados: %v", m.States)
	}
	check := func(key string, current, defects, visits int, median, p90 *float64) {
		t.Helper()
		s := byKey[key]
		if s.Current != current || s.Defects != defects || s.Visits != visits || !equalPtr(s.MedianDays, median) || !equalPtr(s.P90Days, p90) {
			t.Errorf("%s: %+v (mediana %v, p90 %v)", key, s, deref(s.MedianDays), deref(s.P90Days))
		}
	}
	check("open", 1, 4, 4, ptr(1.5), ptr(4))   // 2, 1, 4, 1
	check("in_fix", 1, 2, 3, ptr(9), ptr(15))  // 3 y 1 + 14
	check("reopened", 0, 1, 1, ptr(2), ptr(2)) // 2
	check("closed", 2, 2, 2, nil, nil)         // final: no se mide la estadía
	check("on_hold", 0, 1, 1, ptr(1), ptr(1))  // clave desconocida, medida igual
	if m.Total != 4 || m.Resolved != 2 || !equalPtr(m.ResolutionMedianDays, ptr(3.5)) || !equalPtr(m.ResolutionP90Days, ptr(5)) {
		t.Errorf("resolución: %+v (mediana %v)", m, deref(m.ResolutionMedianDays))
	}
}

func deref(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func TestSummarizeRegression(t *testing.T) {
	s := SummarizeRegression([]RegressionRow{{Proposed: 10, Final: 8, Added: 1, Removed: 3}, {Proposed: 4, Final: 5, Added: 1}})
	if s.Packages != 2 || s.Proposed != 14 || s.Final != 13 || s.Added != 2 || s.Removed != 3 || !equalPtr(s.MedianFinal, ptr(6.5)) {
		t.Errorf("resumen: %+v", s)
	}
	if empty := SummarizeRegression(nil); empty.MedianFinal != nil || empty.Packages != 0 {
		t.Errorf("sin PKG: %+v", empty)
	}
}

func TestMeasureEscaped(t *testing.T) {
	released := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	defect := func(code string, created time.Time, rel *time.Time) DefectRelease {
		return DefectRelease{Defect: CodeTitle{Code: code}, CreatedAt: created, ReleasedAt: rel}
	}
	defects := []DefectRelease{
		defect("DEF-4", time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), &released),  // escapó, 24 días después
		defect("DEF-3", time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), &released), // escapó
		defect("DEF-2", time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), &released),  // antes de producción
		defect("DEF-1", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), nil),         // su REQ nunca llegó
	}
	period := Period{From: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	m := MeasureEscaped(defects, period, time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC))
	if m.Defects != 4 || len(m.Escaped) != 2 || m.Escaped[0].Defect.Code != "DEF-4" || m.RatePercent != 50 {
		t.Errorf("escapados: %+v", m)
	}
	if got := m.Escaped[0].DaysAfterRelease(); got != 24 {
		t.Errorf("días después de llegar a producción = %v", got)
	}
	want := []MonthCount{{"2026-09", 1, 0}, {"2026-10", 2, 1}, {"2026-11", 1, 1}}
	if !slices.Equal(m.Months, want) {
		t.Errorf("meses %v, quiero %v", m.Months, want)
	}
	if third := MeasureEscaped(defects[1:], period, time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)); third.RatePercent != 33.3 {
		t.Errorf("1 de 3 = %v %%, quiero 33.3", third.RatePercent)
	}
	if none := MeasureEscaped(nil, period, released); none.RatePercent != 0 || none.Defects != 0 {
		t.Errorf("sin DEF: %+v", none)
	}
}
