// Package postgres implementa los puertos de workflow con sqlc.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"

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

// Permissions devuelve los permisos que existen.
func (r *Repository) Permissions(ctx context.Context) ([]domain.Permission, error) {
	rows, err := r.queries(ctx).ListPermissions(ctx)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.Permission, len(rows))
	for i, row := range rows {
		out[i] = domain.Permission{Key: row.Key, Description: row.Description}
	}
	return out, nil
}

// AllGrants devuelve los permisos de todos los roles.
func (r *Repository) AllGrants(ctx context.Context) ([]domain.Grant, error) {
	rows, err := r.queries(ctx).ListAllGrants(ctx)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.Grant, len(rows))
	for i, row := range rows {
		out[i] = domain.Grant{Role: row.Role, Permission: row.PermissionKey}
	}
	return out, nil
}

// ReplaceRoleGrants deja al rol exactamente con esos permisos.
func (r *Repository) ReplaceRoleGrants(ctx context.Context, role string, permissions []string) error {
	q := r.queries(ctx)
	if err := q.DeleteRoleGrants(ctx, role); err != nil {
		return corepg.TranslateError(err)
	}
	for _, key := range permissions {
		if err := q.InsertGrant(ctx, sqlcgen.InsertGrantParams{Role: role, PermissionKey: key}); err != nil {
			return corepg.TranslateError(err)
		}
	}
	return nil
}

// FindRule busca una regla de bloqueo. Un id que no es un UUID tampoco existe.
func (r *Repository) FindRule(ctx context.Context, id string) (domain.Rule, error) {
	ruleID, err := uuid.Parse(id)
	if err != nil {
		return domain.Rule{}, axis.Wrap(axis.NotFoundResource, "La regla no existe.", err)
	}
	row, err := r.queries(ctx).GetRule(ctx, ruleID)
	if err != nil {
		return domain.Rule{}, corepg.TranslateError(err)
	}
	rule := domain.Rule{Id: row.ID.String(), Key: row.RuleKey, Blocks: row.Blocks, Enabled: row.Enabled}
	if row.Route != nil {
		rule.Route = *row.Route
	}
	if err := json.Unmarshal(row.Params, &rule.Params); err != nil {
		return domain.Rule{}, fmt.Errorf("leer params de la regla %s: %w", rule.Id, err)
	}
	return rule, nil
}

// UpdateRule guarda los cambios de una regla de bloqueo.
func (r *Repository) UpdateRule(ctx context.Context, rule domain.Rule, now time.Time) error {
	id, err := uuid.Parse(rule.Id)
	if err != nil {
		return axis.Wrap(axis.InternalServerError, "id de regla no válido", err)
	}
	params, err := json.Marshal(rule.Params)
	if err != nil {
		return fmt.Errorf("guardar params de la regla: %w", err)
	}
	rows, err := r.queries(ctx).UpdateRule(ctx, sqlcgen.UpdateRuleParams{
		ID: id, Params: params, Blocks: rule.Blocks, Enabled: rule.Enabled, UpdatedAt: now,
	})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "La regla no existe.")
	}
	return nil
}
