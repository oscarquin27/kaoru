// Package domain modela el calendario: los hitos que mantiene change management y las alertas de atraso,
// que se calculan al leer con las ventanas de los CHG.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// DateLayout es el formato de una fecha sin hora, como la de un hito.
const DateLayout = "2006-01-02"

const maxTitle = 200

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// Snapshot es el contenido de un hito. DueDate es una fecha a medianoche UTC; Owner y CompletedAt son nil si no hay.
type Snapshot struct {
	Id          string
	Title       string
	DueDate     time.Time
	Notes       string
	Owner       *UserRef
	CompletedAt *time.Time
	CreatedBy   UserRef
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Milestone es un hito del calendario.
type Milestone struct {
	s Snapshot
}

// NewParams son los datos para registrar un hito. DueDate viene como 2026-11-30.
type NewParams struct {
	Id        string
	Title     string
	DueDate   string
	Notes     string
	Owner     *UserRef
	CreatedBy shared.UserId
	Now       time.Time
}

// New valida y registra un hito pendiente.
func New(p NewParams) (Milestone, error) {
	var errs axis.FieldErrors
	title := validTitle(p.Title, &errs)
	due := validDate(p.DueDate, &errs)
	if err := errs.Err(); err != nil {
		return Milestone{}, err
	}
	return Milestone{s: Snapshot{
		Id: p.Id, Title: title, DueDate: due, Notes: strings.TrimSpace(p.Notes), Owner: p.Owner,
		CreatedBy: UserRef{Id: p.CreatedBy}, CreatedAt: p.Now, UpdatedAt: p.Now,
	}}, nil
}

// Rehydrate arma un hito ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Milestone { return Milestone{s: s} }

// Snapshot devuelve el contenido del hito.
func (m Milestone) Snapshot() Snapshot { return m.s }

// OwnerEdit es el cambio de responsable: Set dice si se toca; Value nil lo deja sin responsable.
type OwnerEdit struct {
	Set   bool
	Value *UserRef
}

// Edit son los campos a cambiar; un puntero nil deja el campo como está.
type Edit struct {
	Title     *string
	DueDate   *string
	Notes     *string
	Owner     OwnerEdit
	Completed *bool
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply valida y aplica una edición. Marcar cumplido guarda la fecha; volver a pendiente la borra.
// Devuelve solo los campos que cambiaron de verdad.
func (m *Milestone) Apply(e Edit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := m.s
	if e.Title != nil {
		next.Title = validTitle(*e.Title, &errs)
	}
	if e.DueDate != nil {
		next.DueDate = validDate(*e.DueDate, &errs)
	}
	if e.Notes != nil {
		next.Notes = strings.TrimSpace(*e.Notes)
	}
	if e.Owner.Set {
		next.Owner = e.Owner.Value
	}
	if e.Completed != nil {
		switch {
		case *e.Completed && next.CompletedAt == nil:
			next.CompletedAt = &now
		case !*e.Completed:
			next.CompletedAt = nil
		}
	}
	if err := errs.Err(); err != nil {
		return nil, err
	}

	var changes []FieldChange
	add := func(field, old, new string) {
		if old != new {
			changes = append(changes, FieldChange{Field: field, Old: old, New: new})
		}
	}
	add("title", m.s.Title, next.Title)
	add("dueDate", m.s.DueDate.Format(DateLayout), next.DueDate.Format(DateLayout))
	add("notes", m.s.Notes, next.Notes)
	add("owner", ownerId(m.s.Owner), ownerId(next.Owner))
	add("completed", completed(m.s.CompletedAt), completed(next.CompletedAt))
	if len(changes) > 0 {
		next.UpdatedAt = now
		m.s = next
	}
	return changes, nil
}

// DaysLate son los días que pasaron desde la fecha de un hito pendiente; cero si se cumplió o no venció.
// Se cuentan en días UTC.
func (m Milestone) DaysLate(now time.Time) int {
	if m.s.CompletedAt != nil {
		return 0
	}
	return max(0, daysBetween(m.s.DueDate, now))
}

func validTitle(raw string, errs *axis.FieldErrors) string {
	title := strings.TrimSpace(raw)
	switch {
	case title == "":
		errs.Add(axis.Required("title", "Indica el título del hito."))
	case utf8.RuneCountInString(title) > maxTitle:
		errs.Add(axis.InvalidField("title", "El título admite hasta 200 caracteres."))
	}
	return title
}

func validDate(raw string, errs *axis.FieldErrors) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		errs.Add(axis.Required("dueDate", "Indica la fecha del hito."))
		return time.Time{}
	}
	date, err := time.Parse(DateLayout, raw)
	if err != nil {
		errs.Add(axis.InvalidField("dueDate", "La fecha debe tener la forma 2026-11-30."))
	}
	return date
}

func ownerId(u *UserRef) string {
	if u == nil {
		return ""
	}
	return string(u.Id)
}

func completed(at *time.Time) string {
	if at == nil {
		return "false"
	}
	return "true"
}

// daysBetween cuenta los días UTC entre la fecha de from y la de to.
func daysBetween(from, to time.Time) int {
	day := func(t time.Time) time.Time {
		t = t.UTC()
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
	return int(day(to).Sub(day(from)).Hours() / 24)
}
