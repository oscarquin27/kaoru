package domain_test

import (
	"reflect"
	"testing"

	"kaoru/internal/feature/bundle/domain"
)

func TestUnmet(t *testing.T) {
	rule := func(key string) domain.Rule { return domain.Rule{Key: key, Blocks: true} }
	high := domain.Rule{Key: domain.RuleNoOpenDefect, Params: map[string]any{"severity": "high"}, Blocks: true}
	ready := domain.Facts{
		Builds: 1, Requirements: 2, Regression: domain.RegressionFact{Exists: true, Confirmed: true}, DeployedToProduction: true,
		OpenDefects: []domain.DefectFact{{Code: "DEF-2026-0001", Severity: "low"}},
	}
	with := func(mutate func(f *domain.Facts)) domain.Facts {
		f := ready
		mutate(&f)
		return f
	}
	cases := []struct {
		name        string
		rule        domain.Rule
		facts       domain.Facts
		wantMessage string // vacío: la regla se cumple
		wantRelated []string
	}{
		{"contenido: tiene BUILD y REQ", rule(domain.RuleHasContent), ready, "", nil},
		{"contenido: vacío", rule(domain.RuleHasContent), domain.Facts{}, "El PKG está vacío: agrega al menos un BUILD y un REQ.", nil},
		{"contenido: sin BUILD", rule(domain.RuleHasContent), with(func(f *domain.Facts) { f.Builds = 0 }), "El PKG no tiene ningún BUILD.", nil},
		{"contenido: sin REQ", rule(domain.RuleHasContent), with(func(f *domain.Facts) { f.Requirements = 0 }), "El PKG no tiene ningún REQ.", nil},

		{"regresión confirmada", rule(domain.RuleRegressionConfirmed), ready, "", nil},
		{"regresión sin proponer", rule(domain.RuleRegressionConfirmed), with(func(f *domain.Facts) { f.Regression = domain.RegressionFact{} }),
			"Falta proponer y confirmar la regresión.", nil},
		{"regresión propuesta sin confirmar", rule(domain.RuleRegressionConfirmed), with(func(f *domain.Facts) { f.Regression.Confirmed = false }),
			"Falta confirmar la selección de regresión.", nil},

		{"regresión aprobada", rule(domain.RuleRegressionPassed), ready, "", nil},
		{"regresión aprobada: no depende de que esté confirmada", rule(domain.RuleRegressionPassed),
			with(func(f *domain.Facts) { f.Regression.Confirmed = false }), "", nil},
		{"regresión aprobada: sin selección no se cumple", rule(domain.RuleRegressionPassed),
			with(func(f *domain.Facts) { f.Regression = domain.RegressionFact{} }), "El PKG no tiene selección de regresión.", nil},
		{"regresión aprobada: falta un TC", rule(domain.RuleRegressionPassed),
			with(func(f *domain.Facts) { f.Regression.Unpassed = []string{"TC-2026-0112"} }),
			"Falta 1 TC sin RUN aprobado sobre un BUILD del PKG: TC-2026-0112.", []string{"TC-2026-0112"}},
		{"regresión aprobada: faltan dos TC", rule(domain.RuleRegressionPassed),
			with(func(f *domain.Facts) { f.Regression.Unpassed = []string{"TC-2026-0112", "TC-2026-0113"} }),
			"Faltan 2 TC sin RUN aprobado sobre un BUILD del PKG: TC-2026-0112, TC-2026-0113.", []string{"TC-2026-0112", "TC-2026-0113"}},

		{"defectos: solo uno bajo abierto", high, ready, "", nil},
		{"defectos: sin defectos", high, with(func(f *domain.Facts) { f.OpenDefects = nil }), "", nil},
		{"defectos: uno alto abierto", high, with(func(f *domain.Facts) {
			f.OpenDefects = append([]domain.DefectFact{{Code: "DEF-2026-0091", Severity: "high"}}, f.OpenDefects...)
		}), "Hay 1 DEF abierto de severidad alta: DEF-2026-0091.", []string{"DEF-2026-0091"}},
		{"defectos: uno alto aceptado al aprobar su REQ no bloquea", high, with(func(f *domain.Facts) {
			f.OpenDefects = append([]domain.DefectFact{{Code: "DEF-2026-0091", Severity: "high", Accepted: true}}, f.OpenDefects...)
		}), "", nil},
		{"defectos: dos altos abiertos", high, with(func(f *domain.Facts) {
			f.OpenDefects = []domain.DefectFact{{Code: "DEF-2026-0091", Severity: "high"}, {Code: "DEF-2026-0092", Severity: "high"}}
		}), "Hay 2 DEF abiertos de severidad alta: DEF-2026-0091, DEF-2026-0092.", []string{"DEF-2026-0091", "DEF-2026-0092"}},
		{"defectos: la severidad sale de la regla", domain.Rule{Key: domain.RuleNoOpenDefect, Params: map[string]any{"severity": "low"}, Blocks: true},
			ready, "Hay 1 DEF abierto de severidad baja: DEF-2026-0001.", []string{"DEF-2026-0001"}},

		{"desplegado en producción", rule(domain.RuleDeployedToProduction), ready, "", nil},
		{"sin despliegue en producción", rule(domain.RuleDeployedToProduction), with(func(f *domain.Facts) { f.DeployedToProduction = false }),
			"Falta un despliegue exitoso del PKG en un ambiente de producción.", nil},

		{"checklist firmado", rule(domain.RuleChecklistSigned),
			with(func(f *domain.Facts) { f.Production = domain.ProductionFact{ChecklistExists: true} }), "", nil},
		{"checklist sin abrir", rule(domain.RuleChecklistSigned), ready, "Falta abrir y firmar el checklist de producción.", nil},
		{"checklist: falta un paso", rule(domain.RuleChecklistSigned),
			with(func(f *domain.Facts) {
				f.Production = domain.ProductionFact{ChecklistExists: true, ChecklistPending: 1}
			}),
			"Al checklist de producción le falta 1 paso por firmar.", nil},
		{"checklist: faltan tres pasos", rule(domain.RuleChecklistSigned),
			with(func(f *domain.Facts) {
				f.Production = domain.ProductionFact{ChecklistExists: true, ChecklistPending: 3}
			}),
			"Al checklist de producción le faltan 3 pasos por firmar.", nil},

		{"diagnóstico conforme", rule(domain.RuleDiagnosticReviewed),
			with(func(f *domain.Facts) { f.Production.Diagnostic = domain.DiagnosticConforming }), "", nil},
		{"sin diagnóstico", rule(domain.RuleDiagnosticReviewed), ready, "Falta el diagnóstico posterior al despliegue.", nil},
		{"diagnóstico sin revisar", rule(domain.RuleDiagnosticReviewed),
			with(func(f *domain.Facts) { f.Production.Diagnostic = domain.DiagnosticPending }),
			"Falta que QA revise el diagnóstico posterior al despliegue.", nil},
		{"diagnóstico no conforme", rule(domain.RuleDiagnosticReviewed),
			with(func(f *domain.Facts) { f.Production.Diagnostic = "non_conforming" }), "El último diagnóstico no está conforme.", nil},

		{"prueba de humo aprobada", rule(domain.RuleSmokePassed),
			with(func(f *domain.Facts) { f.Production.Smoke = domain.SmokePassed }), "", nil},
		{"sin prueba de humo", rule(domain.RuleSmokePassed), ready, "Falta la prueba de humo posterior al despliegue.", nil},
		{"prueba de humo fallida", rule(domain.RuleSmokePassed),
			with(func(f *domain.Facts) { f.Production.Smoke = "failed" }), "La última prueba de humo falló.", nil},

		{"una regla desconocida no se cumple", rule("min_coverage"), ready, "La regla «min_coverage» no se puede evaluar. Avisa a QA.", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.Unmet([]domain.Rule{tc.rule}, tc.facts)
			if tc.wantMessage == "" {
				if len(got) != 0 {
					t.Errorf("la regla debía cumplirse: %+v", got)
				}
				return
			}
			want := []domain.Condition{{RuleKey: tc.rule.Key, Message: tc.wantMessage, Blocks: true, Related: tc.wantRelated}}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Unmet:\n got %+v\nwant %+v", got, want)
			}
		})
	}
}

