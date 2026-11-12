// Package domain modela la selección de regresión de un PKG: lo que el sistema propuso
// por impacto y lo que QA dejó al final, con el criterio de cada ajuste (regla de negocio 4).
package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Origin dice por qué un TC está en la selección.
type Origin string

// Orígenes de un TC de la selección.
const (
	// OriginImpact: el TC ejercita un componente que toca algún REQ del PKG.
	OriginImpact Origin = "impact"
	// OriginCritical: el TC es crítico del sistema.
	OriginCritical Origin = "critical"
	// OriginManual: lo agregó QA; el sistema no lo propone.
	OriginManual Origin = "manual"
)

// Errores de dominio.
var (
	ErrNotProposed      = axis.New(axis.NotFoundResource, "La regresión de este PKG todavía no se propuso.")
	ErrConfirmed        = axis.New(axis.InvalidAction, "La selección de regresión ya está confirmada y no cambia. Para corregirla, devuelve el PKG a armado.")
	ErrAlreadyConfirmed = axis.New(axis.InvalidAction, "La selección de regresión ya estaba confirmada.")
	ErrTestCaseRetired  = axis.New(axis.InvalidAction, "El TC está retirado: no entra en una regresión.")
	ErrOtherSystem      = axis.New(axis.InvalidAction, "El TC es de otro sistema que el PKG.")
)

// TestCaseRef identifica un TC de la selección.
type TestCaseRef struct {
	Id   string
	Code string
}

// TestCase es lo que la selección necesita saber de un TC para admitirlo.
type TestCase struct {
	Ref       TestCaseRef
	SystemKey string
	Retired   bool
}

// Candidate es un TC que el sistema propone, con su origen: impacto o crítico.
type Candidate struct {
	TestCase TestCaseRef
	Origin   Origin
}

// Item es un TC de la selección. Proposed es lo que propuso el sistema e Included lo que
// queda en la selección final; cuando difieren, Reason dice por qué y Decided*, quién y cuándo.
type Item struct {
	TestCase  TestCaseRef
	Origin    Origin
	Proposed  bool
	Included  bool
	Reason    string
	DecidedBy shared.UserId
	// DecidedAt es nil si QA no ajustó el TC.
	DecidedAt *time.Time
}

// adjusted dice si QA dejó una decisión sobre el TC.
func (i Item) adjusted() bool { return i.DecidedBy != "" }

// Selection es la cabecera de la selección de regresión de un PKG.
type Selection struct {
	PackageId   string
	ProposedBy  shared.UserId
	ProposedAt  time.Time
	ConfirmedBy shared.UserId
	// ConfirmedAt es nil mientras la selección no esté confirmada.
	ConfirmedAt *time.Time
}

// Confirmed dice si la selección ya no cambia.
func (s Selection) Confirmed() bool { return s.ConfirmedAt != nil }

// NewSelection abre la selección de un PKG con su primera propuesta.
func NewSelection(packageId string, by shared.UserId, now time.Time) Selection {
	return Selection{PackageId: packageId, ProposedBy: by, ProposedAt: now}
}

// Propose registra una propuesta nueva. Confirmada, la selección no se vuelve a proponer.
func (s *Selection) Propose(by shared.UserId, now time.Time) error {
	if s.Confirmed() {
		return ErrConfirmed
	}
	s.ProposedBy, s.ProposedAt = by, now
	return nil
}

// Confirm fija la selección: desde entonces no cambia.
func (s *Selection) Confirm(by shared.UserId, now time.Time) error {
	if s.Confirmed() {
		return ErrAlreadyConfirmed
	}
	s.ConfirmedBy, s.ConfirmedAt = by, &now
	return nil
}

// Reopen quita la confirmación. Devuelve false si la selección no estaba confirmada.
func (s *Selection) Reopen() bool {
	if !s.Confirmed() {
		return false
	}
	s.ConfirmedBy, s.ConfirmedAt = "", nil
	return true
}

// Propose combina lo que ya había en la selección con lo que el sistema propone ahora.
// Devuelve los ítems que quedan, por código de TC.
//
//   - Un TC propuesto que ya estaba conserva el ajuste de QA.
//   - Un TC propuesto nuevo entra incluido.
//   - Un TC que QA había agregado a mano y ahora el sistema propone deja de ser un ajuste:
//     la decisión de QA coincide con la propuesta y no necesita criterio.
//   - Un TC que ya no se propone y QA no tocó sale de la selección.
//   - Un TC que ya no se propone pero QA había ajustado pasa a manual y conserva la decisión;
//     si esa decisión era quitarlo, sale: ni el sistema lo propone ni QA lo quiere.
//   - Un TC agregado a mano que sigue sin proponerse se conserva.
func Propose(current []Item, candidates []Candidate) []Item {
	previous := make(map[string]Item, len(current))
	for _, item := range current {
		previous[item.TestCase.Id] = item
	}
	proposed := make(map[string]bool, len(candidates))
	next := make([]Item, 0, len(candidates))
	for _, c := range candidates {
		if proposed[c.TestCase.Id] {
			continue
		}
		proposed[c.TestCase.Id] = true
		item, existed := previous[c.TestCase.Id]
		switch {
		case !existed:
			item = Item{Included: true}
		case !item.Proposed:
			// Era un agregado de QA: ahora coincide con la propuesta.
			item.Reason, item.DecidedBy, item.DecidedAt = "", "", nil
		}
		item.TestCase, item.Origin, item.Proposed = c.TestCase, c.Origin, true
		next = append(next, item)
	}
	for _, item := range current {
		if proposed[item.TestCase.Id] {
			continue
		}
		if item.Proposed {
			if !item.adjusted() {
				continue
			}
			item.Origin, item.Proposed = OriginManual, false
		}
		// Sin propuesta, solo queda lo que QA incluyó con su criterio.
		if !item.Included || item.Reason == "" {
			continue
		}
		next = append(next, item)
	}
	slices.SortFunc(next, func(a, b Item) int { return cmp.Compare(a.TestCase.Code, b.TestCase.Code) })
	return next
}

