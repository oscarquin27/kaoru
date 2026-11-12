package api

import (
	"time"

	"kaoru/internal/feature/build/domain"
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

type buildDTO struct {
	Code         string         `json:"code"`
	System       systemDTO      `json:"system"`
	Version      string         `json:"version"`
	Checksum     string         `json:"checksum"`
	Requirements []codeTitleDTO `json:"requirements"`
	Migrations   string         `json:"migrations"`
	Dependencies string         `json:"dependencies"`
	Notes        string         `json:"notes"`
	BuiltBy      userRefDTO     `json:"builtBy"`
	CreatedAt    time.Time      `json:"createdAt"`
}

type pageDTO struct {
	Items []buildDTO `json:"items"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
	Total int        `json:"total"`
}

type registerDTO struct {
	System       string   `json:"system"`
	Version      string   `json:"version"`
	Checksum     string   `json:"checksum"`
	Requirements []string `json:"requirements"`
	Migrations   string   `json:"migrations"`
	Dependencies string   `json:"dependencies"`
	Notes        string   `json:"notes"`
}

func toDTO(s domain.Snapshot) buildDTO {
	requirements := make([]codeTitleDTO, len(s.Requirements))
	for i, r := range s.Requirements {
		requirements[i] = codeTitleDTO{Code: r.Code, Title: r.Title}
	}
	return buildDTO{
		Code:    s.Code.String(),
		System:  systemDTO{Key: s.System.Key, Name: s.System.Name, DefaultRoute: s.System.DefaultRoute},
		Version: s.Version, Checksum: s.Checksum, Requirements: requirements,
		Migrations: s.Migrations, Dependencies: s.Dependencies, Notes: s.Notes,
		BuiltBy:   userRefDTO{ID: string(s.BuiltBy.Id), DisplayName: s.BuiltBy.Name},
		CreatedAt: s.CreatedAt,
	}
}
