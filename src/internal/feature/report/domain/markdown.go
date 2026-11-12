package domain

import (
	"fmt"
	"strings"
	"time"
)

// Brand es la marca de Kaoru que llevan el encabezado y el pie del informe (requerimiento §7.0).
const (
	BrandName   = "kaoru"
	BrandMotto  = "Trazabilidad a prueba de errores"
	reportTitle = "Informe de conformidad"
)

var (
	routeLabels    = map[string]string{"high": "Alta", "light": "Ligera", "minor_fix": "Fix menor"}
	riskLabels     = map[string]string{"high": "Alto", "medium": "Medio", "low": "Bajo"}
	resultLabels   = map[string]string{"passed": "Aprobado", "failed": "Fallido", "blocked": "Bloqueado"}
	severityLabels = map[string]string{"high": "Alta", "medium": "Media", "low": "Baja"}
	originLabels   = map[string]string{"impact": "Impacto", "critical": "Crítico", "manual": "Agregado por QA"}
	reviewLabels   = map[string]string{"conforming": "Conforme", "non_conforming": "No conforme"}
	smokeLabels    = map[string]string{"passed": "Aprobada", "failed": "Fallida"}
)

// BrandLine es el primer renglón del Markdown. El PDF lo omite porque ya lleva la marca en el encabezado.
const BrandLine = "**" + BrandName + "** · " + BrandMotto + "\n\n"

// Title es el título de un informe, el mismo en el Markdown y en el PDF.
func Title(code string) string { return reportTitle + " " + code }

