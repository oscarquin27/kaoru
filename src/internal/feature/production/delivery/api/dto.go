package api

import (
	"time"

	"kaoru/internal/feature/production/application/usecase"
	"kaoru/internal/feature/production/domain"
)

type systemDTO struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	DefaultRoute string `json:"defaultRoute"`
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type codeTitleDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type environmentRefDTO struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type templateStepDTO struct {
	Position    int    `json:"position"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type templateDTO struct {
	System    systemDTO         `json:"system"`
	Version   int               `json:"version"`
	Steps     []templateStepDTO `json:"steps"`
	CreatedBy userRefDTO        `json:"createdBy"`
	CreatedAt time.Time         `json:"createdAt"`
}

type stepInputDTO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type templateCreateDTO struct {
	System string         `json:"system"`
	Steps  []stepInputDTO `json:"steps"`
}

type signatureDTO struct {
	SignedBy userRefDTO `json:"signedBy"`
	SignedAt time.Time  `json:"signedAt"`
	Comment  string     `json:"comment"`
}

type checklistStepDTO struct {
	Position    int           `json:"position"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Signature   *signatureDTO `json:"signature,omitempty"`
}

type checklistDTO struct {
	Code            string             `json:"code"`
	Package         codeTitleDTO       `json:"package"`
	Environment     environmentRefDTO  `json:"environment"`
	TemplateVersion int                `json:"templateVersion"`
	Steps           []checklistStepDTO `json:"steps"`
	SignedSteps     int                `json:"signedSteps"`
	Complete        bool               `json:"complete"`
	StartedBy       userRefDTO         `json:"startedBy"`
	CreatedAt       time.Time          `json:"createdAt"`
}

type checklistCreateDTO struct {
	Package     string `json:"package"`
	Environment string `json:"environment"`
}

type signDTO struct {
	Comment string `json:"comment"`
}

type deploymentRefDTO struct {
	Code        string            `json:"code"`
	Environment environmentRefDTO `json:"environment"`
	ExecutedAt  time.Time         `json:"executedAt"`
}

type reviewDTO struct {
	Result     string     `json:"result"`
	Notes      string     `json:"notes"`
	ReviewedBy userRefDTO `json:"reviewedBy"`
	ReviewedAt time.Time  `json:"reviewedAt"`
}

type diagnosticDTO struct {
	Code        string           `json:"code"`
	Package     codeTitleDTO     `json:"package"`
	Deployment  deploymentRefDTO `json:"deployment"`
	Notes       string           `json:"notes"`
	SubmittedBy userRefDTO       `json:"submittedBy"`
	CreatedAt   time.Time        `json:"createdAt"`
	Review      *reviewDTO       `json:"review,omitempty"`
}

type diagnosticCreateDTO struct {
	Deployment string `json:"deployment"`
	Notes      string `json:"notes"`
}

type reviewCreateDTO struct {
	Result string `json:"result"`
	Notes  string `json:"notes"`
}

type smokeTestDTO struct {
	Code       string           `json:"code"`
	Package    codeTitleDTO     `json:"package"`
	Deployment deploymentRefDTO `json:"deployment"`
	Result     string           `json:"result"`
	Notes      string           `json:"notes"`
	ExecutedBy userRefDTO       `json:"executedBy"`
	CreatedAt  time.Time        `json:"createdAt"`
}

type smokeTestCreateDTO struct {
	Deployment string `json:"deployment"`
	Result     string `json:"result"`
	Notes      string `json:"notes"`
}

func toUserRefDTO(u domain.UserRef) userRefDTO {
	return userRefDTO{ID: string(u.Id), DisplayName: u.Name}
}

func toEnvironmentRefDTO(e domain.EnvironmentRef) environmentRefDTO {
	return environmentRefDTO{Code: e.Code, Name: e.Name, Kind: e.Kind}
}

func toDeploymentRefDTO(d domain.DeploymentRef) deploymentRefDTO {
	return deploymentRefDTO{Code: d.Code, Environment: toEnvironmentRefDTO(d.Environment), ExecutedAt: d.ExecutedAt}
}

func toTemplateDTO(s domain.TemplateSnapshot) templateDTO {
	steps := make([]templateStepDTO, len(s.Steps))
	for i, step := range s.Steps {
		steps[i] = templateStepDTO{Position: step.Position, Title: step.Title, Description: step.Description}
	}
	return templateDTO{
		System:  systemDTO{Key: s.System.Key, Name: s.System.Name, DefaultRoute: s.System.DefaultRoute},
		Version: s.Version, Steps: steps, CreatedBy: toUserRefDTO(s.CreatedBy), CreatedAt: s.CreatedAt,
	}
}

func toChecklistDTO(out usecase.ChecklistOutput) checklistDTO {
	s := out.Checklist
	steps := make([]checklistStepDTO, len(s.Steps))
	for i, step := range s.Steps {
		steps[i] = checklistStepDTO{Position: step.Position, Title: step.Title, Description: step.Description}
		if g := step.Signature; g != nil {
			steps[i].Signature = &signatureDTO{SignedBy: toUserRefDTO(g.SignedBy), SignedAt: g.SignedAt, Comment: g.Comment}
		}
	}
	return checklistDTO{
		Code: s.Code.String(), Package: codeTitleDTO{Code: s.Package.Code, Title: s.Package.Title},
		Environment: toEnvironmentRefDTO(s.Environment), TemplateVersion: s.TemplateVersion, Steps: steps,
		SignedSteps: out.SignedSteps, Complete: out.Complete,
		StartedBy: toUserRefDTO(s.StartedBy), CreatedAt: s.CreatedAt,
	}
}

func toDiagnosticDTO(s domain.DiagnosticSnapshot) diagnosticDTO {
	dto := diagnosticDTO{
		Code: s.Code.String(), Package: codeTitleDTO{Code: s.Package.Code, Title: s.Package.Title},
		Deployment: toDeploymentRefDTO(s.Deployment), Notes: s.Notes,
		SubmittedBy: toUserRefDTO(s.SubmittedBy), CreatedAt: s.CreatedAt,
	}
	if r := s.Review; r != nil {
		dto.Review = &reviewDTO{Result: string(r.Result), Notes: r.Notes, ReviewedBy: toUserRefDTO(r.ReviewedBy), ReviewedAt: r.ReviewedAt}
	}
	return dto
}

func toSmokeTestDTO(s domain.SmokeTestSnapshot) smokeTestDTO {
	return smokeTestDTO{
		Code: s.Code.String(), Package: codeTitleDTO{Code: s.Package.Code, Title: s.Package.Title},
		Deployment: toDeploymentRefDTO(s.Deployment), Result: string(s.Result), Notes: s.Notes,
		ExecutedBy: toUserRefDTO(s.ExecutedBy), CreatedAt: s.CreatedAt,
	}
}
