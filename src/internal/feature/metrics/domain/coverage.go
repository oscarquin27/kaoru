package domain

import (
	"cmp"
	"slices"
)

// CoverageStatus resume la cobertura de un REQ frente a lo que exige su riesgo.
type CoverageStatus string

// Estados de cobertura, en el orden en que se listan: primero lo que hay que atender.
const (
	CoverageBelow       CoverageStatus = "below"
	CoverageNoTestCases CoverageStatus = "no_test_cases"
	CoverageUnassessed  CoverageStatus = "unassessed"
	CoverageMeets       CoverageStatus = "meets"
)

var coverageOrder = map[CoverageStatus]int{CoverageBelow: 0, CoverageNoTestCases: 1, CoverageUnassessed: 2, CoverageMeets: 3}

// ValidCoverageStatus dice si s es un estado de cobertura conocido.
func ValidCoverageStatus(s string) bool {
	_, ok := coverageOrder[CoverageStatus(s)]
	return ok
}

// CoverageFilter acota la cobertura. Un campo vacío no filtra.
type CoverageFilter struct {
	SystemKey   string
	ChangeCode  string
	PackageCode string
	StateKey    string
}

// CoverageRow es la cobertura de un REQ con lo mismo que cuenta la regla min_coverage: los TC activos
// que validan algún UC del REQ y los que tienen su último RUN aprobado.
type CoverageRow struct {
	Requirement CodeTitle
	SystemKey   string
	State       StateRef
	// RiskLevel vacío: el riesgo no está evaluado.
	RiskLevel string
	Required  int
	Passed    int
	// MinimumPercent es lo que exige la regla para su nivel de riesgo; nil sin riesgo o sin regla activa.
	MinimumPercent *float64
}

// Percent es la cobertura redondeada hacia abajo; 0 sin TC.
func (r CoverageRow) Percent() int {
	if r.Required == 0 {
		return 0
	}
	return r.Passed * 100 / r.Required
}

// Status compara la cobertura con el mínimo con la misma cuenta que la regla: sin riesgo no hay
// mínimo que comparar, y sin TC no hay cobertura.
func (r CoverageRow) Status() CoverageStatus {
	switch {
	case r.RiskLevel == "":
		return CoverageUnassessed
	case r.Required == 0:
		return CoverageNoTestCases
	case r.MinimumPercent != nil && float64(r.Passed)*100 < *r.MinimumPercent*float64(r.Required):
		return CoverageBelow
	default:
		return CoverageMeets
	}
}

// CoverageSummary cuenta los REQ por estado de cobertura y suma sus TC.
type CoverageSummary struct {
	Requirements int
	ByStatus     map[CoverageStatus]int
	Required     int
	Passed       int
}

// SummarizeCoverage resume todas las filas.
func SummarizeCoverage(rows []CoverageRow) CoverageSummary {
	s := CoverageSummary{Requirements: len(rows), ByStatus: map[CoverageStatus]int{}}
	for _, r := range rows {
		s.ByStatus[r.Status()]++
		s.Required += r.Required
		s.Passed += r.Passed
	}
	return s
}

// SortCoverage ordena lo que hay que atender primero: por estado, después menor cobertura, después
// más TC sin aprobar, y por último el código.
func SortCoverage(rows []CoverageRow) {
	slices.SortStableFunc(rows, func(a, b CoverageRow) int {
		return cmp.Or(
			cmp.Compare(coverageOrder[a.Status()], coverageOrder[b.Status()]),
			cmp.Compare(a.Percent(), b.Percent()),
			cmp.Compare(b.Required-b.Passed, a.Required-a.Passed),
			cmp.Compare(a.Requirement.Code, b.Requirement.Code),
		)
	})
}
