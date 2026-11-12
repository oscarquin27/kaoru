package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/traceability/application/usecase"
	"kaoru/internal/feature/traceability/domain"
)

// world hace de los tres puertos. Solo lee: devuelve lo que tiene armado y anota qué le pidieron.
type world struct {
	systems         []domain.System
	header          domain.RequirementHeader
	states          []domain.StateRef
	transitions     []domain.Transition
	functionalCases []domain.CodeTitle
	testCases       []domain.TestCaseSummary
	defects         []domain.DefectSummary
	missing         []domain.Readiness
	coverage        domain.Coverage
	hits            []domain.SearchHit
	trace           domain.PackageTrace

	indicators domain.Indicators
	totalRows  int // lo que dice MatrixIndicators
	storedRows int // las filas que de verdad devuelve Matrix

	fail  map[string]error // por nombre de método
	calls []string
}

var (
	notFound = axis.New(axis.NotFoundResource, "No existe.")
	down     = axis.New(axis.ServiceUnavailable, "Sin servicio.")
)

// call anota la llamada y devuelve el error armado para ese método, si hay.
func (w *world) call(method string, args ...any) error {
	parts := []string{method}
	for _, arg := range args {
		parts = append(parts, fmt.Sprint(arg))
	}
	w.calls = append(w.calls, strings.Join(parts, " "))
	return w.fail[method]
}

func (w *world) Systems(context.Context) ([]domain.System, error) {
	return w.systems, w.call("Systems")
}

func (w *world) RequirementHeader(_ context.Context, code string) (domain.RequirementHeader, error) {
	if err := w.call("RequirementHeader", code); err != nil {
		return domain.RequirementHeader{}, err
	}
	if code != w.header.Code {
		return domain.RequirementHeader{}, notFound
	}
	return w.header, nil
}

func (w *world) LifecycleStates(context.Context) ([]domain.StateRef, error) {
	return w.states, w.call("LifecycleStates")
}

func (w *world) Transitions(_ context.Context, requirementId string) ([]domain.Transition, error) {
	return w.transitions, w.call("Transitions", requirementId)
}

func (w *world) FunctionalCases(_ context.Context, requirementId string) ([]domain.CodeTitle, error) {
	return w.functionalCases, w.call("FunctionalCases", requirementId)
}

func (w *world) TestCases(_ context.Context, requirementId string) ([]domain.TestCaseSummary, error) {
	return w.testCases, w.call("TestCases", requirementId)
}

func (w *world) Defects(_ context.Context, requirementId string) ([]domain.DefectSummary, error) {
	return w.defects, w.call("Defects", requirementId)
}

// Matrix devuelve la página pedida de storedRows filas numeradas.
func (w *world) Matrix(_ context.Context, filter domain.MatrixFilter, page, size int) ([]domain.MatrixRow, error) {
	if err := w.call("Matrix", filter.SystemKey, page, size); err != nil {
		return nil, err
	}
	var rows []domain.MatrixRow
	for i := (page - 1) * size; i < min(page*size, w.storedRows); i++ {
		rows = append(rows, domain.MatrixRow{RequirementCode: fmt.Sprintf("REQ-2026-%04d", i+1), SystemKey: filter.SystemKey})
	}
	return rows, nil
}

func (w *world) MatrixIndicators(_ context.Context, filter domain.MatrixFilter) (domain.Indicators, int, error) {
	return w.indicators, w.totalRows, w.call("MatrixIndicators", filter.SystemKey)
}

func (w *world) Search(_ context.Context, query string, limit int) ([]domain.SearchHit, error) {
	return w.hits, w.call("Search", query, limit)
}

func (w *world) PackageTrace(_ context.Context, packageCode string) (domain.PackageTrace, error) {
	return w.trace, w.call("PackageTrace", packageCode)
}

// readiness y coverage separan los dos puertos que comparten el nombre de método Of.
type readiness struct{ w *world }

func (r readiness) Of(_ context.Context, requirementCode string) ([]domain.Readiness, error) {
	return r.w.missing, r.w.call("Readiness", requirementCode)
}

type coverage struct{ w *world }

