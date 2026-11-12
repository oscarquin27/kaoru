package usecase_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/application/usecase"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// config es la configuración en memoria de los dobles: un flujo, permisos, asignaciones y una regla.
// Lo escrito dentro de Within se descarta si la función falla, como en una transacción.
type config struct {
	states      []domain.State
	transitions []domain.Transition
	grants      []domain.Grant
	rule        domain.Rule
	audit       []port.AuditRecord
	auditErr    error
	// inUse es la clave de un estado con registros: la base impide quitarlo.
	inUse string
}

func newConfig() *config {
	one, two := 1, 2
	return &config{
		states: []domain.State{
			{Id: "s1", Key: "registered", Name: "Registrado", Position: &one, IsInitial: true},
			{Id: "s2", Key: "approved", Name: "Aprobado", Position: &two},
			{Id: "s3", Key: "rejected", Name: "Rechazado", IsFinal: true},
			{Id: "s4", Key: "on_hold", Name: "En espera"},
		},
		transitions: []domain.Transition{
			{From: "registered", To: "approved", Permission: "chg.approve"},
			{From: "registered", To: "rejected", Permission: "chg.reject", RequiresReason: true},
		},
		grants: []domain.Grant{
			{Role: "qa-admin", Permission: "admin.configure"},
			{Role: "qa", Permission: "run.create"},
		},
		rule: domain.Rule{Id: "r1", Key: "min_coverage", Params: map[string]any{"high": 100.0}, Blocks: true, Enabled: true},
	}
}

func (c *config) Within(ctx context.Context, fn func(context.Context) error) error {
	saved := *c
	saved.states, saved.transitions = slices.Clone(c.states), slices.Clone(c.transitions)
	saved.grants, saved.audit = slices.Clone(c.grants), slices.Clone(c.audit)
	if err := fn(ctx); err != nil {
		*c = saved
		return err
	}
	return nil
}

func (c *config) Record(_ context.Context, r port.AuditRecord) error {
	if c.auditErr != nil {
		return c.auditErr
	}
	c.audit = append(c.audit, r)
	return nil
}

func (c *config) Grants(_ context.Context, roles []string) ([]domain.Grant, error) {
	var out []domain.Grant
	for _, g := range c.grants {
		if slices.Contains(roles, g.Role) {
			out = append(out, g)
		}
	}
	return out, nil
}

func (c *config) Workflow(_ context.Context, t domain.EntityType) (domain.Workflow, error) {
	return domain.Workflow{EntityType: t, States: slices.Clone(c.states), Transitions: slices.Clone(c.transitions)}, nil
}

func (c *config) Permissions(context.Context) ([]domain.Permission, error) {
	return []domain.Permission{{Key: "admin.configure"}, {Key: "chg.approve"}, {Key: "chg.reject"}, {Key: "run.create"}}, nil
}

func (c *config) AllGrants(context.Context) ([]domain.Grant, error) { return c.grants, nil }

func (c *config) ReplaceRoleGrants(_ context.Context, role string, permissions []string) error {
	c.grants = slices.DeleteFunc(c.grants, func(g domain.Grant) bool { return g.Role == role })
	for _, p := range permissions {
		c.grants = append(c.grants, domain.Grant{Role: role, Permission: p})
	}
	return nil
}

func (c *config) FindRule(_ context.Context, id string) (domain.Rule, error) {
	if id != c.rule.Id {
		return domain.Rule{}, axis.New(axis.NotFoundResource, "No existe.")
	}
	return c.rule, nil
}

func (c *config) UpdateRule(_ context.Context, rule domain.Rule, _ time.Time) error {
	c.rule = rule
	return nil
}

func (c *config) InsertState(_ context.Context, _ domain.EntityType, s domain.State, _ time.Time) error {
	c.states = append(c.states, s)
	return nil
}

func (c *config) UpdateState(_ context.Context, _ domain.EntityType, s domain.State, _ time.Time) error {
	for i := range c.states {
		if c.states[i].Key == s.Key {
			c.states[i] = s
		}
	}
	return nil
}