// Render escribe el informe en Markdown. Usa solo títulos, párrafos, listas, tablas y negrita, que es lo
// que pintan el visor de Kaoru y el PDF. El resultado es determinista: el mismo contenido da el mismo texto.
func Render(c Content) string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	b.WriteString(BrandLine)
	w("# %s\n\n", Title(c.Code))
	pkg := c.Package
	previous := "Ninguno: es la primera versión."
	if c.Previous != "" {
		previous = c.Previous
	}
	table(&b, []string{"Dato", "Valor"}, [][]string{
		{"PKG", pkg.Code + " — " + pkg.Title},
		{"Sistema", pkg.SystemName},
		{"Ruta", label(routeLabels, pkg.Route)},
		{"Estado del PKG al emitir", pkg.StateName},
		{"Versión del informe", fmt.Sprint(c.Version)},
		{"Reemplaza a", previous},
		{"Emitido por", c.IssuedBy},
		{"Emitido el", when(c.IssuedAt)},
	})

	w("## 1. Contenido del paquete\n\n")
	if len(pkg.Builds) == 0 {
		w("El PKG no tiene BUILD.\n\n")
	} else {
		rows := make([][]string, len(pkg.Builds))
		for i, build := range pkg.Builds {
			rows[i] = []string{build.Code, build.Version, build.Checksum}
		}
		table(&b, []string{"BUILD", "Versión", "SHA-256"}, rows)
	}
	if len(pkg.Requirements) == 0 {
		w("El PKG no tiene REQ.\n\n")
	} else {
		rows := make([][]string, len(pkg.Requirements))
		for i, r := range pkg.Requirements {
			rows[i] = []string{r.Code, r.Title, r.StateName}
		}
		table(&b, []string{"REQ", "Título", "Estado"}, rows)
	}

	w("## 2. Trazabilidad\n\n")
	w("REQ, RISK, UC, TC, el último RUN del TC sobre un BUILD del PKG y los DEF reproducidos en un RUN de ese TC.\n\n")
	if len(c.Trace) == 0 {
		w("Sin REQ que trazar.\n\n")
	} else {
		rows := make([][]string, len(c.Trace))
		for i, t := range c.Trace {
			testCase := dash(t.TestCaseCode)
			if t.Critical {
				testCase += " (crítico)"
			}
			risk := "Sin evaluar"
			if t.RiskCode != "" {
				risk = t.RiskCode + " (" + label(riskLabels, t.RiskLevel) + ")"
			}
			run := "Sin RUN"
			if t.RunCode != "" {
				run = t.RunCode + " (" + label(resultLabels, t.RunResult) + ")"
			} else if t.TestCaseCode == "" {
				run = "—"
			}
			rows[i] = []string{
				t.RequirementCode, dash(t.ChangeCode), label(routeLabels, t.Route), risk,
				dash(strings.Join(t.FunctionalCases, ", ")), testCase, run, dash(strings.Join(t.Defects, ", ")),
			}
		}
		table(&b, []string{"REQ", "CHG", "Ruta", "RISK", "UC", "TC", "Último RUN", "DEF"}, rows)
	}

	w("## 3. Ejecuciones\n\n")
	if len(c.Runs) == 0 {
		w("No hay RUN sobre los BUILD del PKG.\n\n")
	} else {
		passed := 0
		rows := make([][]string, len(c.Runs))
		for i, r := range c.Runs {
			if r.Result == "passed" {
				passed++
			}
			rows[i] = []string{r.Code, r.TestCaseCode, label(resultLabels, r.Result), r.BuildCode + " " + r.BuildVersion, r.Environment, when(r.ExecutedAt), r.ExecutedBy}
		}
		w("%s sobre los BUILD del PKG, %d aprobados.\n\n", count(len(c.Runs), "1 RUN", "%d RUN"), passed)
		table(&b, []string{"RUN", "TC", "Resultado", "BUILD", "Ambiente", "Fecha", "Ejecutó"}, rows)
	}

	w("## 4. Regresión\n\n")
	if g := c.Regression; g == nil {
		w("La regresión del PKG no se propuso.\n\n")
	} else {
		if g.ConfirmedAt != nil {
			w("Selección confirmada por %s el %s.\n\n", g.ConfirmedBy, when(*g.ConfirmedAt))
		} else {
			w("**La selección no está confirmada.**\n\n")
		}
		w("- Propuestos por impacto y criticidad: %d\n", g.Proposed)
		w("- En la selección final: %d (%d agregados y %d quitados por QA)\n", g.Included, g.Added, g.Removed)
		w("- Resultado sobre los BUILD del PKG: %d aprobados, %d fallidos, %d pendientes\n\n", g.Passed, g.Failed, g.Pending)
		if len(g.Items) > 0 {
			rows := make([][]string, len(g.Items))
			for i, it := range g.Items {
				result := "Sin RUN"
				if it.LastResult != "" {
					result = label(resultLabels, it.LastResult)
				}
				if !it.Included {
					result = "—"
				}
				rows[i] = []string{it.TestCaseCode, label(originLabels, it.Origin), yesNo(it.Proposed), yesNo(it.Included), result, dash(it.Reason), dash(it.DecidedBy)}
			}
			table(&b, []string{"TC", "Origen", "Propuesto", "Final", "Resultado", "Criterio del ajuste", "Decidió"}, rows)
		}
	}

	w("## 5. Defectos abiertos y diferidos\n\n")
	if len(c.Defects) == 0 {
		w("Ningún REQ del PKG tiene DEF sin cerrar.\n\n")
	} else {
		rows := make([][]string, len(c.Defects))
		for i, d := range c.Defects {
			deferred := "—"
			if d.DeferredBy != "" && d.DeferredAt != nil {
				deferred = fmt.Sprintf("%s, %s: %s", d.DeferredBy, when(*d.DeferredAt), dash(d.DeferredReason))
			}
			rows[i] = []string{d.Code, d.Title, d.RequirementCode, label(severityLabels, d.Severity), d.StateName, deferred}
		}
		table(&b, []string{"DEF", "Título", "REQ", "Severidad", "Estado", "Diferido por, cuándo y por qué"}, rows)
	}

	w("## 6. Checklist de producción\n\n")
	if len(c.Checklists) == 0 {
		w("El PKG no tiene checklist de producción.\n\n")
	}
	for _, chk := range c.Checklists {
		signed := 0
		rows := make([][]string, len(chk.Steps))
		for i, s := range chk.Steps {
			by, at := "Sin firmar", "—"
			if s.SignedBy != "" {
				signed++
				by, at = s.SignedBy, when(s.SignedAt)
			}
			rows[i] = []string{fmt.Sprint(s.Position), s.Title, by, at, dash(s.Comment)}
		}
		w("### %s\n\n", chk.Code)
		w("Ambiente %s, plantilla versión %d. Abierto por %s el %s. %d de %d pasos firmados.\n\n",
			chk.Environment, chk.TemplateVersion, chk.StartedBy, when(chk.CreatedAt), signed, len(chk.Steps))
		table(&b, []string{"Paso", "Título", "Firmó", "Fecha", "Comentario"}, rows)
	}

	w("## 7. Verificación posterior al despliegue\n\n")
	if len(c.Diagnostics) == 0 && len(c.SmokeTests) == 0 {
		w("No hay DIAG ni SMOKE del PKG.\n\n")
	}
	if len(c.Diagnostics) > 0 {
		rows := make([][]string, len(c.Diagnostics))
		for i, d := range c.Diagnostics {
			review := "Sin revisar"
			if d.ReviewResult != "" {
				review = fmt.Sprintf("%s, por %s el %s", label(reviewLabels, d.ReviewResult), d.ReviewedBy, when(d.ReviewedAt))
				if d.ReviewNotes != "" {
					review += ": " + d.ReviewNotes
				}
			}
			rows[i] = []string{d.Code, d.Deployment + " (" + d.Environment + ")", d.SubmittedBy + ", " + when(d.CreatedAt), dash(d.Notes), review}
		}
		table(&b, []string{"DIAG", "Despliegue", "Entregó", "Notas", "Revisión de QA"}, rows)
	}
	if len(c.SmokeTests) > 0 {
		rows := make([][]string, len(c.SmokeTests))
		for i, s := range c.SmokeTests {
			rows[i] = []string{s.Code, s.Deployment + " (" + s.Environment + ")", label(smokeLabels, s.Result), s.ExecutedBy + ", " + when(s.CreatedAt), dash(s.Notes)}
		}
		table(&b, []string{"SMOKE", "Despliegue", "Resultado", "Registró", "Notas"}, rows)
	}

	w("## Huella del informe\n\n")
	w("Kaoru emitió este informe, versión %d, sin conexión a servicios externos. "+
		"Su huella es el SHA-256 de este texto: Kaoru la guarda y la muestra junto al informe, y el PDF la lleva en el pie. "+
		"Si algo cambia, se emite una versión nueva que referencia a esta.\n\n", c.Version)
	w("%s · %s\n", BrandName, BrandMotto)
	return b.String()
}

// table escribe una tabla de Markdown. Una barra en el texto partiría la celda: se cambia por otra parecida.
func table(b *strings.Builder, head []string, rows [][]string) {
	line := func(cells []string) {
		clean := make([]string, len(cells))
		for i, c := range cells {
			clean[i] = cell(c)
		}
		b.WriteString("| " + strings.Join(clean, " | ") + " |\n")
	}
	line(head)
	rule := make([]string, len(head))
	for i := range rule {
		rule[i] = "---"
	}
	b.WriteString("|" + strings.Join(rule, "|") + "|\n")
	for _, row := range rows {
		line(row)
	}
	b.WriteString("\n")
}

func cell(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.ReplaceAll(s, "|", "¦")
}

func label(labels map[string]string, key string) string {
	if l, ok := labels[key]; ok {
		return l
	}
	return dash(key)
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func yesNo(v bool) string {
	if v {
		return "Sí"
	}
	return "No"
}

func count(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return fmt.Sprintf(many, n)
}

// when escribe una fecha en UTC, sin depender de la zona del servidor.
func when(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 UTC") }
