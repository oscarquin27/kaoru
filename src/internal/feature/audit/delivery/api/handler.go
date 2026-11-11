// Package api expone la auditoría por HTTP.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/audit/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de audit.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/audit", h.list)
	r.GET("/api/audit/verification", h.verify)
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type entryDTO struct {
	Seq        int64      `json:"seq"`
	OccurredAt time.Time  `json:"occurredAt"`
	Actor      userRefDTO `json:"actor"`
	EntityType string     `json:"entityType"`
	EntityCode string     `json:"entityCode,omitempty"`
	Action     string     `json:"action"`
	Field      string     `json:"field,omitempty"`
	OldValue   string     `json:"oldValue,omitempty"`
	NewValue   string     `json:"newValue,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	PrevHash   string     `json:"prevHash,omitempty"`
	Hash       string     `json:"hash"`
}

type pageDTO struct {
	Items []entryDTO `json:"items"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
	Total int        `json:"total"`
}

type verificationDTO struct {
	Intact         bool   `json:"intact"`
	Entries        int    `json:"entries"`
	FirstBrokenSeq *int64 `json:"firstBrokenSeq,omitempty"`
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func (h *Handler) list(c *gin.Context) {
	in := usecase.ListEntriesInput{Actor: actorOf(c)}
	var err error
	if in.Filter.Page, in.Filter.Size, err = httpserver.PageParams(c); err != nil {
		problem.Write(c, err)
		return
	}
	in.Filter.EntityCode = c.Query("entityCode")
	in.Filter.ActorId = c.Query("actor")
	if in.Filter.From, err = timeParam(c, "from"); err != nil {
		problem.Write(c, err)
		return
	}
	if in.Filter.To, err = timeParam(c, "to"); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListEntries.Execute(c.Request.Context(), in)
	if err != nil {
		problem.Write(c, err)
		return
	}
	page := pageDTO{Items: make([]entryDTO, len(out.Entries)), Page: in.Filter.Page, Size: in.Filter.Size, Total: out.Total}
	for i, v := range out.Entries {
		e := v.Entry
		page.Items[i] = entryDTO{
			Seq: e.Seq, OccurredAt: e.OccurredAt,
			Actor:      userRefDTO{ID: string(e.ActorId), DisplayName: v.ActorName},
			EntityType: e.EntityType, EntityCode: e.EntityCode, Action: string(e.Action),
			Field: e.Field, OldValue: e.OldValue, NewValue: e.NewValue, Reason: e.Reason,
			PrevHash: e.PrevHash, Hash: e.Hash,
		}
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) verify(c *gin.Context) {
	out, err := h.uc.VerifyChain.Execute(c.Request.Context(), usecase.VerifyChainInput{Actor: actorOf(c)})
	if err != nil {
		problem.Write(c, err)
		return
	}
	dto := verificationDTO{Intact: out.Intact, Entries: out.Entries}
	if !out.Intact {
		dto.FirstBrokenSeq = &out.FirstBrokenSeq
	}
	c.JSON(http.StatusOK, dto)
}

func timeParam(c *gin.Context, name string) (*time.Time, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, axis.Invalid(axis.InvalidField(name, "La fecha debe tener el formato 2026-11-11T00:00:00Z."))
	}
	return &t, nil
}