func (c *config) DeleteState(_ context.Context, _ domain.EntityType, key string) error {
	if key == c.inUse {
		return axis.New(axis.InvalidAction, "Hay registros en este estado.")
	}
	c.states = slices.DeleteFunc(c.states, func(s domain.State) bool { return s.Key == key })
	return nil
}

func (c *config) SaveTransition(_ context.Context, _ domain.EntityType, t domain.Transition, _ time.Time) error {
	for i := range c.transitions {
		if c.transitions[i].From == t.From && c.transitions[i].To == t.To {
			c.transitions[i].Permission, c.transitions[i].RequiresReason = t.Permission, t.RequiresReason
			return nil
		}
	}
	c.transitions = append(c.transitions, t)
	return nil
}

func (c *config) DeleteTransition(_ context.Context, _ domain.EntityType, from, to string) error {
	c.transitions = slices.DeleteFunc(c.transitions, func(t domain.Transition) bool { return t.From == from && t.To == to })
	return nil
}

func adminUseCases(t *testing.T, c *config) *usecase.UseCases {
	t.Helper()
	uc, err := usecase.New(usecase.Deps{Repository: c, AuditTrail: c, UnitOfWork: c, Clock: fixedClock{}})
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

var (
	admin  = shared.Actor{Id: "sub-admin", Roles: []string{"qa-admin"}}
	tester = shared.Actor{Id: "sub-qa", Roles: []string{"qa"}}
	ctx    = context.Background()
)

func intPtr(n int) *int { return &n }

func strPtr(s string) *string { return &s }

// Cada caso de uso de administración niega a quien no tiene admin.configure y no guarda nada.
func TestAdministrationRequiresConfigurePermission(t *testing.T) {
	calls := map[string]func(uc *usecase.UseCases) error{
		"ListRolePermissions": func(uc *usecase.UseCases) error {
			_, err := uc.ListRolePermissions.Execute(ctx, usecase.ListRolePermissionsInput{Actor: tester})
			return err
		},
		"SetRolePermissions": func(uc *usecase.UseCases) error {
			_, err := uc.SetRolePermissions.Execute(ctx, usecase.SetRolePermissionsInput{Actor: tester, Role: "qa", Permissions: []string{"admin.configure"}})
			return err
		},
		"UpdateBlockingRule": func(uc *usecase.UseCases) error {
			off := false
			_, err := uc.UpdateBlockingRule.Execute(ctx, usecase.UpdateBlockingRuleInput{Actor: tester, RuleId: "r1", Enabled: &off})
			return err
		},
		"AddState": func(uc *usecase.UseCases) error {
			_, err := uc.AddState.Execute(ctx, usecase.AddStateInput{Actor: tester, EntityType: "change", Key: "paused", Name: "Pausado"})
			return err
		},
		"UpdateState": func(uc *usecase.UseCases) error {
			_, err := uc.UpdateState.Execute(ctx, usecase.UpdateStateInput{Actor: tester, EntityType: "change", Key: "approved", Edit: domain.StateEdit{Name: strPtr("Otro")}})
			return err
		},
		"RemoveState": func(uc *usecase.UseCases) error {
			_, err := uc.RemoveState.Execute(ctx, usecase.RemoveStateInput{Actor: tester, EntityType: "change", Key: "on_hold"})
			return err
		},
		"SetTransition": func(uc *usecase.UseCases) error {
			_, err := uc.SetTransition.Execute(ctx, usecase.SetTransitionInput{Actor: tester, EntityType: "change", From: "approved", To: "on_hold", Permission: "chg.approve"})
			return err
		},
		"RemoveTransition": func(uc *usecase.UseCases) error {
			_, err := uc.RemoveTransition.Execute(ctx, usecase.RemoveTransitionInput{Actor: tester, EntityType: "change", From: "registered", To: "approved"})
			return err
		},
	}
	for name, call := range calls {
		c := newConfig()
		err := call(adminUseCases(t, c))
		if !errors.Is(err, domain.ErrPermissionDenied) {
			t.Errorf("%s: quiero ErrPermissionDenied, obtuve %v", name, err)
		}
		if len(c.states) != 4 || len(c.transitions) != 2 || len(c.grants) != 2 || !c.rule.Enabled || len(c.audit) != 0 {
			t.Errorf("%s: un permiso negado no debe cambiar ni auditar nada", name)
		}
	}
}

func TestListRolePermissionsListsEveryRole(t *testing.T) {
	out, err := adminUseCases(t, newConfig()).ListRolePermissions.Execute(ctx, usecase.ListRolePermissionsInput{Actor: admin})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Roles) != len(domain.Roles) || len(out.Permissions) != 4 {
		t.Fatalf("quiero todos los roles y permisos: %+v", out)
	}
	for _, r := range out.Roles {
		if r.Permissions == nil {
			t.Errorf("el rol %s sin permisos debe traer una lista vacía, no nil", r.Role)
		}
	}
}

