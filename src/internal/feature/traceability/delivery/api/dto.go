package api

import (
	"time"

	"kaoru/internal/feature/traceability/domain"
)

type systemDTO struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	DefaultRoute string `json:"defaultRoute"`
}

type stateDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type codeTitleDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type runDTO struct {
	Code            string    `json:"code"`
	Result          string    `json:"result"`
	BuildVersion    string    `json:"buildVersion"`
	EnvironmentName string    `json:"environmentName"`
	ExecutedAt      time.Time `json:"executedAt"`
}

type testCaseDTO struct {
	Code       string  `json:"code"`
	Title      string  `json:"title"`
	IsCritical bool    `json:"isCritical"`
	Retired    bool    `json:"retired"`
	LastRun    *runDTO `json:"lastRun,omitempty"`
}

type defectDTO struct {
	Code     string   `json:"code"`
	Title    string   `json:"title,omitempty"`
	Severity string   `json:"severity"`
	State    stateDTO `json:"state"`
}

type riskDTO struct {
	Code          string `json:"code"`
	Level         string `json:"level"`
	Justification string `json:"justification"`
}

type headerDTO struct {
	Code               string        `json:"code"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	AcceptanceCriteria string        `json:"acceptanceCriteria"`
	System             systemRefDTO  `json:"system"`
	Change             *codeTitleDTO `json:"change,omitempty"`
	Route              string        `json:"route"`
	State              stateDTO      `json:"state"`
	Owner              *userRefDTO   `json:"owner,omitempty"`
	Risk               *riskDTO      `json:"risk,omitempty"`
	CreatedAt          time.Time     `json:"createdAt"`
	UpdatedAt          time.Time     `json:"updatedAt"`
}

type systemRefDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type lifecycleDTO struct {
	State     stateDTO   `json:"state"`
	Status    string     `json:"status"`
	EnteredAt *time.Time `json:"enteredAt,omitempty"`
	Days      float64    `json:"days"`
}

type conditionDTO struct {
	RuleKey string   `json:"ruleKey"`
	Message string   `json:"message"`
	Blocks  bool     `json:"blocks"`
	Related []string `json:"related"`
}

type missingDTO struct {
	To         stateDTO       `json:"to"`
	Conditions []conditionDTO `json:"conditions"`
}

type coverageDTO struct {
	Required int `json:"required"`
	Passed   int `json:"passed"`
}

type overviewDTO struct {
	Requirement     headerDTO      `json:"requirement"`
	Lifecycle       []lifecycleDTO `json:"lifecycle"`
	Missing         []missingDTO   `json:"missing"`
	Coverage        coverageDTO    `json:"coverage"`
	FunctionalCases []codeTitleDTO `json:"functionalCases"`
	TestCases       []testCaseDTO  `json:"testCases"`
	Defects         []defectDTO    `json:"defects"`
}

type matrixRowDTO struct {
	Requirement     codeTitleDTO `json:"requirement"`
	System          string       `json:"system"`
	State           stateDTO     `json:"state"`
	Change          string       `json:"change,omitempty"`
	Risk            string       `json:"risk,omitempty"`
	RiskLevel       string       `json:"riskLevel,omitempty"`
	Owner           string       `json:"owner,omitempty"`
	FunctionalCases []string     `json:"functionalCases"`
	TestCase        *testCaseDTO `json:"testCase,omitempty"`
	Defects         []defectDTO  `json:"defects"`
}

type indicatorsDTO struct {
	CoveragePercent     int            `json:"coveragePercent"`
	RequiredTestCases   int            `json:"requiredTestCases"`
	PassedTestCases     int            `json:"passedTestCases"`
	BlockedRequirements int            `json:"blockedRequirements"`
	OpenDefects         map[string]int `json:"openDefects"`
}

type matrixDTO struct {
	Indicators indicatorsDTO  `json:"indicators"`
	Items      []matrixRowDTO `json:"items"`
	Page       int            `json:"page"`
	Size       int            `json:"size"`
	Total      int            `json:"total"`
}

type searchHitDTO struct {
	Type  string    `json:"type"`
	Code  string    `json:"code"`
	Title string    `json:"title"`
	State *stateDTO `json:"state,omitempty"`
}

func toState(s domain.StateRef) stateDTO { return stateDTO{Key: s.Key, Name: s.Name} }

func toTestCase(t domain.TestCaseSummary) testCaseDTO {
	dto := testCaseDTO{Code: t.Code, Title: t.Title, IsCritical: t.IsCritical, Retired: t.Retired}
	if t.LastRun.Code != "" {
		dto.LastRun = &runDTO{
			Code: t.LastRun.Code, Result: t.LastRun.Result, BuildVersion: t.LastRun.BuildVersion,
			EnvironmentName: t.LastRun.EnvironmentName, ExecutedAt: t.LastRun.ExecutedAt,
		}
	}
	return dto
}

func toDefects(defects []domain.DefectSummary) []defectDTO {
	out := make([]defectDTO, len(defects))
	for i, d := range defects {
		out[i] = defectDTO{Code: d.Code, Title: d.Title, Severity: d.Severity, State: toState(d.State)}
	}
	return out
}

func toCodeTitles(items []domain.CodeTitle) []codeTitleDTO {
	out := make([]codeTitleDTO, len(items))
	for i, item := range items {
		out[i] = codeTitleDTO{Code: item.Code, Title: item.Title}
	}
	return out
}

func toOverview(o domain.Overview) overviewDTO {
	h := o.Requirement
	dto := overviewDTO{
		Requirement: headerDTO{
			Code: h.Code, Title: h.Title, Description: h.Description, AcceptanceCriteria: h.AcceptanceCriteria,
			System: systemRefDTO{Key: h.System.Code, Name: h.System.Title}, Route: h.Route, State: toState(h.State),
			CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
		},
		Lifecycle:       make([]lifecycleDTO, len(o.Lifecycle)),
		Missing:         make([]missingDTO, len(o.Missing)),
		Coverage:        coverageDTO{Required: o.Coverage.Required, Passed: o.Coverage.Passed},
		FunctionalCases: toCodeTitles(o.FunctionalCases),
		TestCases:       make([]testCaseDTO, len(o.TestCases)),
		Defects:         toDefects(o.Defects),
	}
	if h.Change.Code != "" {
		dto.Requirement.Change = &codeTitleDTO{Code: h.Change.Code, Title: h.Change.Title}
	}
	if h.Owner.Id != "" {
		dto.Requirement.Owner = &userRefDTO{ID: h.Owner.Id, DisplayName: h.Owner.Name}
	}
	if h.Risk.Code != "" {
		dto.Requirement.Risk = &riskDTO{Code: h.Risk.Code, Level: h.Risk.Level, Justification: h.Risk.Justification}
	}
	for i, step := range o.Lifecycle {
		dto.Lifecycle[i] = lifecycleDTO{State: toState(step.State), Status: step.Status, EnteredAt: step.EnteredAt, Days: round1(step.Days)}
	}
	for i, m := range o.Missing {
		dto.Missing[i] = missingDTO{To: toState(m.To), Conditions: make([]conditionDTO, len(m.Conditions))}
		for j, c := range m.Conditions {
			related := c.Related
			if related == nil {
				related = []string{}
			}
			dto.Missing[i].Conditions[j] = conditionDTO{RuleKey: c.RuleKey, Message: c.Message, Blocks: c.Blocks, Related: related}
		}
	}
	for i, t := range o.TestCases {
		dto.TestCases[i] = toTestCase(t)
	}
	return dto
}

func toMatrixRow(r domain.MatrixRow) matrixRowDTO {
	dto := matrixRowDTO{
		Requirement: codeTitleDTO{Code: r.RequirementCode, Title: r.RequirementTitle}, System: r.SystemKey,
		State: toState(r.State), Change: r.ChangeCode, Risk: r.RiskCode, RiskLevel: r.RiskLevel, Owner: r.OwnerName,
		FunctionalCases: r.FunctionalCases, Defects: toDefects(r.Defects),
	}
	if dto.FunctionalCases == nil {
		dto.FunctionalCases = []string{}
	}
	if r.TestCase.Code != "" {
		testCase := toTestCase(r.TestCase)
		dto.TestCase = &testCase
	}
	return dto
}

// round1 deja un decimal: basta para leer «3,5 días».
func round1(v float64) float64 { return float64(int64(v*10+0.5)) / 10 }
