// Package usecase contiene los casos de uso de metrics. Todos son lecturas y ninguno exige permiso.
package usecase

import (
	"errors"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/feature/metrics/application/port"
	"kaoru/internal/feature/metrics/domain"
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository port.Repository
	Clock      clock.Clock
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	GetCoverage       *GetCoverage
	GetDefectStates   *GetDefectStates
	GetRegression     *GetRegression
	GetEscapedDefects *GetEscapedDefects
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Clock == nil {
		return nil, errors.New("metrics: faltan dependencias")
	}
	return &UseCases{
		GetCoverage:       &GetCoverage{d: d},
		GetDefectStates:   &GetDefectStates{d: d},
		GetRegression:     &GetRegression{d: d},
		GetEscapedDefects: &GetEscapedDefects{d: d},
	}, nil
}

// PeriodInput es el período pedido: From inclusivo y To exclusivo, en UTC. Sin From, los últimos
// 12 meses; sin To, sin fin.
type PeriodInput struct {
	From *time.Time
	To   *time.Time
}

// resolve completa el período y comprueba que no termine antes de empezar.
func (p PeriodInput) resolve(now time.Time, errs *axis.FieldErrors) domain.Period {
	period := domain.Period{From: domain.DefaultFrom(now), To: p.To}
	if p.From != nil {
		period.From = p.From.UTC()
	}
	if period.To != nil && !period.To.After(period.From) {
		errs.Add(axis.InvalidField("to", "La fecha final no puede ser anterior a la inicial."))
	}
	return period
}

func checkSeverity(severity string, errs *axis.FieldErrors) {
	if severity != "" && severity != "high" && severity != "medium" && severity != "low" {
		errs.Add(axis.InvalidField("severity", "La severidad debe ser high, medium o low."))
	}
}

// page devuelve la página pedida de items; una página después de la última está vacía.
func page[T any](items []T, number, size int) []T {
	start := min((number-1)*size, len(items))
	return items[start:min(start+size, len(items))]
}