func (c coverage) Of(_ context.Context, requirementCode string) (domain.Coverage, error) {
	return c.w.coverage, c.w.call("Coverage", requirementCode)
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return t0 }

var (
	t0  = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	ctx = context.Background()
)

func days(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }

// setup arma un REQ que nació hace diez días en borrador y hoy está en pruebas.
func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{
		systems: []domain.System{{Key: "core", Name: "Core bancario", DefaultRoute: "full"}, {Key: "web", Name: "Banca web"}},
		header: domain.RequirementHeader{
			Id: "r1", Code: "REQ-2026-0042", Title: "Consultar saldo", State: domain.StateRef{Key: "testing", Name: "En pruebas"},
			CreatedAt: t0.Add(-days(10)),
		},
		states: []domain.StateRef{
			{Key: "draft", Name: "Borrador"}, {Key: "analysis", Name: "En análisis"},
			{Key: "testing", Name: "En pruebas"}, {Key: "released", Name: "Liberado"},
		},
		transitions: []domain.Transition{
			{At: t0.Add(-days(8)), From: "draft", To: "analysis"},
			{At: t0.Add(-days(2)), From: "analysis", To: "testing"},
		},
		functionalCases: []domain.CodeTitle{{Code: "UC-2026-0001", Title: "Consultar saldo"}},
		testCases: []domain.TestCaseSummary{
			{Code: "TC-2026-0001", Title: "Saldo", IsCritical: true, LastRun: domain.RunSummary{Code: "RUN-2026-0001", Result: "failed"}},
			{Code: "TC-2026-0002", Title: "Saldo viejo", Retired: true},
		},
		defects: []domain.DefectSummary{{Code: "DEF-2026-0001", Title: "Saldo negativo", Severity: "high"}},
		missing: []domain.Readiness{{
			To:         domain.StateRef{Key: "released", Name: "Liberado"},
			Conditions: []domain.Condition{{RuleKey: "no_blocking_defects", Message: "Hay un DEF alto abierto.", Blocks: true, Related: []string{"DEF-2026-0001"}}},
		}},
		coverage:   domain.Coverage{Required: 1, Passed: 0},
		hits:       []domain.SearchHit{{Kind: "requirement", Code: "REQ-2026-0042", Title: "Consultar saldo"}},
		indicators: domain.NewIndicators(4, 3, 1, map[string]int{"high": 1}),
		fail:       map[string]error{},
	}
	uc, err := usecase.New(usecase.Deps{Repository: w, Readiness: readiness{w}, Coverage: coverage{w}, Clock: fixedClock{}})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

// lifecycleOf resume la línea de vida como «etapa:estado:días».
func lifecycleOf(steps []domain.LifecycleStep) []string {
	var out []string
	for _, s := range steps {
		out = append(out, fmt.Sprintf("%s:%s:%g", s.State.Key, s.Status, s.Days))
	}
	return out
}

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestListSystems(t *testing.T) {
	uc, w := setup(t)
	out, err := uc.ListSystems.Execute(ctx)
	if err != nil || !slices.Equal(out, w.systems) {
		t.Errorf("ListSystems = %+v, %v", out, err)
	}
	w.fail["Systems"] = down
	if _, err := uc.ListSystems.Execute(ctx); !errors.Is(err, down) {
		t.Errorf("si el repositorio falla: %v", err)
	}
}

