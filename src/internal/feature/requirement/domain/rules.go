package domain

import (
	"fmt"
	"strings"
	"time"
)

// EntryItem es un ítem del contrato de entrada dev → QA con su estado para un REQ.
type EntryItem struct {
	Key       string
	Label     string
	Done      bool
	Note      string
	CheckedBy *UserRef
	CheckedAt *time.Time
}

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
	RuleRiskAssessed           = "risk_assessed"
	RuleRiskHasCharacteristics = "risk_has_quality_characteristics"
	RuleEntryContractComplete  = "entry_contract_complete"
	RuleMinCoverage            = "min_coverage"
	RuleNoOpenDefect           = "no_open_defect_with_severity"
	RuleLinksHaveSnapshot      = "links_have_snapshot"
	RuleCertificationPlanned   = "certification_planned"
)

// Estados de la certificación del CHG de un REQ, como los calcula planning.
const (
	CertificationNone        = "none"
	CertificationNoOperators = "no_operators"
	CertificationOutside     = "outside"
	CertificationReady       = "ready"
)

// CertificationFact es cómo está planificada hoy la certificación del CHG del REQ. Nil si el REQ no
// tiene CHG. From y To son el período más próximo, como 2026-11-30.
type CertificationFact struct {
	ChangeCode string
	State      string
	From       string
	To         string
}

// RiskFact es lo que el REQ necesita saber de su RISK.
type RiskFact struct {
	Level           string
	Characteristics int
}

// CoverageFact es la cobertura del REQ: TC requeridos y cuáles no tienen un RUN aprobado.
type CoverageFact struct {
	Required int
	// Pending son los códigos de los TC cuyo último RUN no está aprobado.
	Pending []string
}

// DefectFact es un defecto abierto del REQ.
type DefectFact struct {
	Code     string
	Severity string
}

// Facts son los hechos con los que se evalúan las reglas de una transición.
type Facts struct {
	Route string
	// Risk es nil si el riesgo no está evaluado.
	Risk                 *RiskFact
	EntryItems           []EntryItem
	Coverage             CoverageFact
	OpenDefects          []DefectFact
	LinksWithoutSnapshot int
	Certification        *CertificationFact
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
	case RuleRiskAssessed:
		if f.Risk == nil {
			return "Falta evaluar el riesgo.", nil, false
		}
	case RuleRiskHasCharacteristics:
		if f.Risk == nil || f.Risk.Characteristics == 0 {
			return "El RISK no indica qué características de calidad (ISO 25010) están en juego.", nil, false
		}
	case RuleEntryContractComplete:
		var missing []string
		for _, item := range f.EntryItems {
			if !item.Done {
				missing = append(missing, item.Label)
			}
		}
		if len(missing) > 0 {
			return fmt.Sprintf("%s del contrato de entrada: %s.",
				plural(len(missing), "Falta 1 ítem", "Faltan %d ítems"), strings.Join(missing, "; ")), nil, false
		}
	case RuleMinCoverage:
		return evaluateCoverage(rule, f)
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
	case RuleLinksHaveSnapshot:
		if f.LinksWithoutSnapshot > 0 {
			return plural(f.LinksWithoutSnapshot, "1 enlace de Google Sheets no tiene", "%d enlaces de Google Sheets no tienen") +
				" instantánea. Para el informe de conformidad cuenta la instantánea, no el enlace.", nil, false
		}
	case RuleCertificationPlanned:
		return evaluateCertification(f.Certification)
	default:
		// Una regla que el dominio no conoce bloquea: mejor parar que dejar pasar sin evaluar.
		return fmt.Sprintf("La regla «%s» no se puede evaluar. Avisa a QA.", rule.Key), nil, false
	}
	return "", nil, true
}

func evaluateCoverage(rule Rule, f Facts) (string, []string, bool) {
	if f.Risk == nil {
		return "Falta evaluar el riesgo: de él depende la cobertura mínima.", nil, false
	}
	if f.Coverage.Required == 0 {
		return "El REQ no tiene TC definidos.", nil, false
	}
	// Un porcentaje que falta o no es un número exige cobertura total.
	minimum := 100.0
	if configured, ok := rule.Params[f.Risk.Level].(float64); ok {
		minimum = configured
	}
	pending := len(f.Coverage.Pending)
	passed := f.Coverage.Required - pending
	if float64(passed)*100 >= minimum*float64(f.Coverage.Required) {
		return "", nil, true
	}
	return fmt.Sprintf("%s sin RUN aprobado: %s. Riesgo %s exige %g %% de cobertura; hay %d de %d.",
		plural(pending, "Falta 1 TC", "Faltan %d TC"), strings.Join(f.Coverage.Pending, ", "),
		levelName(f.Risk.Level), minimum, passed, f.Coverage.Required), f.Coverage.Pending, false
}

// evaluateCertification pide que el CHG del REQ tenga reservado, con operadores, un período de
// certificación que incluya hoy. Un REQ sin CHG la cumple: no hay proyecto que planificar.
func evaluateCertification(c *CertificationFact) (string, []string, bool) {
	if c == nil || c.State == CertificationReady {
		return "", nil, true
	}
	related := []string{c.ChangeCode}
	switch c.State {
	case CertificationNoOperators:
		return fmt.Sprintf("La certificación de %s no tiene operadores asignados.", c.ChangeCode), related, false
	case CertificationOutside:
		return fmt.Sprintf("Hoy está fuera del período de certificación de %s (del %s al %s).", c.ChangeCode, c.From, c.To), related, false
	default:
		return fmt.Sprintf("%s no tiene reservado el período de certificación. Planifícalo en Planificación.", c.ChangeCode), related, false
	}
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

func levelName(l string) string {
	switch l {
	case "high":
		return "alto"
	case "medium":
		return "medio"
	case "low":
		return "bajo"
	default:
		return l
	}
}
