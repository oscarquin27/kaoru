package api

import (
	"time"

	"kaoru/internal/feature/report/domain"
)

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type codeTitleDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type reportDTO struct {
	Code         string       `json:"code"`
	Package      codeTitleDTO `json:"package"`
	Version      int          `json:"version"`
	Previous     string       `json:"previous,omitempty"`
	SupersededBy string       `json:"supersededBy,omitempty"`
	Checksum     string       `json:"checksum"`
	IssuedBy     userRefDTO   `json:"issuedBy"`
	CreatedAt    time.Time    `json:"createdAt"`
	Content      string       `json:"content,omitempty"`
}

type issueDTO struct {
	Package string `json:"package"`
}

func toReportDTO(s domain.Snapshot) reportDTO {
	return reportDTO{
		Code: s.Code.String(), Package: codeTitleDTO{Code: s.Package.Code, Title: s.Package.Title},
		Version: s.Version, Previous: s.PreviousCode, SupersededBy: s.SupersededBy, Checksum: s.Checksum,
		IssuedBy: userRefDTO{ID: string(s.IssuedBy.Id), DisplayName: s.IssuedBy.Name}, CreatedAt: s.CreatedAt, Content: s.Content,
	}
}
