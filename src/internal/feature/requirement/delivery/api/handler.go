// Package api expone los REQ, sus enlaces y su contrato de entrada por HTTP.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/requirement/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de requirement.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/requirements", h.list)
	r.POST("/api/requirements", h.create)
	r.GET("/api/requirements/:code", h.get)
	r.PATCH("/api/requirements/:code", h.update)
	r.POST("/api/requirements/:code/transitions", h.transition)
	r.GET("/api/requirements/:code/links", h.listLinks)
	r.POST("/api/requirements/:code/links", h.addLink)
	r.PATCH("/api/requirements/:code/links/:id", h.updateLink)
	r.DELETE("/api/requirements/:code/links/:id", h.removeLink)
	r.GET("/api/requirements/:code/entry-contract", h.getEntryContract)
	r.PUT("/api/requirements/:code/entry-contract/:itemKey", h.checkEntryItem)
	r.DELETE("/api/requirements/:code/entry-contract/:itemKey", h.uncheckEntryItem)
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func (h *Handler) list(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListRequirements.Execute(c.Request.Context(), usecase.ListRequirementsInput{
		SystemKey: c.Query("system"), StateKey: c.Query("state"), ChangeCode: c.Query("change"),
		OwnerId: c.Query("owner"), Query: c.Query("q"), Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	dto := pageDTO{Items: make([]requirementDTO, len(out.Requirements)), Page: page, Size: size, Total: out.Total}
	for i, s := range out.Requirements {
		dto.Items[i] = toDTO(s)
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) create(c *gin.Context) {
	var body createDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateRequirement.Execute(c.Request.Context(), usecase.CreateRequirementInput{
		Actor: actorOf(c), Title: body.Title, Description: body.Description, AcceptanceCriteria: body.AcceptanceCriteria,
		SystemKey: body.System, ChangeCode: body.Change, OwnerId: body.Owner,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetRequirement.Execute(c.Request.Context(), usecase.GetRequirementInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *Handler) update(c *gin.Context) {
	var body patchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateRequirement.Execute(c.Request.Context(), usecase.UpdateRequirementInput{
		Actor: actorOf(c), Code: c.Param("code"),
		Title: body.Title, Description: body.Description, AcceptanceCriteria: body.AcceptanceCriteria,
		ChangeSet: body.Change.set, ChangeCode: body.Change.value,
		OwnerSet: body.Owner.set, OwnerId: body.Owner.value,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *Handler) transition(c *gin.Context) {
	var body transitionDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.TransitionRequirement.Execute(c.Request.Context(), usecase.TransitionRequirementInput{
		Actor: actorOf(c), Code: c.Param("code"), To: body.To, Reason: body.Reason,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *Handler) listLinks(c *gin.Context) {
	links, err := h.uc.ListLinks.Execute(c.Request.Context(), usecase.ListLinksInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]linkDTO, len(links))
	for i, l := range links {
		items[i] = toLinkDTO(l)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) addLink(c *gin.Context) {
	var body linkCreateDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	link, err := h.uc.AddLink.Execute(c.Request.Context(), usecase.AddLinkInput{
		Actor: actorOf(c), Code: c.Param("code"), URL: body.URL, Kind: body.Kind, Label: body.Label, IsOfficial: body.IsOfficial,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toLinkDTO(link))
}

func (h *Handler) updateLink(c *gin.Context) {
	var body linkPatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	link, err := h.uc.UpdateLink.Execute(c.Request.Context(), usecase.UpdateLinkInput{
		Actor: actorOf(c), Code: c.Param("code"), LinkId: c.Param("id"),
		Label: body.Label, IsOfficial: body.IsOfficial,
		SnapshotSet: body.SnapshotDocument.set, SnapshotCode: body.SnapshotDocument.value,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toLinkDTO(link))
}

func (h *Handler) removeLink(c *gin.Context) {
	err := h.uc.RemoveLink.Execute(c.Request.Context(), usecase.RemoveLinkInput{
		Actor: actorOf(c), Code: c.Param("code"), LinkId: c.Param("id"),
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) getEntryContract(c *gin.Context) {
	items, err := h.uc.GetEntryContract.Execute(c.Request.Context(), usecase.GetEntryContractInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	dto := make([]entryItemDTO, len(items))
	for i, item := range items {
		dto[i] = toEntryItemDTO(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": dto})
}

func (h *Handler) checkEntryItem(c *gin.Context) {
	var body entryCheckDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	item, err := h.uc.CheckEntryItem.Execute(c.Request.Context(), usecase.CheckEntryItemInput{
		Actor: actorOf(c), Code: c.Param("code"), ItemKey: c.Param("itemKey"), Note: body.Note,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toEntryItemDTO(item))
}

func (h *Handler) uncheckEntryItem(c *gin.Context) {
	err := h.uc.UncheckEntryItem.Execute(c.Request.Context(), usecase.UncheckEntryItemInput{
		Actor: actorOf(c), Code: c.Param("code"), ItemKey: c.Param("itemKey"),
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
