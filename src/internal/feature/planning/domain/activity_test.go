package domain_test

import (
	"slices"
	"testing"
	"time"

	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	ana  = domain.UserRef{Id: "u-ana", Name: "Ana Rojas"}
	luis = domain.UserRef{Id: "u-luis", Name: "Luis Paz"}
	chg1 = &domain.ChangeRef{Id: "c1", Code: "CHG-2026-0001", Title: "Límite diario", System: "SyPago"}
	chg2 = &domain.ChangeRef{Id: "c2", Code: "CHG-2026-0002", Title: "Mensajería", System: "SIMF"}
)

func activity(t *testing.T, code, kind, start, end string, change *domain.ChangeRef, ops ...domain.UserRef) domain.Activity {
	t.Helper()
	a, err := domain.NewActivity(domain.NewActivityParams{
		Id: code, Code: code, Kind: kind, Title: "Actividad " + code, Change: change, Start: start, End: end,
		Operators: ops, CreatedBy: "u1", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func day(s string) time.Time {
	d, err := time.Parse(domain.DateLayout, s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestNewActivityValidates(t *testing.T) {
	cases := map[string]struct {
		p    domain.NewActivityParams
		want []string
	}{
		"válida sin CHG": {domain.NewActivityParams{Kind: "regression", Title: "Regresión", Start: "2026-11-16", End: "2026-11-16"}, nil},
		"certificación sin CHG": {
			domain.NewActivityParams{Kind: "certification", Title: "Cert", Start: "2026-11-16", End: "2026-11-20"},
			[]string{"change:required"},
		},
		"todo vacío": {
			domain.NewActivityParams{},
			[]string{"kind:invalid", "title:required", "startDate:required", "endDate:required"},
		},
		"fin antes del inicio": {
			domain.NewActivityParams{Kind: "other", Title: "x", Start: "2026-11-20", End: "2026-11-16"},
			[]string{"endDate:invalid"},
		},
		"más de un año": {
			domain.NewActivityParams{Kind: "other", Title: "x", Start: "2026-01-01", End: "2027-01-02"},
			[]string{"endDate:invalid"},
		},
		"fecha mal escrita": {
			domain.NewActivityParams{Kind: "other", Title: "x", Start: "16/11/2026", End: "2026-11-20"},
			[]string{"startDate:invalid"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.p.Now = now
			_, err := domain.NewActivity(tc.p)
			if got := fields(err); !slices.Equal(got, tc.want) {
				t.Errorf("errores %v, quería %v", got, tc.want)
			}
		})
	}
}

func TestActivityOperatorsAreUniqueAndSorted(t *testing.T) {
	a := activity(t, "ACT-1", "other", "2026-11-16", "2026-11-20", nil, luis, ana, luis)
	if got := a.Snapshot().Operators; !slices.Equal(got, []domain.UserRef{ana, luis}) {
		t.Errorf("operadores %v", got)
	}
}

func TestActivityStatus(t *testing.T) {
	cases := map[string]struct {
		start, end string
		want       domain.Status
	}{
		"empieza mañana":  {"2026-11-13", "2026-11-20", domain.StatusPlanned},
		"empieza hoy":     {"2026-11-12", "2026-11-20", domain.StatusActive},
		"termina hoy":     {"2026-11-01", "2026-11-12", domain.StatusActive},
		"terminó ayer":    {"2026-11-01", "2026-11-11", domain.StatusDone},
		"un solo día hoy": {"2026-11-12", "2026-11-12", domain.StatusActive},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := activity(t, "ACT-1", "other", tc.start, tc.end, nil).Status(now); got != tc.want {
				t.Errorf("estado %s, quería %s", got, tc.want)
			}
		})
	}
}

func TestApplyAuditsOnlyWhatChangedAndCancels(t *testing.T) {
	a := activity(t, "ACT-1", "certification", "2026-11-16", "2026-11-20", chg1, ana)
	later := now.Add(time.Hour)
	end, ops, cancel := "2026-11-27", []domain.UserRef{luis, ana}, true
	changes, err := a.Apply(domain.ActivityEdit{End: &end, Operators: &ops, Cancelled: &cancel}, later)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][2]string{}
	for _, c := range changes {
		got[c.Field] = [2]string{c.Old, c.New}
	}
	want := map[string][2]string{
		"endDate":   {"2026-11-20", "2026-11-27"},
		"operators": {"u-ana", "u-ana,u-luis"},
		"cancelled": {"false", "true"},
	}
	if len(got) != len(want) || got["endDate"] != want["endDate"] || got["operators"] != want["operators"] || got["cancelled"] != want["cancelled"] {
		t.Errorf("cambios %v, quería %v", got, want)
	}
	if a.Status(now) != domain.StatusCancelled || !a.Snapshot().UpdatedAt.Equal(later) {
		t.Errorf("la actividad debe quedar cancelada y con la fecha de edición: %+v", a.Snapshot())
	}

	// Quitar el CHG de una certificación no se puede; reactivarla sí.
	reactivate := false
	if _, err := a.Apply(domain.ActivityEdit{Change: domain.ChangeEdit{Set: true}, Cancelled: &reactivate}, later); !slices.Equal(fields(err), []string{"change:required"}) {
		t.Errorf("sin CHG debía fallar: %v", err)
	}
	if changes, err := a.Apply(domain.ActivityEdit{Cancelled: &reactivate}, later); err != nil || len(changes) != 1 || a.Cancelled() {
		t.Errorf("reactivar: %v %v", changes, err)
	}

	same := "2026-11-27"
	if changes, _ := a.Apply(domain.ActivityEdit{End: &same}, later); len(changes) != 0 {
		t.Errorf("sin cambios reales no debe haber nada que auditar: %v", changes)
	}
}

func TestOverlaps(t *testing.T) {
	cert := activity(t, "ACT-1", "certification", "2026-11-16", "2026-11-27", chg1, ana, luis)
	regr := activity(t, "ACT-2", "regression", "2026-11-25", "2026-12-02", nil, ana)
	prod := activity(t, "ACT-3", "production", "2026-11-28", "2026-11-28", nil, luis) // empieza el día después: no se cruza
	edge := activity(t, "ACT-4", "other", "2026-11-27", "2026-11-27", chg2, luis)     // el último día: se cruza uno
	gone := activity(t, "ACT-5", "other", "2026-11-16", "2026-11-30", nil, ana, luis) // cancelada
	alone := activity(t, "ACT-6", "other", "2026-11-16", "2026-11-30", nil)           // sin operadores
	cancel := true
	if _, err := gone.Apply(domain.ActivityEdit{Cancelled: &cancel}, now); err != nil {
		t.Fatal(err)
	}

	got := domain.Overlaps([]domain.Activity{regr, prod, cert, edge, gone, alone})
	type row struct {
		op, first, second, from, to string
		days                        int
	}
	var rows []row
	for _, o := range got {
		rows = append(rows, row{o.Operator.Name, o.First.Code, o.Second.Code, o.From.Format(domain.DateLayout), o.To.Format(domain.DateLayout), o.Days()})
	}
	want := []row{
		{"Ana Rojas", "ACT-1", "ACT-2", "2026-11-25", "2026-11-27", 3},
		{"Luis Paz", "ACT-1", "ACT-4", "2026-11-27", "2026-11-27", 1},
	}
	if !slices.Equal(rows, want) {
		t.Errorf("solapes %v\nquería   %v", rows, want)
	}

	of := domain.OverlapsOf(regr, []domain.Activity{cert, regr, prod, gone})
	if len(of) != 1 || of[0].First.Code != "ACT-2" || of[0].Second.Code != "ACT-1" || of[0].Days() != 3 {
		t.Errorf("solapes de ACT-2: %+v", of)
	}
	if len(domain.OverlapsOf(gone, []domain.Activity{cert})) != 0 {
		t.Error("una actividad cancelada no se solapa")
	}
}

func TestCheckCertification(t *testing.T) {
	cancel := true
	cancelled := activity(t, "ACT-9", "certification", "2026-11-01", "2026-11-30", chg1, ana)
	if _, err := cancelled.Apply(domain.ActivityEdit{Cancelled: &cancel}, now); err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		list     []domain.Activity
		want     domain.CertificationState
		from, to string
	}{
		"nada":                {nil, domain.CertificationNone, "", ""},
		"solo otras clases":   {[]domain.Activity{activity(t, "ACT-1", "regression", "2026-11-01", "2026-11-30", chg1, ana)}, domain.CertificationNone, "", ""},
		"solo cancelada":      {[]domain.Activity{cancelled}, domain.CertificationNone, "", ""},
		"sin operadores":      {[]domain.Activity{activity(t, "ACT-1", "certification", "2026-11-01", "2026-11-30", chg1)}, domain.CertificationNoOperators, "", ""},
		"cubre hoy":           {[]domain.Activity{activity(t, "ACT-1", "certification", "2026-11-12", "2026-11-20", chg1, ana)}, domain.CertificationReady, "2026-11-12", "2026-11-20"},
		"todavía no empieza":  {[]domain.Activity{activity(t, "ACT-1", "certification", "2026-11-16", "2026-11-20", chg1, ana)}, domain.CertificationOutside, "2026-11-16", "2026-11-20"},
		"la más próxima gana": {[]domain.Activity{activity(t, "ACT-1", "certification", "2026-12-14", "2026-12-18", chg1, ana), activity(t, "ACT-2", "certification", "2026-11-02", "2026-11-10", chg1, luis)}, domain.CertificationOutside, "2026-11-02", "2026-11-10"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := domain.CheckCertification(tc.list, now)
			if got.State != tc.want {
				t.Fatalf("estado %s, quería %s", got.State, tc.want)
			}
			if tc.from != "" && (got.From.Format(domain.DateLayout) != tc.from || got.To.Format(domain.DateLayout) != tc.to) {
				t.Errorf("período %s a %s, quería %s a %s", got.From.Format(domain.DateLayout), got.To.Format(domain.DateLayout), tc.from, tc.to)
			}
		})
	}
}

