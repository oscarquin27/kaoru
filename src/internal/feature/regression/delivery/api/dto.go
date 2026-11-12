package api

import (
	"time"

	"kaoru/internal/feature/regression/application/usecase"
)

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type codeTitleDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type componentRefDTO struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type runSummaryDTO struct {
	Code            string    `json:"code"`
	Result          string    `json:"result"`
	BuildVersion    string    `json:"buildVersion"`
	EnvironmentName string    `json:"environmentName"`
	Build           string    `json:"build,omitempty"`
	Environment     string    `json:"environment,omitempty"`
	ExecutedAt      time.Time `json:"executedAt"`
}

type summaryDTO struct {
	Proposed int `json:"proposed"`
	Included int `json:"included"`
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Pending  int `json:"pending"`
}

type itemDTO struct {
	TestCase   codeTitleDTO      `json:"testCase"`
	IsCritical bool              `json:"isCritical"`
	Origin     string            `json:"origin"`
	Proposed   bool              `json:"proposed"`
	Included   bool              `json:"included"`
	Reason     string            `json:"reason"`
	DecidedBy  *userRefDTO       `json:"decidedBy,omitempty"`
	DecidedAt  *time.Time        `json:"decidedAt,omitempty"`
	Components []componentRefDTO `json:"components"`
	LastRun    *runSummaryDTO    `json:"lastRun,omitempty"`
}

type regressionDTO struct {
	Package     string      `json:"package"`
	ProposedBy  userRefDTO  `json:"proposedBy"`
	ProposedAt  time.Time   `json:"proposedAt"`
	ConfirmedBy *userRefDTO `json:"confirmedBy,omitempty"`
	ConfirmedAt *time.Time  `json:"confirmedAt,omitempty"`
	Summary     summaryDTO  `json:"summary"`
	Items       []itemDTO   `json:"items"`
}

// adjustDTO es el cuerpo de un ajuste; sin `included` no vale.
type adjustDTO struct {
	Included *bool  `json:"included"`
	Reason   string `json:"reason"`
}

func toUserRef(u *usecase.UserRef) *userRefDTO {
	if u == nil {
		return nil
	}
	return &userRefDTO{ID: string(u.Id), DisplayName: u.Name}
}

func toDTO(out usecase.RegressionOutput) regressionDTO {
	dto := regressionDTO{
		Package: out.PackageCode, ProposedBy: *toUserRef(&out.ProposedBy), ProposedAt: out.ProposedAt,
		ConfirmedBy: toUserRef(out.ConfirmedBy), ConfirmedAt: out.ConfirmedAt,
		Summary: summaryDTO{
			Proposed: out.Summary.Proposed, Included: out.Summary.Included, Added: out.Summary.Added, Removed: out.Summary.Removed,
			Passed: out.Summary.Passed, Failed: out.Summary.Failed, Pending: out.Summary.Pending,
		},
		Items: make([]itemDTO, len(out.Items)),
	}
	for i, item := range out.Items {
		row := itemDTO{
			TestCase: codeTitleDTO{Code: item.TestCase.Code, Title: item.Title}, IsCritical: item.IsCritical,
			Origin: string(item.Origin), Proposed: item.Proposed, Included: item.Included, Reason: item.Reason,
			DecidedBy: toUserRef(item.DecidedBy), DecidedAt: item.DecidedAt,
			Components: make([]componentRefDTO, len(item.Components)),
		}
		for j, c := range item.Components {
			row.Components[j] = componentRefDTO{Code: c.Code, Name: c.Name, Kind: c.Kind}
		}
		if run := item.LastRun; run != nil {
			row.LastRun = &runSummaryDTO{
				Code: run.Code, Result: run.Result, BuildVersion: run.BuildVersion, EnvironmentName: run.EnvironmentName,
				Build: run.BuildCode, Environment: run.EnvironmentCode, ExecutedAt: run.ExecutedAt,
			}
		}
		dto.Items[i] = row
	}
	return dto
}
