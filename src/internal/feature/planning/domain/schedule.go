package domain

import (
	"cmp"
	"slices"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Largo de la planificación por omisión y máximo, en días.
const (
	DefaultScheduleDays = 42
	MaxScheduleDays     = 92
)

// Range son los días de una planificación, ambos incluidos.
type Range struct {
	From time.Time
	To   time.Time
}

// ScheduleRange resuelve el rango pedido: sin fechas, seis semanas desde el lunes de esta semana; con
// una sola, seis semanas desde ella o hasta ella. Valida el orden y el largo máximo.
func ScheduleRange(from, to *time.Time, now time.Time) (Range, error) {
	var r Range
	switch {
	case from != nil && to != nil:
		r = Range{From: Day(*from), To: Day(*to)}
	case from != nil:
		r = Range{From: Day(*from), To: Day(*from).AddDate(0, 0, DefaultScheduleDays-1)}
	case to != nil:
		r = Range{From: Day(*to).AddDate(0, 0, -(DefaultScheduleDays - 1)), To: Day(*to)}
	default:
		today := Day(now)
		monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
		r = Range{From: monday, To: monday.AddDate(0, 0, DefaultScheduleDays-1)}
	}
	switch {
	case r.To.Before(r.From):
		return Range{}, axis.Invalid(axis.InvalidField("to", "La fecha final no puede ser anterior a la inicial."))
	case daysBetween(r.From, r.To) >= MaxScheduleDays:
		return Range{}, axis.Invalid(axis.InvalidField("to", "La planificación muestra como máximo 92 días."))
	}
	return r, nil
}

// Intersects dice si los días de [start, end] tocan el rango.
func (r Range) Intersects(start, end time.Time) bool {
	return !Day(end).Before(r.From) && !Day(start).After(r.To)
}

// OperatorPlan son las actividades de un operador en el rango y sus solapes.
type OperatorPlan struct {
	Operator   UserRef
	Activities []Activity
	Overlaps   []Overlap
}

// ProjectPlan es un CHG con sus períodos de certificación en el rango.
type ProjectPlan struct {
	Change         ChangeRef
	Certifications []Activity
}

// Schedule es la planificación de un rango: quién hace qué, qué se cruza y qué proyectos coinciden.
type Schedule struct {
	Range      Range
	Operators  []OperatorPlan
	Unassigned []Activity
	Projects   []ProjectPlan
	Windows    []Window
	Milestones []Milestone
}

// OverlapCount cuenta los solapes de todos los operadores.
func (s Schedule) OverlapCount() int {
	n := 0
	for _, op := range s.Operators {
		n += len(op.Overlaps)
	}
	return n
}

// ActivityCount cuenta las actividades distintas del rango.
func (s Schedule) ActivityCount() int {
	seen := map[string]bool{}
	for _, op := range s.Operators {
		for _, a := range op.Activities {
			seen[a.s.Code] = true
		}
	}
	for _, a := range s.Unassigned {
		seen[a.s.Code] = true
	}
	return len(seen)
}

// BuildSchedule arma la planificación con las actividades, ventanas e hitos que tocan el rango; descarta
// lo cancelado y lo que queda fuera. Con operator, solo esa fila de operadores.
func BuildSchedule(r Range, activities []Activity, windows []Window, milestones []Milestone, operator shared.UserId) Schedule {
	s := Schedule{Range: r, Operators: []OperatorPlan{}, Unassigned: []Activity{}, Projects: []ProjectPlan{}, Windows: []Window{}, Milestones: []Milestone{}}
	var live []Activity
	for _, a := range activities {
		if !a.Cancelled() && r.Intersects(a.s.Start, a.s.End) {
			live = append(live, a)
		}
	}
	SortActivities(live)

	rows := map[shared.UserId]*OperatorPlan{}
	projects := map[string]*ProjectPlan{}
	for _, a := range live {
		if len(a.s.Operators) == 0 {
			s.Unassigned = append(s.Unassigned, a)
		}
		for _, op := range a.s.Operators {
			if operator != "" && op.Id != operator {
				continue
			}
			row, ok := rows[op.Id]
			if !ok {
				row = &OperatorPlan{Operator: op, Overlaps: []Overlap{}}
				rows[op.Id] = row
			}
			row.Activities = append(row.Activities, a)
		}
		if a.s.Kind == KindCertification && a.s.Change != nil {
			project, ok := projects[a.s.Change.Code]
			if !ok {
				project = &ProjectPlan{Change: *a.s.Change}
				projects[a.s.Change.Code] = project
			}
			project.Certifications = append(project.Certifications, a)
		}
	}
	for _, o := range Overlaps(live) {
		if row, ok := rows[o.Operator.Id]; ok && r.Intersects(o.From, o.To) {
			row.Overlaps = append(row.Overlaps, o)
		}
	}
	for _, row := range rows {
		s.Operators = append(s.Operators, *row)
	}
	slices.SortFunc(s.Operators, func(a, b OperatorPlan) int {
		return cmp.Or(cmp.Compare(a.Operator.Name, b.Operator.Name), cmp.Compare(a.Operator.Id, b.Operator.Id))
	})
	for _, p := range projects {
		s.Projects = append(s.Projects, *p)
	}
	slices.SortFunc(s.Projects, func(a, b ProjectPlan) int {
		return cmp.Or(compareActivities(a.Certifications[0], b.Certifications[0]), cmp.Compare(a.Change.Code, b.Change.Code))
	})
	for _, w := range windows {
		if r.Intersects(w.Start, w.End) {
			s.Windows = append(s.Windows, w)
		}
	}
	for _, m := range milestones {
		if due := m.Snapshot().DueDate; r.Intersects(due, due) {
			s.Milestones = append(s.Milestones, m)
		}
	}
	return s
}