// Adjustment es el resultado de un ajuste de QA sobre un TC.
type Adjustment struct {
	// Item es el TC como queda; no vale si Removed.
	Item Item
	// Changed es false si el ajuste deja todo como estaba: no se guarda ni se audita.
	Changed bool
	// Removed dice que el TC sale de la selección: era un agregado de QA y QA lo quitó.
	Removed bool
}

// Decision es lo que QA decide sobre un TC.
type Decision struct {
	Included bool
	Reason   string
	By       shared.UserId
	At       time.Time
}

// Adjust incluye o excluye un TC de la selección de un PKG de ese sistema. current es nil si el TC
// no está en la selección: entra como agregado de QA. Solo se incluye un TC activo del sistema del
// PKG. El criterio es obligatorio cuando la decisión difiere de lo propuesto; cuando coincide, no
// queda criterio ni autor.
func Adjust(current *Item, testCase TestCase, systemKey string, d Decision) (Adjustment, error) {
	item := Item{TestCase: testCase.Ref, Origin: OriginManual}
	if current != nil {
		item = *current
	}
	if d.Included {
		switch {
		case testCase.SystemKey != systemKey:
			return Adjustment{}, ErrOtherSystem
		case testCase.Retired:
			return Adjustment{}, ErrTestCaseRetired
		}
	}
	included, reason, by, now := d.Included, strings.TrimSpace(d.Reason), d.By, d.At
	if included == item.Proposed {
		if !item.Proposed {
			// Un agregado de QA que QA quita no deja rastro en la selección (sí en la auditoría).
			return Adjustment{Changed: current != nil, Removed: current != nil}, nil
		}
		changed := !item.Included || item.Reason != "" || item.adjusted()
		item.Included, item.Reason, item.DecidedBy, item.DecidedAt = included, "", "", nil
		return Adjustment{Item: item, Changed: changed}, nil
	}
	if reason == "" {
		message := "Escribe por qué se agrega un TC que el sistema no propuso."
		if item.Proposed {
			message = "Escribe por qué se quita un TC que el sistema propuso."
		}
		return Adjustment{}, axis.Invalid(axis.Required("reason", message))
	}
	if current != nil && item.Included == included && item.Reason == reason {
		return Adjustment{Item: item}, nil
	}
	item.Included, item.Reason, item.DecidedBy, item.DecidedAt = included, reason, by, &now
	return Adjustment{Item: item, Changed: true}, nil
}

// runPassed es el resultado de un RUN aprobado, como lo guarda execution.
const runPassed = "passed"

// Summary resume la selección: lo propuesto contra lo final y el avance de la ejecución.
type Summary struct {
	Proposed int
	Included int
	Added    int
	Removed  int
	Passed   int
	Failed   int
	Pending  int
}

// Progress es un TC de la selección con el resultado de su último RUN sobre un BUILD del PKG;
// LastResult vacío significa que no tiene ninguno.
type Progress struct {
	Item       Item
	LastResult string
}

// Summarize cuenta lo propuesto, lo final y, sobre lo final, lo aprobado, lo fallido y lo pendiente.
func Summarize(items []Progress) Summary {
	var s Summary
	for _, p := range items {
		if p.Item.Proposed {
			s.Proposed++
		}
		switch {
		case p.Item.Included && !p.Item.Proposed:
			s.Added++
		case !p.Item.Included && p.Item.Proposed:
			s.Removed++
		}
		if !p.Item.Included {
			continue
		}
		s.Included++
		switch p.LastResult {
		case "":
			s.Pending++
		case runPassed:
			s.Passed++
		default:
			s.Failed++
		}
	}
	return s
}

// Unpassed devuelve los códigos de los TC de la selección final cuyo último RUN sobre un BUILD
// del PKG no está aprobado: los pendientes y los fallidos.
func Unpassed(items []Progress) []string {
	var codes []string
	for _, p := range items {
		if p.Item.Included && p.LastResult != runPassed {
			codes = append(codes, p.Item.TestCase.Code)
		}
	}
	return codes
}
