package domain

import (
	"fmt"
	"slices"
	"strings"
)

// Rule es una regla de bloqueo de una transición, tal como la configura workflow.
type Rule struct {
	Key    string
	Params map[string]any
	// Route limita la regla a una ruta; vacío aplica a todas.
	Route string
	// Blocks en falso significa que la regla solo avisa.
	Blocks bool
}

// Claves de las reglas que este dominio sabe evaluar.
const (
	RuleHasContent           = "package_has_content"
	RuleRegressionConfirmed  = "regression_confirmed"
	RuleRegressionPassed     = "regression_passed"
	RuleNoOpenDefect         = "no_open_defect_with_severity"
	RuleDeployedToProduction = "deployed_to_production"
	RuleChecklistSigned      = "checklist_signed"
	RuleDiagnosticReviewed   = "diagnostic_reviewed"
	RuleSmokePassed          = "smoke_passed"
)

// Rutas, de la más exigente a la menos (requerimiento §8.2).
const (
	RouteHigh     = "high"
	RouteLight    = "light"
	RouteMinorFix = "minor_fix"
)

// MostDemandingRoute es la ruta de un PKG: la más exigente de las de sus REQ. Sin REQ, o si
// ninguna es una ruta conocida, es la de su sistema.
func MostDemandingRoute(requirementRoutes []string, systemRoute string) string {
	for _, route := range []string{RouteHigh, RouteLight, RouteMinorFix} {
		if slices.Contains(requirementRoutes, route) {
			return route
		}
	}
	return systemRoute
}

// RegressionFact es lo que el PKG necesita saber de su selección de regresión.
type RegressionFact struct {
	// Exists es false si la regresión todavía no se propuso.
	Exists    bool
	Confirmed bool
	// Unpassed son los códigos de los TC de la selección final cuyo último RUN sobre un BUILD
	// del PKG no está aprobado.
	Unpassed []string
}

// DefectFact es un defecto abierto de un REQ del PKG.
type DefectFact struct {
	Code     string
	Severity string
}

// ProductionFact es lo que el PKG necesita saber de producción: lo último de cada cosa.
type ProductionFact struct {
	// ChecklistExists es false si el PKG no tiene ningún checklist abierto.
	ChecklistExists bool
	// ChecklistPending son los pasos sin firmar del último checklist.
	ChecklistPending int
	// Diagnostic es el estado del último DIAG: vacío si no hay, DiagnosticPending si falta
	// revisarlo, o el resultado de su revisión.
	Diagnostic string
	// Smoke es el resultado del último SMOKE; vacío si no hay.
	Smoke string
}

// Estados del último DIAG y del último SMOKE que las reglas reconocen.
const (
	DiagnosticPending    = "pending"
	DiagnosticConforming = "conforming"
	SmokePassed          = "passed"
)

// Facts son los hechos con los que se evalúan las reglas de una transición.
type Facts struct {
	// Route es la ruta del PKG: decide qué reglas aplican.
	Route                string
	Builds               int
	Requirements         int
	Regression           RegressionFact
	OpenDefects          []DefectFact
	DeployedToProduction bool
	Production           ProductionFact
}

// Condition es una regla que no se cumple: lo que falta para avanzar.
type Condition struct {
	RuleKey string
	Message string
	Blocks  bool
	// Related son los IDs legibles de lo que hay que resolver.
	Related []string
}

// Unmet evalúa las reglas de una transición que aplican a la ruta del PKG y devuelve las que no se cumplen.
func Unmet(rules []Rule, f Facts) []Condition {
	var conditions []Condition
	for _, rule := range rules {
		if rule.Route != "" && rule.Route != f.Route {
			continue
		}
		if message, related, ok := evaluate(rule, f); !ok {
			conditions = append(conditions, Condition{RuleKey: rule.Key, Message: message, Blocks: rule.Blocks, Related: related})
		}
	}
	return conditions
}

// Blocking devuelve solo las condiciones que impiden la transición.
func Blocking(conditions []Condition) []Condition {
	var out []Condition
	for _, c := range conditions {
		if c.Blocks {
			out = append(out, c)
		}
	}
	return out
}

// evaluate dice si la regla se cumple; si no, con qué mensaje y qué IDs relacionados.
func evaluate(rule Rule, f Facts) (message string, related []string, ok bool) {
	switch rule.Key {
	case RuleHasContent:
		switch {
		case f.Builds == 0 && f.Requirements == 0:
			return "El PKG está vacío: agrega al menos un BUILD y un REQ.", nil, false
		case f.Builds == 0:
			return "El PKG no tiene ningún BUILD.", nil, false
		case f.Requirements == 0:
			return "El PKG no tiene ningún REQ.", nil, false
		}
	case RuleRegressionConfirmed:
		switch {
		case !f.Regression.Exists:
			return "Falta proponer y confirmar la regresión.", nil, false
		case !f.Regression.Confirmed:
			return "Falta confirmar la selección de regresión.", nil, false
		}
	case RuleRegressionPassed:
		if !f.Regression.Exists {
			return "El PKG no tiene selección de regresión.", nil, false
		}
		if pending := f.Regression.Unpassed; len(pending) > 0 {
			return fmt.Sprintf("%s sin RUN aprobado sobre un BUILD del PKG: %s.",
				plural(len(pending), "Falta 1 TC", "Faltan %d TC"), strings.Join(pending, ", ")), pending, false
		}
	case RuleNoOpenDefect:
		severity, _ := rule.Params["severity"].(string)
		for _, d := range f.OpenDefects {
			if d.Severity == severity {
				related = append(related, d.Code)
			}
		}
		if len(related) > 0 {
			return fmt.Sprintf("%s de severidad %s: %s.",
				plural(len(related), "Hay 1 DEF abierto", "Hay %d DEF abiertos"), severityName(severity), strings.Join(related, ", ")), related, false
		}
	case RuleDeployedToProduction:
		if !f.DeployedToProduction {
			return "Falta un despliegue exitoso del PKG en un ambiente de producción.", nil, false
		}
	case RuleChecklistSigned:
		switch p := f.Production; {
		case !p.ChecklistExists:
			return "Falta abrir y firmar el checklist de producción.", nil, false
		case p.ChecklistPending > 0:
			return fmt.Sprintf("Al checklist de producción %s por firmar.",
				plural(p.ChecklistPending, "le falta 1 paso", "le faltan %d pasos")), nil, false
		}
	case RuleDiagnosticReviewed:
		switch f.Production.Diagnostic {
		case DiagnosticConforming:
		case "":
			return "Falta el diagnóstico posterior al despliegue.", nil, false
		case DiagnosticPending:
			return "Falta que QA revise el diagnóstico posterior al despliegue.", nil, false
		default:
			return "El último diagnóstico no está conforme.", nil, false
		}
	case RuleSmokePassed:
		switch f.Production.Smoke {
		case SmokePassed:
		case "":
			return "Falta la prueba de humo posterior al despliegue.", nil, false
		default:
			return "La última prueba de humo falló.", nil, false
		}
	default:
		// Una regla que el dominio no conoce bloquea: mejor parar que dejar pasar sin evaluar.
		return fmt.Sprintf("La regla «%s» no se puede evaluar. Avisa a QA.", rule.Key), nil, false
	}
	return "", nil, true
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return fmt.Sprintf(many, n)
}

func severityName(s string) string {
	switch s {
	case "high":
		return "alta"
	case "medium":
		return "media"
	case "low":
		return "baja"
	default:
		return s
	}
}