func TestSetRolePermissionsReplacesAndAudits(t *testing.T) {
	c := newConfig()
	uc := adminUseCases(t, c)
	in := usecase.SetRolePermissionsInput{Actor: admin, Role: "qa", Permissions: []string{"run.create", "chg.approve"}}
	if _, err := uc.SetRolePermissions.Execute(ctx, in); err != nil {
		t.Fatal(err)
	}
	if len(c.audit) != 1 || c.audit[0].OldValue != "run.create" || c.audit[0].NewValue != "chg.approve,run.create" || c.audit[0].Actor != admin.Id {
		t.Fatalf("auditoría inesperada: %+v", c.audit)
	}
	// Repetir lo mismo no cambia ni audita.
	if _, err := uc.SetRolePermissions.Execute(ctx, in); err != nil || len(c.audit) != 1 {
		t.Errorf("un cambio sin diferencias no se audita: %v, %d registros", err, len(c.audit))
	}
}

func TestSetRolePermissionsBusinessErrors(t *testing.T) {
	cases := []struct {
		name string
		in   usecase.SetRolePermissionsInput
		code axis.Code
	}{
		{"rol desconocido", usecase.SetRolePermissionsInput{Actor: admin, Role: "root"}, axis.NotFoundResource},
		{"permiso desconocido", usecase.SetRolePermissionsInput{Actor: admin, Role: "qa", Permissions: []string{"volar"}}, axis.InvalidRequest},
		{"qa-admin sin configurar", usecase.SetRolePermissionsInput{Actor: admin, Role: "qa-admin", Permissions: []string{"run.create"}}, axis.InvalidAction},
	}
	for _, tc := range cases {
		c := newConfig()
		_, err := adminUseCases(t, c).SetRolePermissions.Execute(ctx, tc.in)
		if !axis.IsCode(err, tc.code) || len(c.grants) != 2 || len(c.audit) != 0 {
			t.Errorf("%s: quiero %s sin cambios, obtuve %v", tc.name, tc.code, err)
		}
	}
}

