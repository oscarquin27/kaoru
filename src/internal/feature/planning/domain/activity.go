package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// ActivityKind es la clase de una actividad.
type ActivityKind string

// Clases de actividad. La certificación siempre es de un CHG.
const (
	KindCertification ActivityKind = "certification"
	KindRegression    ActivityKind = "regression"
	KindProduction    ActivityKind = "production"
	KindOther         ActivityKind = "other"
)

// ValidKind dice si la clase existe.
func ValidKind(k ActivityKind) bool {
	switch k {
	case KindCertification, KindRegression, KindProduction, KindOther:
		return true
	}
	return false
}

// Status es el estado de una actividad: se deduce de sus fechas, salvo la cancelada.
type Status string

// Estados de una actividad.
const (
	StatusPlanned   Status = "planned"
	StatusActive    Status = "active"
	StatusDone      Status = "done"
	StatusCancelled Status = "cancelled"
)

// ValidStatus dice si el estado existe.
func ValidStatus(s Status) bool {
	switch s {
	case StatusPlanned, StatusActive, StatusDone, StatusCancelled:
		return true
	}
	return false
}

// maxSpanDays acota una actividad: una más larga es un error de tipeo, no un plan.
const maxSpanDays = 366

// ChangeRef es el CHG (el proyecto) de una actividad.
type ChangeRef struct {
	Id     string
	Code   string
	Title  string
	System string
}

