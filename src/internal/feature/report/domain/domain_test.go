package domain_test

import (
	"strings"
	"testing"
	"time"

	"kaoru/internal/feature/report/domain"
	shared "kaoru/internal/shared/domain"
)

var issuedAt = time.Date(2026, 11, 12, 15, 4, 5, 0, time.FixedZone("CST", -6*3600))

func sample() domain.Content {
	deferredAt := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	confirmedAt := time.Date(2026, 11, 9, 9, 0, 0, 0, time.UTC)
	return domain.Content{
		Code: "RPT-2026-0001", Version: 1, IssuedBy: "Quinn QA", IssuedAt: issuedAt,
		Package: domain.PackageInfo{
			Code: "PKG-2026-0001", Title: "Pagos | noviembre", SystemName: "Pagos", StateName: "Desplegado", Route: "high",
			Builds:       []domain.Build{{Code: "BUILD-2026-0001", Version: "1.4.0", Checksum: "abc"}},
			Requirements: []domain.Requirement{{Code: "REQ-2026-0001", Title: "Cobro", StateName: "Listo"}},
		},
		Trace: []domain.TraceRow{
			{RequirementCode: "REQ-2026-0001", ChangeCode: "CHG-2026-0001", Route: "high", RiskCode: "RISK-2026-0001", RiskLevel: "high",
				FunctionalCases: []string{"UC-2026-0001"}, TestCaseCode: "TC-2026-0001", Critical: true, RunCode: "RUN-2026-0001", RunResult: "passed"},
			{RequirementCode: "REQ-2026-0002", Route: "light"},
		},
		Runs: []domain.Run{{Code: "RUN-2026-0001", TestCaseCode: "TC-2026-0001", Result: "passed", BuildCode: "BUILD-2026-0001",
			BuildVersion: "1.4.0", Environment: "QA", ExecutedAt: issuedAt, ExecutedBy: "Quinn QA"}},
		Regression: &domain.Regression{ConfirmedBy: "Quinn QA", ConfirmedAt: &confirmedAt, Proposed: 1, Included: 1, Passed: 1,
			Items: []domain.RegressionItem{{TestCaseCode: "TC-2026-0001", Origin: "critical", Proposed: true, Included: true, LastResult: "passed"}}},
		Defects: []domain.Defect{{Code: "DEF-2026-0001", Title: "Texto\ncortado", Severity: "low", StateName: "Diferido",
			RequirementCode: "REQ-2026-0001", DeferredBy: "Ana", DeferredAt: &deferredAt, DeferredReason: "Cosmético"}},
		Checklists: []domain.Checklist{{Code: "CHK-2026-0001", Environment: "PROD", TemplateVersion: 2, StartedBy: "Carla", CreatedAt: issuedAt,
			Steps: []domain.ChecklistStep{{Position: 1, Title: "Respaldo", SignedBy: "Omar", SignedAt: issuedAt, Comment: "ok"}, {Position: 2, Title: "Avisar"}}}},
		Diagnostics: []domain.Diagnostic{{Code: "DIAG-2026-0001", Deployment: "DEP-2026-0001", Environment: "PROD", SubmittedBy: "Dev",
			CreatedAt: issuedAt, ReviewResult: "conforming", ReviewedBy: "Quinn QA", ReviewedAt: issuedAt}},
		SmokeTests: []domain.SmokeTest{{Code: "SMOKE-2026-0001", Deployment: "DEP-2026-0001", Environment: "PROD", Result: "failed",
			Notes: "Falla el login", ExecutedBy: "Quinn QA", CreatedAt: issuedAt}},
	}
}

func TestRenderWritesEverySectionWithTheBrand(t *testing.T) {
	md := domain.Render(sample())
	for _, want := range []string{
		"**kaoru** · Trazabilidad a prueba de errores\n",
		"# Informe de conformidad RPT-2026-0001\n",
		"| Emitido el | 2026-11-12 21:04 UTC |",
		"| Reemplaza a | Ninguno: es la primera versión. |",
		"| PKG | PKG-2026-0001 — Pagos ¦ noviembre |",
		"| REQ-2026-0001 | CHG-2026-0001 | Alta | RISK-2026-0001 (Alto) | UC-2026-0001 | TC-2026-0001 (crítico) | RUN-2026-0001 (Aprobado) | — |",
		"| REQ-2026-0002 | — | Ligera | Sin evaluar | — | — | — | — |",
		"1 RUN sobre los BUILD del PKG, 1 aprobados.",
		"Selección confirmada por Quinn QA el 2026-11-09 09:00 UTC.",
		"| TC-2026-0001 | Crítico | Sí | Sí | Aprobado | — | — |",
		"| DEF-2026-0001 | Texto cortado | REQ-2026-0001 | Baja | Diferido | Ana, 2026-11-10 09:00 UTC: Cosmético |",
		"1 de 2 pasos firmados.",
		"| 2 | Avisar | Sin firmar | — | — |",
		"Conforme, por Quinn QA el 2026-11-12 21:04 UTC",
		"| SMOKE-2026-0001 | DEP-2026-0001 (PROD) | Fallida |",
		"## Huella del informe",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("falta %q en\n%s", want, md)
		}
	}
	if !strings.HasSuffix(md, "kaoru · Trazabilidad a prueba de errores\n") {
		t.Errorf("el informe no cierra con la marca")
	}
}

func TestRenderPutsABlankLineBeforeEveryTable(t *testing.T) {
	lines := strings.Split(domain.Render(sample()), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "|") && i > 0 && lines[i-1] != "" && !strings.HasPrefix(lines[i-1], "|") {
			t.Errorf("la tabla de la línea %d no va precedida de una línea en blanco: %q", i+1, lines[i-1])
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	first, second := domain.Render(sample()), domain.Render(sample())
	if first != second {
		t.Fatal("el mismo contenido dio dos textos distintos")
	}
}

func TestRenderSaysWhatIsMissing(t *testing.T) {
	md := domain.Render(domain.Content{Code: "RPT-2026-0002", Version: 2, Previous: "RPT-2026-0001", IssuedAt: issuedAt})
	for _, want := range []string{
		"| Reemplaza a | RPT-2026-0001 |", "El PKG no tiene BUILD.", "Sin REQ que trazar.", "No hay RUN",
		"La regresión del PKG no se propuso.", "Ningún REQ del PKG tiene DEF sin cerrar.",
		"El PKG no tiene checklist de producción.", "No hay DIAG ni SMOKE del PKG.",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("falta %q", want)
		}
	}
}

func TestNewChainsVersionsAndHashesTheContent(t *testing.T) {
	code := shared.NewCode(shared.PrefixReport, 2026, 1)
	first := domain.New(domain.NewParams{Id: "1", Code: code, Content: "hola", IssuedBy: "u", Now: issuedAt}).Snapshot()
	if first.Version != 1 || first.PreviousId != "" {
		t.Fatalf("primera versión = %+v", first)
	}
	if first.Checksum != "b221d9dbb083a7f33428d7c2a3c3198ae925614d70210e28716ccaa7cd4ddb79" {
		t.Errorf("checksum = %s", first.Checksum)
	}
	second := domain.New(domain.NewParams{Id: "2", Code: shared.NewCode(shared.PrefixReport, 2026, 2), Previous: &first, Content: "chao", Now: issuedAt}).Snapshot()
	if second.Version != 2 || second.PreviousId != "1" || second.PreviousCode != code.String() {
		t.Errorf("segunda versión = %+v", second)
	}
}
