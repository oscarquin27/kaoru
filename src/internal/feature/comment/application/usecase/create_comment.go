package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/comment/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateComment comenta en una entidad.
type CreateComment struct {
	d Deps
}

// CreateCommentInput es la entrada de CreateComment.
type CreateCommentInput struct {
	Actor      shared.Actor
	EntityType string
	EntityCode string
	Body       string
}

// Execute exige el permiso comment.create.
func (uc *CreateComment) Execute(ctx context.Context, in CreateCommentInput) (domain.Comment, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionCreate); err != nil {
		return domain.Comment{}, err
	}
	entity, err := uc.d.Entities.Resolve(ctx, in.EntityType, in.EntityCode)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Comment{}, axis.Invalid(axis.InvalidField("entityCode", "No existe o no admite comentarios."))
	}
	if err != nil {
		return domain.Comment{}, err
	}
	authorName, err := uc.d.Repository.FindUserName(ctx, in.Actor.Id)
	if err != nil {
		return domain.Comment{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.Comment{}, err
	}
	comment, err := domain.New(id, entity, in.Body, in.Actor.Id, authorName, uc.d.Clock.Now())
	if err != nil {
		return domain.Comment{}, err
	}
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		if err := uc.d.Repository.Insert(ctx, comment); err != nil {
			return err
		}
		return uc.d.AuditTrail.RecordComment(ctx, comment)
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return comment, nil
}