func TestUpdateBlockingRule(t *testing.T) {
	c := newConfig()
	uc := adminUseCases(t, c)
	off := false
	rule, err := uc.UpdateBlockingRule.Execute(ctx, usecase.UpdateBlockingRuleInput{
		Actor: admin, RuleId: "r1", Enabled: &off, Params: map[string]any{"high": 90.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule.Enabled || !rule.Blocks || rule.Params["high"] != 90.0 || c.rule.Enabled {
		t.Errorf("la regla debía quedar apagada, con bloqueo y cobertura 90: %+v", rule)
	}
	if len(c.audit) != 1 || c.audit[0].Entity != "blocking_rule" || !strings.Contains(c.audit[0].OldValue, `"enabled":true`) ||
		!strings.Contains(c.audit[0].NewValue, `"enabled":false`) {
		t.Errorf("auditoría inesperada: %+v", c.audit)
	}
	if _, err := uc.UpdateBlockingRule.Execute(ctx, usecase.UpdateBlockingRuleInput{Actor: admin, RuleId: "r1", Enabled: &off}); err != nil || len(c.audit) != 1 {
		t.Errorf("un cambio sin diferencias no se audita: %v", err)
	}
	if _, err := uc.UpdateBlockingRule.Execute(ctx, usecase.UpdateBlockingRuleInput{Actor: admin, RuleId: "nope"}); !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("regla inexistente: quiero NOT_FOUND_RESOURCE, obtuve %v", err)
	}
}

func TestAddState(t *testing.T) {
	c := newConfig()
	wf, err := adminUseCases(t, c).AddState.Execute(ctx, usecase.AddStateInput{
		Actor: admin, EntityType: "change", Key: "paused", Name: "  Pausado ", Position: intPtr(3),
	})
	if err != nil {
		t.Fatal(err)
	}
	added, err := wf.State("paused")
	if err != nil || added.Name != "Pausado" || *added.Position != 3 || added.IsInitial || added.IsFinal {
		t.Errorf("estado agregado inesperado: %+v, %v", added, err)
	}
	if len(c.audit) != 1 || c.audit[0].Entity != "workflow_state" || c.audit[0].Key != "change:paused" || c.audit[0].OldValue != "" {
		t.Errorf("auditoría inesperada: %+v", c.audit)
	}
}

func TestAddStateBusinessErrors(t *testing.T) {
	cases := []struct {
		name string
		in   usecase.AddStateInput
		code axis.Code
	}{
		{"flujo desconocido", usecase.AddStateInput{EntityType: "risk", Key: "paused", Name: "Pausado"}, axis.NotFoundResource},
		{"clave inválida", usecase.AddStateInput{EntityType: "change", Key: "En Pausa", Name: "Pausado"}, axis.InvalidRequest},
		{"clave repetida", usecase.AddStateInput{EntityType: "change", Key: "approved", Name: "Otro"}, axis.DuplicateResource},
		{"sin nombre", usecase.AddStateInput{EntityType: "change", Key: "paused", Name: "  "}, axis.InvalidRequest},
		{"nombre largo", usecase.AddStateInput{EntityType: "change", Key: "paused", Name: strings.Repeat("ñ", 61)}, axis.InvalidRequest},
		{"posición cero", usecase.AddStateInput{EntityType: "change", Key: "paused", Name: "Pausado", Position: intPtr(0)}, axis.InvalidRequest},
	}
	for _, tc := range cases {
		c := newConfig()
		tc.in.Actor = admin
		_, err := adminUseCases(t, c).AddState.Execute(ctx, tc.in)
		if !axis.IsCode(err, tc.code) || len(c.states) != 4 || len(c.audit) != 0 {
			t.Errorf("%s: quiero %s sin cambios, obtuve %v", tc.name, tc.code, err)
		}
	}
}

func TestUpdateState(t *testing.T) {
	c := newConfig()
	uc := adminUseCases(t, c)
	wf, err := uc.UpdateState.Execute(ctx, usecase.UpdateStateInput{
		Actor: admin, EntityType: "change", Key: "approved", Edit: domain.StateEdit{Name: strPtr("Autorizado"), PositionSet: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	edited, _ := wf.State("approved")
	if edited.Name != "Autorizado" || edited.Position != nil {
		t.Errorf("debía cambiar de nombre y pasar a lateral: %+v", edited)
	}
	if len(c.audit) != 1 || !strings.Contains(c.audit[0].OldValue, "Aprobado") || !strings.Contains(c.audit[0].NewValue, "Autorizado") {
		t.Errorf("auditoría inesperada: %+v", c.audit)
	}
	// Solo el nombre: la posición queda como está.
	wf, err = uc.UpdateState.Execute(ctx, usecase.UpdateStateInput{
		Actor: admin, EntityType: "change", Key: "registered", Edit: domain.StateEdit{Name: strPtr("Registrado")},
	})
	registered, _ := wf.State("registered")
	if err != nil || *registered.Position != 1 || len(c.audit) != 1 {
		t.Errorf("un cambio sin diferencias no toca ni audita: %v, %+v", err, registered)
	}

	for name, in := range map[string]usecase.UpdateStateInput{
		"estado inexistente": {EntityType: "change", Key: "nope", Edit: domain.StateEdit{Name: strPtr("X")}},
		"flujo desconocido":  {EntityType: "risk", Key: "approved"},
	} {
		in.Actor = admin
		if _, err := uc.UpdateState.Execute(ctx, in); !axis.IsCode(err, axis.NotFoundResource) {
			t.Errorf("%s: quiero NOT_FOUND_RESOURCE, obtuve %v", name, err)
		}
	}
	_, err = uc.UpdateState.Execute(ctx, usecase.UpdateStateInput{
		Actor: admin, EntityType: "change", Key: "approved", Edit: domain.StateEdit{PositionSet: true, Position: intPtr(-1)},
	})
	if !axis.IsCode(err, axis.InvalidRequest) {
		t.Errorf("posición negativa: quiero INVALID_REQUEST, obtuve %v", err)
	}
}

func TestRemoveState(t *testing.T) {
	c := newConfig()
	uc := adminUseCases(t, c)
	wf, err := uc.RemoveState.Execute(ctx, usecase.RemoveStateInput{Actor: admin, EntityType: "change", Key: "on_hold"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wf.State("on_hold"); err == nil || len(c.audit) != 1 || c.audit[0].NewValue != "" || !strings.Contains(c.audit[0].OldValue, "En espera") {
		t.Errorf("el estado debía desaparecer y quedar auditado: %+v", c.audit)
	}

	cases := []struct {
		name string
		key  string
		code axis.Code
	}{
		{"inexistente", "nope", axis.NotFoundResource},
		{"inicial", "registered", axis.InvalidAction},
		{"con transiciones", "approved", axis.InvalidAction},
	}
	for _, tc := range cases {
		c := newConfig()
		_, err := adminUseCases(t, c).RemoveState.Execute(ctx, usecase.RemoveStateInput{Actor: admin, EntityType: "change", Key: tc.key})
		if !axis.IsCode(err, tc.code) || len(c.states) != 4 || len(c.audit) != 0 {
			t.Errorf("%s: quiero %s sin cambios, obtuve %v", tc.name, tc.code, err)
		}
	}

	c = newConfig()
	c.inUse = "on_hold"
	_, err = adminUseCases(t, c).RemoveState.Execute(ctx, usecase.RemoveStateInput{Actor: admin, EntityType: "change", Key: "on_hold"})
	if !axis.IsCode(err, axis.InvalidAction) || len(c.states) != 4 {
		t.Errorf("un estado con registros no se quita: %v", err)
	}
}

func TestSetTransition(t *testing.T) {
	c := newConfig()
	uc := adminUseCases(t, c)
	in := usecase.SetTransitionInput{Actor: admin, EntityType: "change", From: "approved", To: "on_hold", Permission: "chg.approve"}
	wf, err := uc.SetTransition.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := wf.FindTransition("approved", "on_hold"); !ok || len(c.audit) != 1 || c.audit[0].Key != "change:approved>on_hold" || c.audit[0].OldValue != "" {
		t.Fatalf("la transición nueva debía existir y quedar auditada: %+v", c.audit)
	}
	// Cambiar el permiso y exigir motivo de una que ya existe.
	in.Permission, in.RequiresReason = "chg.reject", true
	wf, err = uc.SetTransition.Execute(ctx, in)
	changed, _ := wf.FindTransition("approved", "on_hold")
	if err != nil || changed.Permission != "chg.reject" || !changed.RequiresReason || len(wf.Transitions) != 3 || len(c.audit) != 2 {
		t.Errorf("debía actualizar la transición, no duplicarla: %+v, %v", changed, err)
	}
	if _, err := uc.SetTransition.Execute(ctx, in); err != nil || len(c.audit) != 2 {
		t.Errorf("un cambio sin diferencias no se audita: %v", err)
	}
}

func TestSetTransitionBusinessErrors(t *testing.T) {
	cases := []struct {
		name string
		in   usecase.SetTransitionInput
		code axis.Code
	}{
		{"origen inexistente", usecase.SetTransitionInput{EntityType: "change", From: "nope", To: "approved", Permission: "chg.approve"}, axis.NotFoundResource},
		{"destino inexistente", usecase.SetTransitionInput{EntityType: "change", From: "approved", To: "nope", Permission: "chg.approve"}, axis.NotFoundResource},
		{"mismo estado", usecase.SetTransitionInput{EntityType: "change", From: "approved", To: "approved", Permission: "chg.approve"}, axis.InvalidAction},
		{"sin permiso", usecase.SetTransitionInput{EntityType: "change", From: "approved", To: "on_hold"}, axis.InvalidRequest},
		{"permiso desconocido", usecase.SetTransitionInput{EntityType: "change", From: "approved", To: "on_hold", Permission: "volar"}, axis.InvalidRequest},
	}
	for _, tc := range cases {
		c := newConfig()
		tc.in.Actor = admin
		_, err := adminUseCases(t, c).SetTransition.Execute(ctx, tc.in)
		if !axis.IsCode(err, tc.code) || len(c.transitions) != 2 || len(c.audit) != 0 {
			t.Errorf("%s: quiero %s sin cambios, obtuve %v", tc.name, tc.code, err)
		}
	}
}

func TestRemoveTransition(t *testing.T) {
	c := newConfig()
	uc := adminUseCases(t, c)
	wf, err := uc.RemoveTransition.Execute(ctx, usecase.RemoveTransitionInput{Actor: admin, EntityType: "change", From: "registered", To: "rejected"})
	if err != nil {
		t.Fatal(err)
	}
	if len(wf.Transitions) != 1 || len(c.audit) != 1 || !strings.Contains(c.audit[0].OldValue, "chg.reject") || c.audit[0].NewValue != "" {
		t.Errorf("la transición debía desaparecer y quedar auditada: %+v", c.audit)
	}
	_, err = uc.RemoveTransition.Execute(ctx, usecase.RemoveTransitionInput{Actor: admin, EntityType: "change", From: "registered", To: "rejected"})
	if !errors.Is(err, domain.ErrTransitionMissing) {
		t.Errorf("quitar dos veces: quiero ErrTransitionMissing, obtuve %v", err)
	}
}

// Si la auditoría falla, el cambio de configuración no se guarda.
func TestNothingIsSavedWhenTheAuditFails(t *testing.T) {
	boom := errors.New("auditoría caída")
	off := false
	calls := map[string]func(uc *usecase.UseCases) error{
		"SetRolePermissions": func(uc *usecase.UseCases) error {
			_, err := uc.SetRolePermissions.Execute(ctx, usecase.SetRolePermissionsInput{Actor: admin, Role: "qa", Permissions: []string{"chg.approve"}})
			return err
		},
		"UpdateBlockingRule": func(uc *usecase.UseCases) error {
			_, err := uc.UpdateBlockingRule.Execute(ctx, usecase.UpdateBlockingRuleInput{Actor: admin, RuleId: "r1", Enabled: &off})
			return err
		},
		"AddState": func(uc *usecase.UseCases) error {
			_, err := uc.AddState.Execute(ctx, usecase.AddStateInput{Actor: admin, EntityType: "change", Key: "paused", Name: "Pausado"})
			return err
		},
		"UpdateState": func(uc *usecase.UseCases) error {
			_, err := uc.UpdateState.Execute(ctx, usecase.UpdateStateInput{Actor: admin, EntityType: "change", Key: "approved", Edit: domain.StateEdit{Name: strPtr("Otro")}})
			return err
		},
		"RemoveState": func(uc *usecase.UseCases) error {
			_, err := uc.RemoveState.Execute(ctx, usecase.RemoveStateInput{Actor: admin, EntityType: "change", Key: "on_hold"})
			return err
		},
		"SetTransition": func(uc *usecase.UseCases) error {
			_, err := uc.SetTransition.Execute(ctx, usecase.SetTransitionInput{Actor: admin, EntityType: "change", From: "approved", To: "on_hold", Permission: "chg.approve"})
			return err
		},
		"RemoveTransition": func(uc *usecase.UseCases) error {
			_, err := uc.RemoveTransition.Execute(ctx, usecase.RemoveTransitionInput{Actor: admin, EntityType: "change", From: "registered", To: "approved"})
			return err
		},
	}
	for name, call := range calls {
		c := newConfig()
		c.auditErr = boom
		if err := call(adminUseCases(t, c)); !errors.Is(err, boom) {
			t.Errorf("%s: quiero el error de auditoría, obtuve %v", name, err)
		}
		approved := c.states[1]
		if len(c.states) != 4 || approved.Name != "Aprobado" || len(c.transitions) != 2 || len(c.grants) != 2 || !c.rule.Enabled {
			t.Errorf("%s: nada debía guardarse si la auditoría falla", name)
		}
	}
}
