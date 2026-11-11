package usecase

import (
	"context"

	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// AddLink agrega un enlace externo a un REQ. El servidor nunca abre la URL.
type AddLink struct {
	d Deps
}

// AddLinkInput es la entrada de AddLink. Kind y Label vacíos se deducen de la URL.
type AddLinkInput struct {
	Actor      shared.Actor
	Code       string
	URL        string
	Kind       string
	Label      string
	IsOfficial bool
}

// Execute exige el permiso link.manage.
func (uc *AddLink) Execute(ctx context.Context, in AddLinkInput) (domain.LinkSnapshot, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionManageLinks); err != nil {
		return domain.LinkSnapshot{}, err
	}
	requirement, err := findRequirement(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return domain.LinkSnapshot{}, err
	}
	s := requirement.Snapshot()
	adder, err := uc.d.Repository.FindUser(ctx, in.Actor.Id)
	if err != nil {
		return domain.LinkSnapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.LinkSnapshot{}, err
	}
	link, err := domain.NewLink(domain.NewLinkParams{
		Id: id, RequirementId: s.Id, URL: in.URL, Kind: in.Kind, Label: in.Label,
		IsOfficial: in.IsOfficial, AddedBy: adder, Now: uc.d.Clock.Now(),
	})
	if err != nil {
		return domain.LinkSnapshot{}, err
	}
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		if err := uc.d.Repository.Lock(ctx, s.Code); err != nil {
			return err
		}
		if err := uc.d.Repository.InsertLink(ctx, link); err != nil {
			return err
		}
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, RequirementId: s.Id, Code: s.Code, Action: port.AuditCreate,
			Field: "link", NewValue: link.Snapshot().URL,
		})
	})
	if err != nil {
		return domain.LinkSnapshot{}, err
	}
	return link.Snapshot(), nil
}
