package api

import (
	"math"
	"time"

	"kaoru/internal/feature/metrics/domain"
)

type stateDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type codeTitleDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type periodDTO struct {
	From string  `json:"from"`
	To   *string `json:"to,omitempty"`
}

type pageDTO struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}

type coverageDTO struct {
	pageDTO
	Summary coverageSummaryDTO `json:"summary"`
	Items   []coverageRowDTO   `json:"items"`
}

type coverageSummaryDTO struct {
	Requirements      int `json:"requirements"`
	Below             int `json:"below"`
	NoTestCases       int `json:"noTestCases"`
	Unassessed        int `json:"unassessed"`
	Meets             int `json:"meets"`
	RequiredTestCases int `json:"requiredTestCases"`
	PassedTestCases   int `json:"passedTestCases"`
}

type coverageRowDTO struct {
	Requirement    codeTitleDTO `json:"requirement"`
	System         string       `json:"system"`
	State          stateDTO     `json:"state"`
	RiskLevel      string       `json:"riskLevel,omitempty"`
	Required       int          `json:"required"`
	Passed         int          `json:"passed"`
	Percent        int          `json:"percent"`
	MinimumPercent *float64     `json:"minimumPercent,omitempty"`
	Status         string       `json:"status"`
}

type defectStatesDTO struct {
	Period     periodDTO        `json:"period"`
	Total      int              `json:"total"`
	States     []stateMetricDTO `json:"states"`
	Resolution resolutionDTO    `json:"resolution"`
}

type stateMetricDTO struct {
	State      stateDTO `json:"state"`
	IsFinal    bool     `json:"isFinal"`
	Current    int      `json:"current"`
	Defects    int      `json:"defects"`
	Visits     int      `json:"visits"`
	MedianDays *float64 `json:"medianDays,omitempty"`
	P90Days    *float64 `json:"p90Days,omitempty"`
}

type resolutionDTO struct {
	Defects    int      `json:"defects"`
	MedianDays *float64 `json:"medianDays,omitempty"`
	P90Days    *float64 `json:"p90Days,omitempty"`
}

type regressionDTO struct {
	pageDTO
	Period  periodDTO            `json:"period"`
	Summary regressionSummaryDTO `json:"summary"`
	Items   []regressionRowDTO   `json:"items"`
}

type regressionSummaryDTO struct {
	Packages    int      `json:"packages"`
	Proposed    int      `json:"proposed"`
	Final       int      `json:"final"`
	Added       int      `json:"added"`
	Removed     int      `json:"removed"`
	MedianFinal *float64 `json:"medianFinal,omitempty"`
}

type regressionRowDTO struct {
	Package         codeTitleDTO `json:"package"`
	System          string       `json:"system"`
	State           stateDTO     `json:"state"`
	ConfirmedAt     time.Time    `json:"confirmedAt"`
	Proposed        int          `json:"proposed"`
	Final           int          `json:"final"`
	Added           int          `json:"added"`
	Removed         int          `json:"removed"`
	ByOrigin        originDTO    `json:"byOrigin"`
	Passed          int          `json:"passed"`
	SystemTestCases int          `json:"systemTestCases"`
}

type originDTO struct {
	Impact   int `json:"impact"`
	Critical int `json:"critical"`
	Manual   int `json:"manual"`
}

type escapedDTO struct {
	pageDTO
	Period  periodDTO         `json:"period"`
	Summary escapedSummaryDTO `json:"summary"`
	Months  []monthDTO        `json:"months"`
	Items   []escapedRowDTO   `json:"items"`
}

type escapedSummaryDTO struct {
	Defects     int     `json:"defects"`
	Escaped     int     `json:"escaped"`
	RatePercent float64 `json:"ratePercent"`
}

type monthDTO struct {
	Month   string `json:"month"`
	Defects int    `json:"defects"`
	Escaped int    `json:"escaped"`
}

type escapedRowDTO struct {
	Defect           codeTitleDTO `json:"defect"`
	Severity         string       `json:"severity"`
	State            stateDTO     `json:"state"`
	Requirement      string       `json:"requirement"`
	Package          string       `json:"package"`
	ReleasedAt       time.Time    `json:"releasedAt"`
	CreatedAt        time.Time    `json:"createdAt"`
	DaysAfterRelease float64      `json:"daysAfterRelease"`
}

func toState(s domain.StateRef) stateDTO { return stateDTO{Key: s.Key, Name: s.Name} }

func toCodeTitle(c domain.CodeTitle) codeTitleDTO { return codeTitleDTO{Code: c.Code, Title: c.Title} }

// toPeriod devuelve el período con su fin inclusivo, como se pidió: el día anterior al fin exclusivo.
func toPeriod(p domain.Period) periodDTO {
	out := periodDTO{From: p.From.UTC().Format(time.DateOnly)}
	if p.To != nil {
		to := p.To.UTC().AddDate(0, 0, -1).Format(time.DateOnly)
		out.To = &to
	}
	return out
}

// oneDecimal redondea días a un decimal.
func oneDecimal(f float64) float64 { return math.Round(f*10) / 10 }

// rounded redondea a un decimal; nil sigue nil.
func rounded(f *float64) *float64 {
	if f == nil {
		return nil
	}
	v := oneDecimal(*f)
	return &v
}
