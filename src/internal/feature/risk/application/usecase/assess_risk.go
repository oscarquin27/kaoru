package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/risk/application/port"
	"kaoru/internal/feature/risk/domain"
	shared "kaoru/internal/shared/domain"
)

// AssessRisk evalúa o reevalúa el riesgo de un REQ. Hay un solo RISK vigente por REQ:
// reevaluar lo actualiza y deja el cambio en la auditoría.
type AssessRisk struct {
	d Deps
}

// AssessRiskInput es la entrada de AssessRisk.
type AssessRiskInput struct {
	Actor           shared.Actor
	RequirementCode string
	Level           string
	Justification   string
	Characteristics []string
}

// Execute exige el permiso risk.assess.
func (uc *AssessRisk) Execute(ctx context.Context, in AssessRiskInput) (domain.Snapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionAssess); err != nil {
		return domain.Snapshot{}, err
	}
	requirement, err := uc.d.Requirements.Find(ctx, in.RequirementCode)
	if err != nil {
		return domain.Snapshot{}, err
	}
	now := uc.d.Clock.Now()
	input := domain.Input{
		Level: in.Level, Justification: in.Justification, Characteristics: in.Characteristics,
		AssessedBy: in.Actor.Id, Now: now,
	}
	var out domain.Snapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		current, err := uc.d.Repository.FindByRequirement(ctx, requirement.Id)
		switch {
		case axis.IsCode(err, axis.NotFoundResource):
			err = uc.create(ctx, requirement.Id, input)
		case err == nil:
			err = uc.reassess(ctx, current, input)
		}
		if err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindByRequirement(ctx, requirement.Id)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}

func (uc *AssessRisk) create(ctx context.Context, requirementId string, input domain.Input) error {
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return err
	}
	code, err := uc.d.Repository.NextCode(ctx, input.Now.Year())
	if err != nil {
		return err
	}
	assessment, err := domain.New(id, code, requirementId, input)
	if err != nil {
		return err
	}
	if err := uc.d.Repository.Insert(ctx, assessment); err != nil {
		return err
	}
	return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
		Actor: input.AssessedBy, RiskId: id, Code: code, Action: port.AuditCreate, NewValue: string(assessment.Snapshot().Level),
	})
}

func (uc *AssessRisk) reassess(ctx context.Context, assessment domain.Assessment, input domain.Input) error {
	changes, err := assessment.Reassess(input)
	if err != nil || len(changes) == 0 {
		return err
	}
	if err := uc.d.Repository.Update(ctx, assessment); err != nil {
		return err
	}
	s := assessment.Snapshot()
	for _, c := range changes {
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: input.AssessedBy, RiskId: s.Id, Code: s.Code, Action: port.AuditUpdate,
			Field: c.Field, OldValue: c.Old, NewValue: c.New,
		}); err != nil {
			return err
		}
	}
	return nil
}
