package usecase

import (
	"context"
	"encoding/json"
	"maps"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateBlockingRule activa, desactiva o ajusta una regla de bloqueo.
type UpdateBlockingRule struct {
	d Deps
}

// UpdateBlockingRuleInput es la entrada de UpdateBlockingRule; nil deja el campo como está.
type UpdateBlockingRuleInput struct {
	Actor   shared.Actor
	RuleId  string
	Enabled *bool
	Blocks  *bool
	Params  map[string]any
}

// Execute exige el permiso admin.configure.
func (uc *UpdateBlockingRule) Execute(ctx context.Context, in UpdateBlockingRuleInput) (domain.Rule, error) {
	repo := uc.d.Repository
	if err := authorize(ctx, repo, in.Actor.Roles, domain.PermissionConfigure); err != nil {
		return domain.Rule{}, err
	}
	var rule domain.Rule
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		var err error
		if rule, err = repo.FindRule(ctx, in.RuleId); err != nil {
			if axis.IsCode(err, axis.NotFoundResource) {
				return axis.New(axis.NotFoundResource, "La regla no existe.")
			}
			return err
		}
		before := describe(rule)
		if in.Enabled != nil {
			rule.Enabled = *in.Enabled
		}
		if in.Blocks != nil {
			rule.Blocks = *in.Blocks
		}
		if in.Params != nil {
			rule.Params = maps.Clone(in.Params)
		}
		after := describe(rule)
		if before == after {
			return nil
		}
		if err := repo.UpdateRule(ctx, rule, uc.d.Clock.Now()); err != nil {
			return err
		}
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, Entity: "blocking_rule", Key: rule.Id, Field: rule.Key, OldValue: before, NewValue: after,
		})
	})
	if err != nil {
		return domain.Rule{}, err
	}
	return rule, nil
}

// describe resume una regla en una línea, para comparar y para la auditoría.
func describe(r domain.Rule) string {
	raw, _ := json.Marshal(struct {
		Enabled bool           `json:"enabled"`
		Blocks  bool           `json:"blocks"`
		Params  map[string]any `json:"params"`
	}{r.Enabled, r.Blocks, r.Params})
	return string(raw)
}
