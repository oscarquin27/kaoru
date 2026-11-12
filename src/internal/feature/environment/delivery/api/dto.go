package api

import (
	"time"

	"kaoru/internal/feature/environment/application/usecase"
	"kaoru/internal/feature/environment/domain"
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

type buildRefDTO struct {
	Code     string `json:"code"`
	Version  string `json:"version"`
	Checksum string `json:"checksum"`
}

type codeTitleDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type installedDTO struct {
	Deployment string        `json:"deployment"`
	Build      *buildRefDTO  `json:"build,omitempty"`
	Package    *codeTitleDTO `json:"package,omitempty"`
	DeployedAt time.Time     `json:"deployedAt"`
}

type environmentDTO struct {
	Code      string        `json:"code"`
	System    systemDTO     `json:"system"`
	Name      string        `json:"name"`
	Kind      string        `json:"kind"`
	OS        string        `json:"os"`
	Notes     string        `json:"notes"`
	Retired   bool          `json:"retired"`
	Installed *installedDTO `json:"installed,omitempty"`
	CreatedBy userRefDTO    `json:"createdBy"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

type environmentCreateDTO struct {
	System string `json:"system"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	OS     string `json:"os"`
	Notes  string `json:"notes"`
}

type environmentPatchDTO struct {
	Name    *string `json:"name"`
	Kind    *string `json:"kind"`
	OS      *string `json:"os"`
	Notes   *string `json:"notes"`
	Retired *bool   `json:"retired"`
}

type environmentRefDTO struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type deploymentDTO struct {
	Code        string            `json:"code"`
	Environment environmentRefDTO `json:"environment"`
	Build       *buildRefDTO      `json:"build,omitempty"`
	Package     *codeTitleDTO     `json:"package,omitempty"`
	Result      string            `json:"result"`
	Notes       string            `json:"notes"`
	ExecutedBy  userRefDTO        `json:"executedBy"`
	ExecutedAt  time.Time         `json:"executedAt"`
	CreatedAt   time.Time         `json:"createdAt"`
}

type deploymentCreateDTO struct {
	Environment string    `json:"environment"`
	Build       string    `json:"build"`
	Package     string    `json:"package"`
	Result      string    `json:"result"`
	Notes       string    `json:"notes"`
	ExecutedAt  time.Time `json:"executedAt"`
}

func toBuildRefDTO(b *domain.BuildRef) *buildRefDTO {
	if b == nil {
		return nil
	}
	return &buildRefDTO{Code: b.Code, Version: b.Version, Checksum: b.Checksum}
}

func toPackageDTO(p *domain.PackageRef) *codeTitleDTO {
	if p == nil {
		return nil
	}
	return &codeTitleDTO{Code: p.Code, Title: p.Title}
}

func toEnvironmentDTO(v usecase.EnvironmentOutput) environmentDTO {
	s := v.Environment
	dto := environmentDTO{
		Code:   s.Code.String(),
		System: systemDTO{Key: s.System.Key, Name: s.System.Name, DefaultRoute: s.System.DefaultRoute},
		Name:   s.Name, Kind: string(s.Kind), OS: string(s.OS), Notes: s.Notes, Retired: s.RetiredAt != nil,
		CreatedBy: userRefDTO{ID: string(s.CreatedBy.Id), DisplayName: s.CreatedBy.Name},
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if i := v.Installed; i != nil {
		dto.Installed = &installedDTO{
			Deployment: i.Deployment, Build: toBuildRefDTO(i.Build), Package: toPackageDTO(i.Package), DeployedAt: i.DeployedAt,
		}
	}
	return dto
}

func toDeploymentDTO(s domain.DeploymentSnapshot) deploymentDTO {
	return deploymentDTO{
		Code:        s.Code.String(),
		Environment: environmentRefDTO{Code: s.Environment.Code, Name: s.Environment.Name, Kind: string(s.Environment.Kind)},
		Build:       toBuildRefDTO(s.Build), Package: toPackageDTO(s.Package),
		Result: string(s.Result), Notes: s.Notes,
		ExecutedBy: userRefDTO{ID: string(s.ExecutedBy.Id), DisplayName: s.ExecutedBy.Name},
		ExecutedAt: s.ExecutedAt, CreatedAt: s.CreatedAt,
	}
}