func TestBlockingKeepsOnlyRulesThatBlock(t *testing.T) {
	rules := []domain.Rule{
		{Key: domain.RuleHasContent, Blocks: true},
		{Key: domain.RuleRegressionConfirmed, Blocks: false}, // solo avisa
		{Key: "desconocida", Blocks: false},                  // desconocida que solo avisa
		{Key: "otra_desconocida", Blocks: true},              // desconocida que bloquea
		{Key: domain.RuleDeployedToProduction, Blocks: true}, // se cumple
	}
	unmet := domain.Unmet(rules, domain.Facts{DeployedToProduction: true})
	var keys []string
	for _, c := range unmet {
		keys = append(keys, c.RuleKey)
	}
	if want := []string{domain.RuleHasContent, domain.RuleRegressionConfirmed, "desconocida", "otra_desconocida"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("no cumplidas = %v, quiero %v", keys, want)
	}
	keys = nil
	for _, c := range domain.Blocking(unmet) {
		keys = append(keys, c.RuleKey)
	}
	if want := []string{domain.RuleHasContent, "otra_desconocida"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("bloquean = %v, quiero %v", keys, want)
	}
	if got := domain.Blocking(nil); got != nil {
		t.Errorf("sin condiciones = %v", got)
	}
}

func TestUnmetSkipsRulesOfAnotherRoute(t *testing.T) {
	rules := []domain.Rule{
		{Key: domain.RuleChecklistSigned, Route: domain.RouteHigh, Blocks: true},
		{Key: domain.RuleSmokePassed, Route: domain.RouteLight, Blocks: true},
		{Key: domain.RuleDeployedToProduction, Blocks: true}, // sin ruta: aplica a todas
	}
	cases := []struct {
		route string
		want  []string
	}{
		{domain.RouteHigh, []string{domain.RuleChecklistSigned, domain.RuleDeployedToProduction}},
		{domain.RouteLight, []string{domain.RuleSmokePassed, domain.RuleDeployedToProduction}},
		{domain.RouteMinorFix, []string{domain.RuleDeployedToProduction}},
	}
	for _, tc := range cases {
		var keys []string
		for _, c := range domain.Unmet(rules, domain.Facts{Route: tc.route}) {
			keys = append(keys, c.RuleKey)
		}
		if !reflect.DeepEqual(keys, tc.want) {
			t.Errorf("ruta %s: no cumplidas = %v, quiero %v", tc.route, keys, tc.want)
		}
	}
}

func TestMostDemandingRoute(t *testing.T) {
	cases := []struct {
		name   string
		routes []string
		system string
		want   string
	}{
		{"sin REQ: la del sistema", nil, "light", "light"},
		{"una sola", []string{"minor_fix"}, "high", "minor_fix"},
		{"gana la alta", []string{"light", "high", "minor_fix"}, "light", "high"},
		{"ligera sobre fix menor", []string{"minor_fix", "light"}, "high", "light"},
		{"ninguna conocida: la del sistema", []string{"otra"}, "high", "high"},
	}
	for _, tc := range cases {
		if got := domain.MostDemandingRoute(tc.routes, tc.system); got != tc.want {
			t.Errorf("%s: %q, quiero %q", tc.name, got, tc.want)
		}
	}
}