func TestGetRequirementOverview(t *testing.T) {
	uc, w := setup(t)
	// Leer no exige permiso: ni siquiera recibe un actor.
	out, err := uc.GetRequirementOverview.Execute(ctx, usecase.GetRequirementOverviewInput{Code: "REQ-2026-0042"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Requirement != w.header || !reflect.DeepEqual(out.Missing, w.missing) || out.Coverage != w.coverage ||
		!slices.Equal(out.FunctionalCases, w.functionalCases) || !slices.Equal(out.TestCases, w.testCases) || !slices.Equal(out.Defects, w.defects) {
		t.Errorf("la ficha no reúne lo que devolvió cada puerto: %+v", out)
	}
	// El tiempo en cada etapa sale del reloj; la siguiente está bloqueada por una regla.
	want := []string{"draft:done:2", "analysis:done:6", "testing:current:2", "released:blocked:0"}
	if !slices.Equal(lifecycleOf(out.Lifecycle), want) {
		t.Errorf("línea de vida = %v; quiero %v", lifecycleOf(out.Lifecycle), want)
	}
	if at := out.Lifecycle[1].EnteredAt; at == nil || !at.Equal(t0.Add(-days(8))) || out.Lifecycle[3].EnteredAt != nil {
		t.Errorf("cuándo entró a cada etapa: %+v", out.Lifecycle)
	}
	// A requirement y execution se les pregunta por código; al repositorio, por id.
	wantCalls := []string{
		"RequirementHeader REQ-2026-0042", "Readiness REQ-2026-0042", "Coverage REQ-2026-0042",
		"FunctionalCases r1", "TestCases r1", "Defects r1", "LifecycleStates", "Transitions r1",
	}
	if !slices.Equal(w.calls, wantCalls) {
		t.Errorf("llamadas = %v", w.calls)
	}
}

func TestGetRequirementOverviewLifecycle(t *testing.T) {
	blocking := func(to string, blocks bool) []domain.Readiness {
		return []domain.Readiness{{To: domain.StateRef{Key: to}, Conditions: []domain.Condition{{RuleKey: "rule", Blocks: false}, {RuleKey: "rule", Blocks: blocks}}}}
	}
	cases := []struct {
		name   string
		mutate func(w *world)
		want   []string
	}{
		{"sin transiciones nació en el estado actual", func(w *world) {
			w.transitions, w.missing, w.header.State.Key = nil, nil, "draft"
		}, []string{"draft:current:10", "analysis:pending:0", "testing:pending:0", "released:pending:0"}},
		{"nada falta para avanzar", func(w *world) { w.missing = nil },
			[]string{"draft:done:2", "analysis:done:6", "testing:current:2", "released:pending:0"}},
		{"una condición que solo advierte no bloquea", func(w *world) { w.missing = blocking("released", false) },
			[]string{"draft:done:2", "analysis:done:6", "testing:current:2", "released:pending:0"}},
		{"una condición que bloquea otra transición no bloquea la siguiente etapa", func(w *world) { w.missing = blocking("rejected", true) },
			[]string{"draft:done:2", "analysis:done:6", "testing:current:2", "released:pending:0"}},
		{"en la última etapa no hay siguiente que bloquear", func(w *world) {
			w.transitions = append(w.transitions, domain.Transition{At: t0.Add(-days(1)), From: "testing", To: "released"})
			w.header.State.Key = "released"
		}, []string{"draft:done:2", "analysis:done:6", "testing:done:1", "released:current:1"}},
		{"en un estado lateral ninguna etapa es la actual", func(w *world) {
			w.transitions = append(w.transitions, domain.Transition{At: t0.Add(-days(1)), From: "testing", To: "with-defects"})
			w.header.State.Key = "with-defects"
		}, []string{"draft:done:2", "analysis:done:6", "testing:done:1", "released:pending:0"}},
		{"una etapa a la que volvió acumula el tiempo", func(w *world) {
			w.missing = nil
			w.transitions = append(w.transitions,
				domain.Transition{At: t0.Add(-days(1)), From: "testing", To: "analysis"})
			w.header.State.Key = "analysis"
		}, []string{"draft:done:2", "analysis:current:7", "testing:done:1", "released:pending:0"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			tc.mutate(w)
			out, err := uc.GetRequirementOverview.Execute(ctx, usecase.GetRequirementOverviewInput{Code: "REQ-2026-0042"})
			if err != nil || !slices.Equal(lifecycleOf(out.Lifecycle), tc.want) {
				t.Errorf("línea de vida = %v, %v; quiero %v", lifecycleOf(out.Lifecycle), err, tc.want)
			}
		})
	}
}

func TestGetRequirementOverviewErrors(t *testing.T) {
	uc, w := setup(t)
	if _, err := uc.GetRequirementOverview.Execute(ctx, usecase.GetRequirementOverviewInput{Code: "REQ-2026-0099"}); !errors.Is(err, usecase.ErrRequirementNotFound) {
		t.Errorf("REQ inexistente: %v", err)
	}
	if len(w.calls) != 1 {
		t.Errorf("sin REQ no se pregunta nada más: %v", w.calls)
	}

	// Si cualquier puerto falla, no hay ficha a medias.
	for _, method := range []string{
		"RequirementHeader", "Readiness", "Coverage", "FunctionalCases", "TestCases", "Defects", "LifecycleStates", "Transitions",
	} {
		t.Run("falla "+method, func(t *testing.T) {
			uc, w := setup(t)
			w.fail[method] = down
			out, err := uc.GetRequirementOverview.Execute(ctx, usecase.GetRequirementOverviewInput{Code: "REQ-2026-0042"})
			if !errors.Is(err, down) || !reflect.DeepEqual(out, domain.Overview{}) {
				t.Errorf("esperaba el error del puerto y una ficha vacía: %+v, %v", out, err)
			}
		})
	}
}

func TestGetMatrix(t *testing.T) {
	uc, w := setup(t)
	w.totalRows, w.storedRows = 7, 7
	out, err := uc.GetMatrix.Execute(ctx, usecase.GetMatrixInput{Filter: domain.MatrixFilter{SystemKey: "core"}, Page: 2, Size: 3})
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, row := range out.Rows {
		codes = append(codes, row.RequirementCode+"/"+row.SystemKey)
	}
	// La página trae sus filas; los indicadores y el total son de todo lo filtrado.
	if !slices.Equal(codes, []string{"REQ-2026-0004/core", "REQ-2026-0005/core", "REQ-2026-0006/core"}) || out.Total != 7 ||
		!reflect.DeepEqual(out.Indicators, w.indicators) || out.Indicators.CoveragePercent != 75 {
		t.Errorf("GetMatrix = %v, total %d, %+v", codes, out.Total, out.Indicators)
	}
	if !slices.Equal(w.calls, []string{"MatrixIndicators core", "Matrix core 2 3"}) {
		t.Errorf("el filtro y la página deben llegar al repositorio: %v", w.calls)
	}

	for _, method := range []string{"MatrixIndicators", "Matrix"} {
		uc, w := setup(t)
		w.totalRows, w.storedRows = 7, 7
		w.fail[method] = down
		out, err := uc.GetMatrix.Execute(ctx, usecase.GetMatrixInput{Page: 1, Size: 3})
		if !errors.Is(err, down) || !reflect.DeepEqual(out, usecase.GetMatrixOutput{}) {
			t.Errorf("falla %s: %+v, %v", method, out, err)
		}
	}
}

func TestExportMatrix(t *testing.T) {
	cases := []struct {
		name      string
		total     int
		stored    int
		wantRows  int
		wantCalls []string
	}{
		{"sin filas no lee ninguna página", 0, 0, 0, []string{"MatrixIndicators web"}},
		{"menos de un lote", 3, 3, 3, []string{"MatrixIndicators web", "Matrix web 1 1000"}},
		{"varios lotes", 2500, 2500, 2500, []string{"MatrixIndicators web", "Matrix web 1 1000", "Matrix web 2 1000", "Matrix web 3 1000"}},
		{"justo en el tope", usecase.MaxExportRows, 1, 1, []string{"MatrixIndicators web", "Matrix web 1 1000", "Matrix web 2 1000"}},
		{"si las filas se acaban antes del total, termina", 1500, 1000, 1000, []string{"MatrixIndicators web", "Matrix web 1 1000", "Matrix web 2 1000"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			w.totalRows, w.storedRows = tc.total, tc.stored
			rows, err := uc.ExportMatrix.Execute(ctx, usecase.ExportMatrixInput{Filter: domain.MatrixFilter{SystemKey: "web"}})
			if err != nil || len(rows) != tc.wantRows || !slices.Equal(w.calls, tc.wantCalls) {
				t.Fatalf("%d filas, %v, llamadas %v", len(rows), err, w.calls)
			}
			// Los lotes van en orden y sin repetir filas.
			if tc.wantRows > 0 && (rows[0].RequirementCode != "REQ-2026-0001" ||
				rows[len(rows)-1].RequirementCode != fmt.Sprintf("REQ-2026-%04d", tc.wantRows)) {
				t.Errorf("primera y última fila: %s, %s", rows[0].RequirementCode, rows[len(rows)-1].RequirementCode)
			}
		})
	}
}

func TestExportMatrixErrors(t *testing.T) {
	uc, w := setup(t)
	w.totalRows, w.storedRows = usecase.MaxExportRows+1, usecase.MaxExportRows+1
	rows, err := uc.ExportMatrix.Execute(ctx, usecase.ExportMatrixInput{Filter: domain.MatrixFilter{SystemKey: "web"}})
	if !errors.Is(err, usecase.ErrExportTooLarge) || !axis.IsCode(err, axis.InvalidAction) || rows != nil {
		t.Errorf("exportación demasiado grande: %d filas, %v", len(rows), err)
	}
	if !slices.Equal(w.calls, []string{"MatrixIndicators web"}) {
		t.Errorf("pasado el tope no se lee ninguna fila: %v", w.calls)
	}

	for _, method := range []string{"MatrixIndicators", "Matrix"} {
		uc, w := setup(t)
		w.totalRows, w.storedRows = 1500, 1500
		w.fail[method] = down
		if rows, err := uc.ExportMatrix.Execute(ctx, usecase.ExportMatrixInput{}); !errors.Is(err, down) || rows != nil {
			t.Errorf("falla %s: %d filas, %v", method, len(rows), err)
		}
	}
}

func TestSearch(t *testing.T) {
	uc, w := setup(t)
	out, err := uc.Search.Execute(ctx, usecase.SearchInput{Query: "  REQ-2026-0042 "})
	if err != nil || !slices.Equal(out, w.hits) {
		t.Errorf("Search = %+v, %v", out, err)
	}
	// La consulta llega sin espacios y con el tope de 20 resultados.
	if !slices.Equal(w.calls, []string{"Search REQ-2026-0042 20"}) {
		t.Errorf("llamadas = %v", w.calls)
	}

	// El mínimo se cuenta en caracteres, no en bytes.
	if _, err := uc.Search.Execute(ctx, usecase.SearchInput{Query: "ñu"}); err != nil {
		t.Errorf("dos caracteres bastan: %v", err)
	}
	w.calls = nil
	for _, query := range []string{"", "   ", "a", " a ", "ñ"} {
		_, err := uc.Search.Execute(ctx, usecase.SearchInput{Query: query})
		var ge *axis.GeneralError
		if !axis.IsCode(err, axis.InvalidRequest) || !errors.As(err, &ge) || len(ge.Fields()) != 1 || ge.Fields()[0].Field != "q" {
			t.Errorf("consulta %q: %v", query, err)
		}
	}
	if len(w.calls) != 0 {
		t.Errorf("una consulta demasiado corta no llega al repositorio: %v", w.calls)
	}

	w.fail["Search"] = down
	if _, err := uc.Search.Execute(ctx, usecase.SearchInput{Query: "saldo"}); !errors.Is(err, down) {
		t.Errorf("si el repositorio falla: %v", err)
	}
}

func TestGetPackageTrace(t *testing.T) {
	uc, w := setup(t)
	w.trace = domain.PackageTrace{
		Rows: []domain.PackageTraceRow{{RequirementCode: "REQ-2026-0001", TestCaseCode: "TC-2026-0001", RunCode: "RUN-2026-0001"}},
		Runs: []domain.PackageRun{{Code: "RUN-2026-0001"}},
	}
	got, err := uc.GetPackageTrace.Execute(context.Background(), usecase.GetPackageTraceInput{PackageCode: "PKG-2026-0001"})
	if err != nil || !reflect.DeepEqual(got, w.trace) {
		t.Errorf("trazabilidad del PKG: %v %+v", err, got)
	}
	if !reflect.DeepEqual(w.calls, []string{"PackageTrace PKG-2026-0001"}) {
		t.Errorf("llamadas: %v", w.calls)
	}
}
