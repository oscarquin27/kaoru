package domain_test

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now     = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	sypago  = domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"}
	initial = domain.State{Id: "s1", Key: "registered", Name: "Registrado"}
	ana     = domain.UserRef{Id: "u1", Name: "Ana"}
	chgSy   = &domain.ChangeRef{Id: "c1", Code: "CHG-2026-0007", Title: "Pagos", Route: "minor_fix", SystemKey: "SYPAGO"}
	chgSimf = &domain.ChangeRef{Id: "c2", Code: "CHG-2026-0008", Title: "SIMF", Route: "high", SystemKey: "SIMF"}
)

func params() domain.NewParams {
	return domain.NewParams{
		Id: "r1", Code: shared.NewCode(shared.PrefixRequirement, 2026, 42), Title: " Límite diario ",
		AcceptanceCriteria: " Rechaza sobre el límite ", System: sypago, State: initial, CreatedBy: ana, Now: now,
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

func TestNewRequirement(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewParams)
		wantFields []string
		wantRoute  string
	}{
		{"sin CHG toma la ruta del sistema", func(*domain.NewParams) {}, nil, "light"},
		{"con CHG toma la ruta del CHG", func(p *domain.NewParams) { p.Change = chgSy }, nil, "minor_fix"},
		{"sin título", func(p *domain.NewParams) { p.Title = " " }, []string{"title:required"}, ""},
		{"título largo", func(p *domain.NewParams) { p.Title = strings.Repeat("ñ", 201) }, []string{"title:invalid"}, ""},
		{"CHG de otro sistema", func(p *domain.NewParams) { p.Change = chgSimf }, []string{"change:invalid"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			requirement, err := domain.New(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := requirement.Snapshot()
			if requirement.Route() != tc.wantRoute || s.Title != "Límite diario" ||
				s.AcceptanceCriteria != "Rechaza sobre el límite" || s.State != initial {
				t.Errorf("REQ inesperado: %+v, ruta %s", s, requirement.Route())
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

func TestApplyRequirement(t *testing.T) {
	later := now.Add(time.Hour)
	cases := []struct {
		name        string
		edit        domain.Edit
		wantChanged []string
		wantFields  []string
	}{
		{"sin cambios", domain.Edit{Title: ptr("Límite diario")}, nil, nil},
		{"título y criterios", domain.Edit{Title: ptr("Otro"), AcceptanceCriteria: ptr("Nuevos")}, []string{"title", "acceptanceCriteria"}, nil},
		{"asigna CHG y responsable", domain.Edit{Change: domain.ChangeEdit{Set: true, Value: chgSy}, Owner: domain.OwnerEdit{Set: true, Value: &ana}}, []string{"change", "owner"}, nil},
		{"CHG de otro sistema", domain.Edit{Change: domain.ChangeEdit{Set: true, Value: chgSimf}}, nil, []string{"change:invalid"}},
		{"título vacío", domain.Edit{Title: ptr("")}, nil, []string{"title:required"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requirement, err := domain.New(params())
			if err != nil {
				t.Fatal(err)
			}
			changes, err := requirement.Apply(tc.edit, later)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				if requirement.Snapshot().UpdatedAt != now {
					t.Error("una edición rechazada no debe cambiar el REQ")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var changed []string
			for _, c := range changes {
				changed = append(changed, c.Field)
			}
			if !slices.Equal(changed, tc.wantChanged) {
				t.Errorf("cambios = %v, quiero %v", changed, tc.wantChanged)
			}
		})
	}

	// Quitar el CHG y el responsable también queda como cambio.
	p := params()
	p.Change, p.Owner = chgSy, &ana
	requirement, _ := domain.New(p)
	changes, err := requirement.Apply(domain.Edit{Change: domain.ChangeEdit{Set: true}, Owner: domain.OwnerEdit{Set: true}}, later)
	if err != nil || len(changes) != 2 || changes[0].Old != "CHG-2026-0007" || changes[0].New != "" || requirement.Route() != "light" {
		t.Errorf("quitar CHG y responsable: %+v, %v", changes, err)
	}
}

// La tabla de casos compartida con el front (reglas §A.1.2).
func TestParseSheetLinkSharedCases(t *testing.T) {
	raw, err := os.ReadFile("../../../../../api/testdata/validation-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		SheetLink []struct {
			Name       string `json:"name"`
			Input      string `json:"input"`
			Valid      bool   `json:"valid"`
			Normalized struct {
				ID    string `json:"id"`
				Gid   string `json:"gid"`
				Range string `json:"range"`
			} `json:"normalized"`
		} `json:"sheetLink"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.SheetLink) < 10 {
		t.Fatalf("la tabla de casos trae %d casos de sheetLink", len(file.SheetLink))
	}
	for _, tc := range file.SheetLink {
		t.Run(tc.Name, func(t *testing.T) {
			ref, ok := domain.ParseSheetLink(tc.Input)
			if ok != tc.Valid {
				t.Fatalf("válido = %v, quiero %v", ok, tc.Valid)
			}
			want := domain.SheetRef{Id: tc.Normalized.ID, Gid: tc.Normalized.Gid, Range: tc.Normalized.Range}
			if ref != want {
				t.Errorf("normalizado = %+v, quiero %+v", ref, want)
			}
		})
	}
}

func TestNewLink(t *testing.T) {
	const sheet = "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789/edit#gid=1842&range=A1:F40"
	cases := []struct {
		name       string
		url        string
		kind       string
		label      string
		wantFields []string
		wantKind   domain.LinkKind
		wantLabel  string
		wantSheet  bool
	}{
		{"hoja de Google con etiqueta legible", sheet, "", "", nil, domain.LinkGoogleSheets, "Google Sheets · pestaña 1842 · A1:F40", true},
		{"Google Docs es un enlace más", "https://docs.google.com/document/d/1AbCdEfGhIjKlMnOpQrStUv/edit", "", "", nil, domain.LinkOther, "Google Docs", false},
		{"Google Slides", "https://docs.google.com/presentation/d/1AbCdEfGhIjKlMnOpQrStUv/edit", "", "", nil, domain.LinkOther, "Google Slides", false},
		{"Google Drive", "https://drive.google.com/drive/folders/1AbCdEf", "", "", nil, domain.LinkOther, "Google Drive", false},
		{"Google Docs como carpeta compartida", "https://docs.google.com/document/d/1AbC/edit", "shared_folder", "Minuta", nil, domain.LinkSharedFolder, "Minuta", false},
		{"hoja con etiqueta propia", sheet, "", " Especificación SyPago ", nil, domain.LinkGoogleSheets, "Especificación SyPago", true},
		{"una hoja siempre es google_sheets", sheet, "wiki", "", nil, domain.LinkGoogleSheets, "Google Sheets · pestaña 1842 · A1:F40", true},
		{"YouTrack se deduce del host", "https://youtrack.lab.invalid/issue/SY-12", "", "", nil, domain.LinkYouTrack, "youtrack.lab.invalid", false},
		{"otro enlace", "http://wiki.lab.invalid/pagos", "wiki", "", nil, domain.LinkWiki, "wiki.lab.invalid", false},
		{"sin tipo es otro", "https://intranet.lab.invalid/x", "", "", nil, domain.LinkOther, "intranet.lab.invalid", false},
		{"sin URL", " ", "", "", []string{"url:required"}, "", "", false},
		{"sin esquema", "wiki.lab.invalid/pagos", "", "", []string{"url:invalid"}, "", "", false},
		{"esquema peligroso", "javascript:alert(1)", "", "", []string{"url:invalid"}, "", "", false},
		{"archivo local", "file:///etc/passwd", "", "", []string{"url:invalid"}, "", "", false},
		{"dice ser hoja y no lo es", "https://intranet.lab.invalid/x", "google_sheets", "", []string{"url:invalid"}, "", "", false},
		{"tipo desconocido", "https://intranet.lab.invalid/x", "correo", "", []string{"kind:invalid"}, "", "", false},
		{"URL larguísima", "https://x.invalid/" + strings.Repeat("a", 2000), "", "", []string{"url:invalid"}, "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			link, err := domain.NewLink(domain.NewLinkParams{
				Id: "l1", RequirementId: "r1", URL: tc.url, Kind: tc.kind, Label: tc.label, AddedBy: ana, Now: now,
			})
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := link.Snapshot()
			if s.Kind != tc.wantKind || s.Label != tc.wantLabel || (s.Sheet != nil) != tc.wantSheet {
				t.Errorf("enlace inesperado: %+v", s)
			}
			if link.NeedsSnapshot() != tc.wantSheet {
				t.Errorf("solo una hoja de Google sin instantánea la necesita")
			}
		})
	}
}

func TestLinkApply(t *testing.T) {
	link, err := domain.NewLink(domain.NewLinkParams{
		Id: "l1", RequirementId: "r1", AddedBy: ana, Now: now,
		URL: "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789/edit",
	})
	if err != nil {
		t.Fatal(err)
	}
	changes, err := link.Apply(domain.LinkEdit{
		IsOfficial: ptr(true), Snapshot: domain.SnapshotEdit{Set: true, DocumentId: "d1", Code: "DOC-2026-0031"},
	}, now.Add(time.Hour))
	if err != nil || len(changes) != 2 || link.NeedsSnapshot() || !link.Snapshot().IsOfficial {
		t.Errorf("adjuntar instantánea: %+v, %v", changes, err)
	}
	if _, err := link.Apply(domain.LinkEdit{Label: ptr(" ")}, now); !slices.Equal(fieldsOf(t, err), []string{"label:required"}) {
		t.Errorf("etiqueta vacía: %v", err)
	}
	changes, _ = link.Apply(domain.LinkEdit{Snapshot: domain.SnapshotEdit{Set: true}}, now)
	if len(changes) != 1 || !link.NeedsSnapshot() {
		t.Errorf("quitar la instantánea: %+v", changes)
	}
}

func TestUnmet(t *testing.T) {
	entry := func(done ...bool) []domain.EntryItem {
		items := []domain.EntryItem{{Key: "ids", Label: "IDs de REQ y CHG"}, {Key: "rollback", Label: "Notas de rollback"}}
		for i, d := range done {
			items[i].Done = d
		}
		return items
	}
	coverage := map[string]any{"high": 100.0, "medium": 80.0, "low": 50.0}
	rule := func(key string, params map[string]any) []domain.Rule {
		return []domain.Rule{{Key: key, Params: params, Blocks: true}}
	}
	cases := []struct {
		name        string
		rules       []domain.Rule
		facts       domain.Facts
		wantMessage string // vacío: la regla se cumple
		wantRelated []string
	}{
		{"riesgo sin evaluar", rule(domain.RuleRiskAssessed, nil), domain.Facts{}, "Falta evaluar el riesgo.", nil},
		{"riesgo evaluado", rule(domain.RuleRiskAssessed, nil), domain.Facts{Risk: &domain.RiskFact{Level: "low"}}, "", nil},
		{"RISK sin características", rule(domain.RuleRiskHasCharacteristics, nil), domain.Facts{Risk: &domain.RiskFact{Level: "high"}},
			"El RISK no indica qué características de calidad (ISO 25010) están en juego.", nil},
		{"RISK con características", rule(domain.RuleRiskHasCharacteristics, nil), domain.Facts{Risk: &domain.RiskFact{Level: "high", Characteristics: 2}}, "", nil},
		{"contrato incompleto", rule(domain.RuleEntryContractComplete, nil), domain.Facts{EntryItems: entry(true, false)},
			"Falta 1 ítem del contrato de entrada: Notas de rollback.", nil},
		{"contrato sin nada", rule(domain.RuleEntryContractComplete, nil), domain.Facts{EntryItems: entry()},
			"Faltan 2 ítems del contrato de entrada: IDs de REQ y CHG; Notas de rollback.", nil},
		{"contrato completo", rule(domain.RuleEntryContractComplete, nil), domain.Facts{EntryItems: entry(true, true)}, "", nil},
		{"cobertura sin riesgo", rule(domain.RuleMinCoverage, coverage), domain.Facts{Coverage: domain.CoverageFact{Required: 2}},
			"Falta evaluar el riesgo: de él depende la cobertura mínima.", nil},
		{"cobertura sin TC", rule(domain.RuleMinCoverage, coverage), domain.Facts{Risk: &domain.RiskFact{Level: "low"}},
			"El REQ no tiene TC definidos.", nil},
		{"cobertura insuficiente", rule(domain.RuleMinCoverage, coverage),
			domain.Facts{Risk: &domain.RiskFact{Level: "medium"}, Coverage: domain.CoverageFact{Required: 5, Pending: []string{"TC-2026-0112", "TC-2026-0113"}}},
			"Faltan 2 TC sin RUN aprobado: TC-2026-0112, TC-2026-0113. Riesgo medio exige 80 % de cobertura; hay 3 de 5.",
			[]string{"TC-2026-0112", "TC-2026-0113"}},
		{"cobertura justa", rule(domain.RuleMinCoverage, coverage),
			domain.Facts{Risk: &domain.RiskFact{Level: "medium"}, Coverage: domain.CoverageFact{Required: 5, Pending: []string{"TC-2026-0112"}}}, "", nil},
		{"riesgo alto exige todo", rule(domain.RuleMinCoverage, coverage),
			domain.Facts{Risk: &domain.RiskFact{Level: "high"}, Coverage: domain.CoverageFact{Required: 5, Pending: []string{"TC-2026-0112"}}},
			"Falta 1 TC sin RUN aprobado: TC-2026-0112. Riesgo alto exige 100 % de cobertura; hay 4 de 5.", []string{"TC-2026-0112"}},
		{"sin porcentaje configurado exige todo", rule(domain.RuleMinCoverage, map[string]any{"low": "mucho"}),
			domain.Facts{Risk: &domain.RiskFact{Level: "low"}, Coverage: domain.CoverageFact{Required: 2, Pending: []string{"TC-2026-0001"}}},
			"Falta 1 TC sin RUN aprobado: TC-2026-0001. Riesgo bajo exige 100 % de cobertura; hay 1 de 2.", []string{"TC-2026-0001"}},
		{"defecto alto abierto", rule(domain.RuleNoOpenDefect, map[string]any{"severity": "high"}),
			domain.Facts{OpenDefects: []domain.DefectFact{{Code: "DEF-2026-0091", Severity: "high"}, {Code: "DEF-2026-0092", Severity: "low"}}},
			"Hay 1 DEF abierto de severidad alta: DEF-2026-0091.", []string{"DEF-2026-0091"}},
		{"solo defectos bajos", rule(domain.RuleNoOpenDefect, map[string]any{"severity": "high"}),
			domain.Facts{OpenDefects: []domain.DefectFact{{Code: "DEF-2026-0092", Severity: "low"}}}, "", nil},
		{"enlaces sin instantánea", rule(domain.RuleLinksHaveSnapshot, nil), domain.Facts{LinksWithoutSnapshot: 2},
			"2 enlaces de Google Sheets no tienen instantánea. Para el informe de conformidad cuenta la instantánea, no el enlace.", nil},
		{"REQ sin CHG cumple la certificación", rule(domain.RuleCertificationPlanned, nil), domain.Facts{}, "", nil},
		{"certificación lista", rule(domain.RuleCertificationPlanned, nil),
			domain.Facts{Certification: &domain.CertificationFact{ChangeCode: "CHG-2026-0001", State: domain.CertificationReady}}, "", nil},
		{"certificación sin reservar", rule(domain.RuleCertificationPlanned, nil),
			domain.Facts{Certification: &domain.CertificationFact{ChangeCode: "CHG-2026-0001", State: domain.CertificationNone}},
			"CHG-2026-0001 no tiene reservado el período de certificación. Planifícalo en Planificación.", []string{"CHG-2026-0001"}},
		{"certificación sin operadores", rule(domain.RuleCertificationPlanned, nil),
			domain.Facts{Certification: &domain.CertificationFact{ChangeCode: "CHG-2026-0001", State: domain.CertificationNoOperators}},
			"La certificación de CHG-2026-0001 no tiene operadores asignados.", []string{"CHG-2026-0001"}},
		{"certificación en otras fechas", rule(domain.RuleCertificationPlanned, nil),
			domain.Facts{Certification: &domain.CertificationFact{ChangeCode: "CHG-2026-0001", State: domain.CertificationOutside, From: "2026-11-16", To: "2026-11-27"}},
			"Hoy está fuera del período de certificación de CHG-2026-0001 (del 2026-11-16 al 2026-11-27).", []string{"CHG-2026-0001"}},
		{"regla desconocida bloquea", rule("inventada", nil), domain.Facts{}, "La regla «inventada» no se puede evaluar. Avisa a QA.", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conditions := domain.Unmet(tc.rules, tc.facts)
			if tc.wantMessage == "" {
				if len(conditions) != 0 {
					t.Fatalf("la regla debía cumplirse: %+v", conditions)
				}
				return
			}
			if len(conditions) != 1 || conditions[0].Message != tc.wantMessage || !slices.Equal(conditions[0].Related, tc.wantRelated) ||
				conditions[0].RuleKey != tc.rules[0].Key || !conditions[0].Blocks {
				t.Errorf("condición inesperada:\n%+v\nquiero: %s %v", conditions, tc.wantMessage, tc.wantRelated)
			}
		})
	}
}

func TestUnmetRespectsRouteAndWarnings(t *testing.T) {
	rules := []domain.Rule{
		{Key: domain.RuleRiskHasCharacteristics, Route: "high", Blocks: true},
		{Key: domain.RuleLinksHaveSnapshot, Blocks: false},
	}
	facts := domain.Facts{Route: "light", Risk: &domain.RiskFact{Level: "low"}, LinksWithoutSnapshot: 1}

	conditions := domain.Unmet(rules, facts)
	if len(conditions) != 1 || conditions[0].RuleKey != domain.RuleLinksHaveSnapshot || conditions[0].Blocks {
		t.Fatalf("en ruta ligera solo queda el aviso: %+v", conditions)
	}
	if len(domain.Blocking(conditions)) != 0 {
		t.Error("un aviso no bloquea")
	}
	facts.Route = "high"
	if blocking := domain.Blocking(domain.Unmet(rules, facts)); len(blocking) != 1 || blocking[0].RuleKey != domain.RuleRiskHasCharacteristics {
		t.Errorf("en ruta alta la regla del RISK bloquea: %+v", blocking)
	}
}
