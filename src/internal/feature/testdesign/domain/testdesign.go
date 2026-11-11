// Package domain modela el diseño de pruebas: los UC (casos de uso funcionales) y los TC que los validan.
package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

const maxTitleLength = 200

// RequirementRef es el REQ al que pertenece un UC.
type RequirementRef struct {
	Id        string
	Code      string
	SystemKey string
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// FunctionalCaseSnapshot es el contenido de un UC.
type FunctionalCaseSnapshot struct {
	Id          string
	Code        shared.Code
	Requirement RequirementRef
	Title       string
	Description string
	// TestCases son los códigos de los TC que validan este UC; solo lectura.
	TestCases []string
	CreatedBy shared.UserId
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FunctionalCase es un UC-: un caso de uso funcional de un REQ.
type FunctionalCase struct {
	s FunctionalCaseSnapshot
}

// NewFunctionalCase valida y crea un UC.
func NewFunctionalCase(id string, code shared.Code, requirement RequirementRef, title, description string,
	by shared.UserId, now time.Time) (FunctionalCase, error) {
	var errs axis.FieldErrors
	title = validTitle(title, &errs)
	if err := errs.Err(); err != nil {
		return FunctionalCase{}, err
	}
	return FunctionalCase{s: FunctionalCaseSnapshot{
		Id: id, Code: code, Requirement: requirement, Title: title, Description: strings.TrimSpace(description),
		TestCases: []string{}, CreatedBy: by, CreatedAt: now, UpdatedAt: now,
	}}, nil
}

// RehydrateFunctionalCase arma un UC ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateFunctionalCase(s FunctionalCaseSnapshot) FunctionalCase { return FunctionalCase{s: s} }

// Snapshot devuelve el contenido del UC.
func (f FunctionalCase) Snapshot() FunctionalCaseSnapshot { return f.s }

// Apply cambia el título o la descripción; un puntero nil deja el campo como está.
func (f *FunctionalCase) Apply(title, description *string, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := f.s
	if title != nil {
		next.Title = validTitle(*title, &errs)
	}
	if description != nil {
		next.Description = strings.TrimSpace(*description)
	}
	if err := errs.Err(); err != nil {
		return nil, err
	}
	var changes []FieldChange
	if next.Title != f.s.Title {
		changes = append(changes, FieldChange{Field: "title", Old: f.s.Title, New: next.Title})
	}
	if next.Description != f.s.Description {
		changes = append(changes, FieldChange{Field: "description", Old: f.s.Description, New: next.Description})
	}
	if len(changes) > 0 {
		next.UpdatedAt = now
		f.s = next
	}
	return changes, nil
}

// FunctionalCaseRef es un UC visto desde un TC.
type FunctionalCaseRef struct {
	Id        string
	Code      string
	SystemKey string
}

// TestCaseSnapshot es el contenido de un TC.
type TestCaseSnapshot struct {
	Id             string
	Code           shared.Code
	SystemKey      string
	SystemName     string
	Title          string
	Preconditions  string
	Steps          string
	ExpectedResult string
	IsCritical     bool
	// RetiredAt no es nil si el TC está retirado: conserva sus ejecuciones pero ya no se exige.
	RetiredAt       *time.Time
	FunctionalCases []FunctionalCaseRef
	CreatedBy       shared.UserId
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TestCase es un TC-: un caso de prueba que valida uno o más UC.
type TestCase struct {
	s TestCaseSnapshot
}

// NewTestCaseParams son los datos para definir un TC.
type NewTestCaseParams struct {
	Id              string
	Code            shared.Code
	SystemKey       string
	SystemName      string
	Title           string
	Preconditions   string
	Steps           string
	ExpectedResult  string
	IsCritical      bool
	FunctionalCases []FunctionalCaseRef
	CreatedBy       shared.UserId
	Now             time.Time
}

// NewTestCase valida y define un TC.
func NewTestCase(p NewTestCaseParams) (TestCase, error) {
	var errs axis.FieldErrors
	s := TestCaseSnapshot{
		Id: p.Id, Code: p.Code, SystemKey: p.SystemKey, SystemName: p.SystemName,
		Title: validTitle(p.Title, &errs), Preconditions: strings.TrimSpace(p.Preconditions),
		Steps:          validRequired("steps", p.Steps, "Escribe los pasos.", &errs),
		ExpectedResult: validRequired("expectedResult", p.ExpectedResult, "Escribe el resultado esperado.", &errs),
		IsCritical:     p.IsCritical, FunctionalCases: validFunctionalCases(p.FunctionalCases, p.SystemKey, &errs),
		CreatedBy: p.CreatedBy, CreatedAt: p.Now, UpdatedAt: p.Now,
	}
	if err := errs.Err(); err != nil {
		return TestCase{}, err
	}
	return TestCase{s: s}, nil
}

// RehydrateTestCase arma un TC ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateTestCase(s TestCaseSnapshot) TestCase { return TestCase{s: s} }

// Snapshot devuelve el contenido del TC.
func (t TestCase) Snapshot() TestCaseSnapshot { return t.s }

// TestCaseEdit son los campos a cambiar; nil deja el campo como está.
type TestCaseEdit struct {
	Title          *string
	Preconditions  *string
	Steps          *string
	ExpectedResult *string
	IsCritical     *bool
	// FunctionalCases reemplaza los UC que valida el TC.
	FunctionalCases *[]FunctionalCaseRef
	// Retired retira el TC o lo vuelve a activar.
	Retired *bool
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (t *TestCase) Apply(e TestCaseEdit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := t.s
	if e.Title != nil {
		next.Title = validTitle(*e.Title, &errs)
	}
	if e.Preconditions != nil {
		next.Preconditions = strings.TrimSpace(*e.Preconditions)
	}
	if e.Steps != nil {
		next.Steps = validRequired("steps", *e.Steps, "Escribe los pasos.", &errs)
	}
	if e.ExpectedResult != nil {
		next.ExpectedResult = validRequired("expectedResult", *e.ExpectedResult, "Escribe el resultado esperado.", &errs)
	}
	if e.IsCritical != nil {
		next.IsCritical = *e.IsCritical
	}
	if e.FunctionalCases != nil {
		next.FunctionalCases = validFunctionalCases(*e.FunctionalCases, next.SystemKey, &errs)
	}
	if e.Retired != nil {
		switch {
		case *e.Retired && next.RetiredAt == nil:
			next.RetiredAt = &now
		case !*e.Retired:
			next.RetiredAt = nil
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
	add("title", t.s.Title, next.Title)
	add("preconditions", t.s.Preconditions, next.Preconditions)
	add("steps", t.s.Steps, next.Steps)
	add("expectedResult", t.s.ExpectedResult, next.ExpectedResult)
	add("isCritical", boolText(t.s.IsCritical), boolText(next.IsCritical))
	add("functionalCases", codes(t.s.FunctionalCases), codes(next.FunctionalCases))
	add("retired", boolText(t.s.RetiredAt != nil), boolText(next.RetiredAt != nil))
	if len(changes) > 0 {
		next.UpdatedAt = now
		t.s = next
	}
	return changes, nil
}

func validTitle(raw string, errs *axis.FieldErrors) string {
	title := strings.TrimSpace(raw)
	switch {
	case title == "":
		errs.Add(axis.Required("title", "Escribe un título."))
	case utf8.RuneCountInString(title) > maxTitleLength:
		errs.Add(axis.InvalidField("title", "El título no puede pasar de 200 caracteres."))
	}
	return title
}

func validRequired(field, raw, message string, errs *axis.FieldErrors) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		errs.Add(axis.Required(field, message))
	}
	return value
}

// validFunctionalCases exige al menos un UC, todos del sistema del TC; quita repetidos y ordena.
func validFunctionalCases(refs []FunctionalCaseRef, systemKey string, errs *axis.FieldErrors) []FunctionalCaseRef {
	out := make([]FunctionalCaseRef, 0, len(refs))
	for _, ref := range refs {
		if ref.SystemKey != systemKey {
			errs.Add(axis.InvalidField("functionalCases", ref.Code+" es de otro sistema."))
			return nil
		}
		if !slices.ContainsFunc(out, func(o FunctionalCaseRef) bool { return o.Id == ref.Id }) {
			out = append(out, ref)
		}
	}
	if len(out) == 0 {
		errs.Add(axis.Required("functionalCases", "Elige al menos un UC que el TC valide."))
	}
	slices.SortFunc(out, func(a, b FunctionalCaseRef) int { return strings.Compare(a.Code, b.Code) })
	return out
}

func codes(refs []FunctionalCaseRef) string {
	parts := make([]string, len(refs))
	for i, r := range refs {
		parts[i] = r.Code
	}
	return strings.Join(parts, ",")
}

func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
