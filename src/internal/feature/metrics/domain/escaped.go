package domain

import (
	"math"
	"time"
)

// DefectRelease es un DEF con la primera vez que su REQ llegó a producción: el primer despliegue
// exitoso, en un ambiente de producción, de un PKG que contiene el REQ. Sin ese despliegue,
// ReleasedAt es nil.
type DefectRelease struct {
	Defect          CodeTitle
	Severity        string
	State           StateRef
	RequirementCode string
	CreatedAt       time.Time
	PackageCode     string
	ReleasedAt      *time.Time
}

// Escaped dice si el DEF se abrió después de que su REQ llegó a producción.
func (d DefectRelease) Escaped() bool {
	return d.ReleasedAt != nil && d.CreatedAt.After(*d.ReleasedAt)
}

// DaysAfterRelease es cuánto después de llegar a producción se abrió; 0 si no escapó.
func (d DefectRelease) DaysAfterRelease() float64 {
	if !d.Escaped() {
		return 0
	}
	return Days(d.CreatedAt.Sub(*d.ReleasedAt))
}

// MonthCount son los DEF creados en un mes y cuántos de ellos escaparon.
type MonthCount struct {
	Month   string
	Defects int
	Escaped int
}

// EscapedMetrics es la cuenta de DEF escapados de un período.
type EscapedMetrics struct {
	Defects int
	Escaped []DefectRelease
	// RatePercent es Escaped / Defects en %, con un decimal; 0 sin DEF.
	RatePercent float64
	Months      []MonthCount
}

// MeasureEscaped cuenta los escapados de defects, que vienen del más reciente al más antiguo, y
// los reparte en los meses del período. Escaped conserva ese orden.
func MeasureEscaped(defects []DefectRelease, period Period, now time.Time) EscapedMetrics {
	out := EscapedMetrics{Defects: len(defects)}
	months := period.Months(now)
	index := make(map[string]int, len(months))
	out.Months = make([]MonthCount, len(months))
	for i, m := range months {
		index[m] = i
		out.Months[i] = MonthCount{Month: m}
	}
	for _, d := range defects {
		i, inPeriod := index[d.CreatedAt.UTC().Format("2006-01")]
		if inPeriod {
			out.Months[i].Defects++
		}
		if !d.Escaped() {
			continue
		}
		out.Escaped = append(out.Escaped, d)
		if inPeriod {
			out.Months[i].Escaped++
		}
	}
	if out.Defects > 0 {
		out.RatePercent = math.Round(float64(len(out.Escaped))*1000/float64(out.Defects)) / 10
	}
	return out
}
