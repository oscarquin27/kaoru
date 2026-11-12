package domain

import (
	"fmt"
	"strings"
)

// Rule es una regla de bloqueo de una transición, tal como la configura workflow.
type Rule struct {
	Key    string
	Params map[string]any
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
)

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

// Facts son los hechos con los que se evalúan las reglas de una transición.
type Facts struct {
	Builds               int
	Requirements         int
	Regression           RegressionFact
	OpenDefects          []DefectFact
	DeployedToProduction bool
}

// Condition es una regla que no se cumple: lo que falta para avanzar.
type Condition struct {
	RuleKey string
	Message string
	Blocks  bool
	// Related son los IDs legibles de lo que hay que resolver.
	Related []string
}

// Unmet evalúa las reglas de una transición y devuelve las que no se cumplen.
func Unmet(rules []Rule, f Facts) []Condition {
	var conditions []Condition
	for _, rule := range rules {
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
