// Package postgres implementa los puertos de workflow con sqlc.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/workflow/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
)

// Repository lee la configuración de los flujos.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// Grants devuelve los permisos de los roles dados.
func (r *Repository) Grants(ctx context.Context, roles []string) ([]domain.Grant, error) {
	rows, err := r.queries(ctx).ListGrantsByRoles(ctx, roles)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	grants := make([]domain.Grant, len(rows))
	for i, row := range rows {
		grants[i] = domain.Grant{Role: row.Role, Permission: row.PermissionKey}
	}
	return grants, nil
}

// Workflow devuelve el flujo de un tipo de entidad con sus reglas.
func (r *Repository) Workflow(ctx context.Context, entityType domain.EntityType) (domain.Workflow, error) {
	q := r.queries(ctx)
	wf := domain.Workflow{EntityType: entityType}

	states, err := q.ListStates(ctx, string(entityType))
	if err != nil {
		return domain.Workflow{}, corepg.TranslateError(err)
	}
	for _, s := range states {
		state := domain.State{Id: s.ID.String(), Key: s.Key, Name: s.Name, IsInitial: s.IsInitial, IsFinal: s.IsFinal}
		if s.Position != nil {
			position := int(*s.Position)
			state.Position = &position
		}
		wf.States = append(wf.States, state)
	}

	rules, err := q.ListRules(ctx, string(entityType))
	if err != nil {
		return domain.Workflow{}, corepg.TranslateError(err)
	}
	rulesByTransition := map[string][]domain.Rule{}
	for _, row := range rules {
		rule := domain.Rule{Id: row.ID.String(), Key: row.RuleKey, Blocks: row.Blocks, Enabled: row.Enabled}
		if row.Route != nil {
			rule.Route = *row.Route
		}
		if err := json.Unmarshal(row.Params, &rule.Params); err != nil {
			return domain.Workflow{}, fmt.Errorf("leer params de la regla %s: %w", rule.Id, err)
		}
		id := row.TransitionID.String()
		rulesByTransition[id] = append(rulesByTransition[id], rule)
	}

	transitions, err := q.ListTransitions(ctx, string(entityType))
	if err != nil {
		return domain.Workflow{}, corepg.TranslateError(err)
	}
	for _, t := range transitions {
		wf.Transitions = append(wf.Transitions, domain.Transition{
			From:           t.FromKey,
			To:             t.ToKey,
			Permission:     t.PermissionKey,
			RequiresReason: t.RequiresReason,
			Rules:          rulesByTransition[t.ID.String()],
		})
	}
	return wf, nil
}