// ActivitySnapshot es el contenido de una actividad. Start y End son días a medianoche UTC, ambos incluidos.
type ActivitySnapshot struct {
	Id          string
	Code        string
	Kind        ActivityKind
	Title       string
	Change      *ChangeRef
	Start       time.Time
	End         time.Time
	Notes       string
	Operators   []UserRef
	CancelledAt *time.Time
	CreatedBy   UserRef
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Activity es un bloque de trabajo con fechas, de cero o más operadores.
type Activity struct {
	s ActivitySnapshot
}

// NewActivityParams son los datos para planificar una actividad. Las fechas vienen como 2026-11-30.
type NewActivityParams struct {
	Id        string
	Code      string
	Kind      string
	Title     string
	Change    *ChangeRef
	Start     string
	End       string
	Notes     string
	Operators []UserRef
	CreatedBy shared.UserId
	Now       time.Time
}

// NewActivity valida y planifica una actividad.
func NewActivity(p NewActivityParams) (Activity, error) {
	var errs axis.FieldErrors
	s := ActivitySnapshot{
		Id: p.Id, Code: p.Code, Kind: validKind(p.Kind, &errs), Title: validActivityTitle(p.Title, &errs),
		Change: p.Change, Start: validDay("startDate", p.Start, &errs), End: validDay("endDate", p.End, &errs),
		Notes: strings.TrimSpace(p.Notes), Operators: uniqueOperators(p.Operators),
		CreatedBy: UserRef{Id: p.CreatedBy}, CreatedAt: p.Now, UpdatedAt: p.Now,
	}
	validSpan(s, &errs)
	if err := errs.Err(); err != nil {
		return Activity{}, err
	}
	return Activity{s: s}, nil
}

// RehydrateActivity arma una actividad ya guardada. Solo lo usa el adaptador de persistencia.
func RehydrateActivity(s ActivitySnapshot) Activity { return Activity{s: s} }

// Snapshot devuelve el contenido de la actividad.
func (a Activity) Snapshot() ActivitySnapshot { return a.s }

// Cancelled dice si la actividad está cancelada.
func (a Activity) Cancelled() bool { return a.s.CancelledAt != nil }

// Status es el estado a la fecha de now: planificada si empieza después de hoy, en curso si hoy está
// dentro, terminada si ya terminó.
func (a Activity) Status(now time.Time) Status {
	today := Day(now)
	switch {
	case a.s.CancelledAt != nil:
		return StatusCancelled
	case today.Before(a.s.Start):
		return StatusPlanned
	case today.After(a.s.End):
		return StatusDone
	default:
		return StatusActive
	}
}

// Covers dice si el día de t cae dentro de la actividad.
func (a Activity) Covers(t time.Time) bool {
	day := Day(t)
	return !day.Before(a.s.Start) && !day.After(a.s.End)
}

// HasOperator dice si el usuario es operador de la actividad.
func (a Activity) HasOperator(id shared.UserId) bool {
	return slices.ContainsFunc(a.s.Operators, func(u UserRef) bool { return u.Id == id })
}

// ActivityEdit son los campos a cambiar; un puntero nil deja el campo como está.
type ActivityEdit struct {
	Kind      *string
	Title     *string
	Start     *string
	End       *string
	Notes     *string
	Change    ChangeEdit
	Operators *[]UserRef
	Cancelled *bool
}

// ChangeEdit es el cambio de CHG: Set dice si se toca; Value nil deja la actividad sin CHG.
type ChangeEdit struct {
	Set   bool
	Value *ChangeRef
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (a *Activity) Apply(e ActivityEdit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := a.s
	if e.Kind != nil {
		next.Kind = validKind(*e.Kind, &errs)
	}
	if e.Title != nil {
		next.Title = validActivityTitle(*e.Title, &errs)
	}
	if e.Start != nil {
		next.Start = validDay("startDate", *e.Start, &errs)
	}
	if e.End != nil {
		next.End = validDay("endDate", *e.End, &errs)
	}
	if e.Notes != nil {
		next.Notes = strings.TrimSpace(*e.Notes)
	}
	if e.Change.Set {
		next.Change = e.Change.Value
	}
	if e.Operators != nil {
		next.Operators = uniqueOperators(*e.Operators)
	}
	if e.Cancelled != nil {
		switch {
		case *e.Cancelled && next.CancelledAt == nil:
			next.CancelledAt = &now
		case !*e.Cancelled:
			next.CancelledAt = nil
		}
	}
	validSpan(next, &errs)
	if err := errs.Err(); err != nil {
		return nil, err
	}

	var changes []FieldChange
	add := func(field, old, new string) {
		if old != new {
			changes = append(changes, FieldChange{Field: field, Old: old, New: new})
		}
	}
	add("kind", string(a.s.Kind), string(next.Kind))
	add("title", a.s.Title, next.Title)
	add("startDate", a.s.Start.Format(DateLayout), next.Start.Format(DateLayout))
	add("endDate", a.s.End.Format(DateLayout), next.End.Format(DateLayout))
	add("notes", a.s.Notes, next.Notes)
	add("change", changeCode(a.s.Change), changeCode(next.Change))
	add("operators", operatorIds(a.s.Operators), operatorIds(next.Operators))
	add("cancelled", boolText(a.s.CancelledAt != nil), boolText(next.CancelledAt != nil))
	if len(changes) > 0 {
		next.UpdatedAt = now
		a.s = next
	}
	return changes, nil
}

// ActivityRef es lo que se muestra de la otra actividad de un solape.
type ActivityRef struct {
	Code  string
	Title string
	Kind  ActivityKind
}

// Ref resume la actividad.
func (a Activity) Ref() ActivityRef {
	return ActivityRef{Code: a.s.Code, Title: a.s.Title, Kind: a.s.Kind}
}

// Overlap es un operador con dos actividades que se cruzan entre From y To (días incluidos).
type Overlap struct {
	Operator UserRef
	First    ActivityRef
	Second   ActivityRef
	From     time.Time
	To       time.Time
}

// Days son los días que se cruzan.
func (o Overlap) Days() int { return daysBetween(o.From, o.To) + 1 }

// Overlaps devuelve los solapes entre las actividades: por cada operador, cada par de actividades no
// canceladas suyas que comparten al menos un día. Ordena por operador, fecha y códigos.
func Overlaps(activities []Activity) []Overlap {
	byOperator := map[shared.UserId][]Activity{}
	names := map[shared.UserId]UserRef{}
	for _, a := range activities {
		if a.Cancelled() {
			continue
		}
		for _, op := range a.s.Operators {
			byOperator[op.Id] = append(byOperator[op.Id], a)
			names[op.Id] = op
		}
	}
	out := []Overlap{}
	for id, list := range byOperator {
		slices.SortFunc(list, compareActivities)
		for i := range list {
			for j := i + 1; j < len(list); j++ {
				if o, ok := cross(names[id], list[i], list[j]); ok {
					out = append(out, o)
				}
			}
		}
	}
	slices.SortFunc(out, func(a, b Overlap) int {
		return cmp.Or(cmp.Compare(a.Operator.Name, b.Operator.Name), cmp.Compare(a.Operator.Id, b.Operator.Id),
			a.From.Compare(b.From), cmp.Compare(a.First.Code, b.First.Code), cmp.Compare(a.Second.Code, b.Second.Code))
	})
	return out
}

// OverlapsOf devuelve los solapes de una actividad con las demás: los de sus operadores con otras
// actividades no canceladas. La actividad va siempre como First.
func OverlapsOf(target Activity, others []Activity) []Overlap {
	out := []Overlap{}
	if target.Cancelled() {
		return out
	}
	for _, op := range target.s.Operators {
		for _, other := range others {
			if other.s.Code == target.s.Code || other.Cancelled() || !other.HasOperator(op.Id) {
				continue
			}
			if o, ok := cross(op, target, other); ok {
				o.First, o.Second = target.Ref(), other.Ref()
				out = append(out, o)
			}
		}
	}
	slices.SortFunc(out, func(a, b Overlap) int {
		return cmp.Or(cmp.Compare(a.Operator.Name, b.Operator.Name), a.From.Compare(b.From), cmp.Compare(a.Second.Code, b.Second.Code))
	})
	return out
}

func cross(op UserRef, a, b Activity) (Overlap, bool) {
	from := a.s.Start
	if b.s.Start.After(from) {
		from = b.s.Start
	}
	to := a.s.End
	if b.s.End.Before(to) {
		to = b.s.End
	}
	if from.After(to) {
		return Overlap{}, false
	}
	return Overlap{Operator: op, First: a.Ref(), Second: b.Ref(), From: from, To: to}, true
}

func compareActivities(a, b Activity) int {
	return cmp.Or(a.s.Start.Compare(b.s.Start), a.s.End.Compare(b.s.End), cmp.Compare(a.s.Code, b.s.Code))
}

// SortActivities ordena por inicio, fin y código.
func SortActivities(list []Activity) { slices.SortFunc(list, compareActivities) }

// CertificationState es cómo está planificada la certificación de un CHG a una fecha.
type CertificationState string

// Estados de la certificación de un CHG.
const (
	CertificationNone        CertificationState = "none"
	CertificationNoOperators CertificationState = "no_operators"
	CertificationOutside     CertificationState = "outside"
	CertificationReady       CertificationState = "ready"
)

// Certification es el resultado de revisar la certificación de un CHG. From y To son el período que
// cubre hoy, o el más próximo si hoy queda fuera.
type Certification struct {
	State CertificationState
	From  time.Time
	To    time.Time
}

// CheckCertification revisa las actividades de certificación de un CHG: lista si una no cancelada con
// operadores cubre hoy; si no, dice qué falta.
func CheckCertification(activities []Activity, now time.Time) Certification {
	var staffed []Activity
	planned := false
	for _, a := range activities {
		if a.Cancelled() || a.s.Kind != KindCertification {
			continue
		}
		planned = true
		if len(a.s.Operators) > 0 {
			staffed = append(staffed, a)
		}
	}
	switch {
	case !planned:
		return Certification{State: CertificationNone}
	case len(staffed) == 0:
		return Certification{State: CertificationNoOperators}
	}
	today := Day(now)
	nearest := staffed[0]
	for _, a := range staffed {
		if a.Covers(now) {
			return Certification{State: CertificationReady, From: a.s.Start, To: a.s.End}
		}
		if distance(a, today) < distance(nearest, today) {
			nearest = a
		}
	}
	return Certification{State: CertificationOutside, From: nearest.s.Start, To: nearest.s.End}
}

func distance(a Activity, day time.Time) int {
	if day.Before(a.s.Start) {
		return daysBetween(day, a.s.Start)
	}
	return daysBetween(a.s.End, day)
}

// Day es el día UTC de t, a medianoche.
func Day(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func validKind(raw string, errs *axis.FieldErrors) ActivityKind {
	kind := ActivityKind(strings.TrimSpace(raw))
	if !ValidKind(kind) {
		errs.Add(axis.InvalidField("kind", "Elige la clase: certificación, regresión, pase a producción u otra."))
	}
	return kind
}

func validActivityTitle(raw string, errs *axis.FieldErrors) string {
	title := strings.TrimSpace(raw)
	switch {
	case title == "":
		errs.Add(axis.Required("title", "Indica el título de la actividad."))
	case len([]rune(title)) > maxTitle:
		errs.Add(axis.InvalidField("title", "El título admite hasta 200 caracteres."))
	}
	return title
}

func validDay(field, raw string, errs *axis.FieldErrors) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		errs.Add(axis.Required(field, "Indica la fecha."))
		return time.Time{}
	}
	day, err := time.Parse(DateLayout, raw)
	if err != nil {
		errs.Add(axis.InvalidField(field, "La fecha debe tener la forma 2026-11-30."))
	}
	return day
}

// validSpan revisa lo que depende de varios campos: el orden de las fechas, el largo y el CHG de la certificación.
func validSpan(s ActivitySnapshot, errs *axis.FieldErrors) {
	if !s.Start.IsZero() && !s.End.IsZero() {
		switch {
		case s.End.Before(s.Start):
			errs.Add(axis.InvalidField("endDate", "La fecha final no puede ser anterior a la inicial."))
		case daysBetween(s.Start, s.End) >= maxSpanDays:
			errs.Add(axis.InvalidField("endDate", "Una actividad dura como máximo un año."))
		}
	}
	if s.Kind == KindCertification && s.Change == nil {
		errs.Add(axis.Required("change", "La certificación es de un CHG: indica cuál."))
	}
}

// uniqueOperators quita repetidos y ordena por nombre, para comparar y mostrar siempre igual.
func uniqueOperators(in []UserRef) []UserRef {
	out := make([]UserRef, 0, len(in))
	seen := map[shared.UserId]bool{}
	for _, u := range in {
		if !seen[u.Id] {
			seen[u.Id] = true
			out = append(out, u)
		}
	}
	slices.SortFunc(out, func(a, b UserRef) int { return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Id, b.Id)) })
	return out
}

func operatorIds(list []UserRef) string {
	ids := make([]string, len(list))
	for i, u := range list {
		ids[i] = string(u.Id)
	}
	slices.Sort(ids)
	return strings.Join(ids, ",")
}

func changeCode(c *ChangeRef) string {
	if c == nil {
		return ""
	}
	return c.Code
}

func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