func TestScheduleRange(t *testing.T) {
	from, to := day("2026-11-20"), day("2026-11-10")
	r, err := domain.ScheduleRange(nil, nil, now) // jueves 12: el lunes es el 9
	if err != nil || r.From.Format(domain.DateLayout) != "2026-11-09" || r.To.Format(domain.DateLayout) != "2026-12-20" {
		t.Errorf("rango por omisión %v a %v (%v)", r.From, r.To, err)
	}
	if r, _ := domain.ScheduleRange(&from, nil, now); r.To.Format(domain.DateLayout) != "2026-12-31" {
		t.Errorf("seis semanas desde el 20: %v", r.To)
	}
	if _, err := domain.ScheduleRange(&from, &to, now); !slices.Equal(fields(err), []string{"to:invalid"}) {
		t.Errorf("al revés: %v", err)
	}
	far := day("2027-03-01")
	if _, err := domain.ScheduleRange(&to, &far, now); !slices.Equal(fields(err), []string{"to:invalid"}) {
		t.Errorf("más de 92 días: %v", err)
	}
}

func TestBuildSchedule(t *testing.T) {
	r := domain.Range{From: day("2026-11-16"), To: day("2026-11-29")}
	cert1 := activity(t, "ACT-1", "certification", "2026-11-16", "2026-11-27", chg1, luis, ana)
	cert2 := activity(t, "ACT-2", "certification", "2026-11-23", "2026-12-04", chg2, ana)
	free := activity(t, "ACT-3", "other", "2026-11-20", "2026-11-20", nil)
	before := activity(t, "ACT-4", "regression", "2026-11-01", "2026-11-13", nil, luis)
	windows := []domain.Window{
		{ChangeCode: "CHG-2026-0001", Start: time.Date(2026, 11, 28, 22, 0, 0, 0, time.UTC), End: time.Date(2026, 11, 29, 2, 0, 0, 0, time.UTC)},
		{ChangeCode: "CHG-2026-0009", Start: time.Date(2026, 12, 10, 22, 0, 0, 0, time.UTC), End: time.Date(2026, 12, 11, 2, 0, 0, 0, time.UTC)},
	}
	inside, outside := milestone(t, "2026-11-25"), milestone(t, "2026-12-25")

	s := domain.BuildSchedule(r, []domain.Activity{cert2, before, free, cert1}, windows, []domain.Milestone{inside, outside}, "")
	if len(s.Operators) != 2 || s.Operators[0].Operator != ana || s.Operators[1].Operator != luis {
		t.Fatalf("filas %+v", s.Operators)
	}
	if codes := codesOf(s.Operators[0].Activities); !slices.Equal(codes, []string{"ACT-1", "ACT-2"}) {
		t.Errorf("Ana: %v", codes)
	}
	if len(s.Operators[0].Overlaps) != 1 || s.Operators[0].Overlaps[0].Days() != 5 || len(s.Operators[1].Overlaps) != 0 {
		t.Errorf("solapes: Ana %+v, Luis %+v", s.Operators[0].Overlaps, s.Operators[1].Overlaps)
	}
	if s.OverlapCount() != 1 || s.ActivityCount() != 3 {
		t.Errorf("resumen: %d solapes, %d actividades", s.OverlapCount(), s.ActivityCount())
	}
	if codes := codesOf(s.Unassigned); !slices.Equal(codes, []string{"ACT-3"}) {
		t.Errorf("sin operador: %v", codes)
	}
	if len(s.Projects) != 2 || s.Projects[0].Change.Code != "CHG-2026-0001" || s.Projects[1].Change.Code != "CHG-2026-0002" {
		t.Errorf("proyectos: %+v", s.Projects)
	}
	if len(s.Windows) != 1 || len(s.Milestones) != 1 {
		t.Errorf("ventanas %d, hitos %d", len(s.Windows), len(s.Milestones))
	}

	only := domain.BuildSchedule(r, []domain.Activity{cert1, cert2}, nil, nil, shared.UserId("u-luis"))
	if len(only.Operators) != 1 || only.Operators[0].Operator != luis {
		t.Errorf("filtrado por Luis: %+v", only.Operators)
	}
}

func codesOf(list []domain.Activity) []string {
	out := make([]string, len(list))
	for i, a := range list {
		out[i] = a.Snapshot().Code
	}
	return out
}
