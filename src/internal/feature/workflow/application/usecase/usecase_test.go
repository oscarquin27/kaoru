package usecase_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/application/usecase"
	"kaoru/internal/feature/workflow/domain"
)

// fakeRepo es un flujo en memoria: Registrado → Aprobado (chg.approve),
// Registrado → Rechazado (chg.reject, con motivo y una regla apagada).
type fakeRepo struct{}

func (fakeRepo) Grants(_ context.Context, roles []string) ([]domain.Grant, error) {
	all := []domain.Grant{
		{Role: "change-manager", Permission: "chg.approve"},
		{Role: "change-manager", Permission: "chg.reject"},
		{Role: "qa", Permission: "run.create"},
	}
	var out []domain.Grant
	for _, g := range all {
		if slices.Contains(roles, g.Role) {
			out = append(out, g)
		}
	}
	return out, nil
}

func (fakeRepo) Workflow(_ context.Context, t domain.EntityType) (domain.Workflow, error) {
	one := 1
	return domain.Workflow{
		EntityType: t,
		States: []domain.State{
			{Id: "s1", Key: "registered", Name: "Registrado", Position: &one, IsInitial: true},
			{Id: "s2", Key: "approved", Name: "Aprobado"},
			{Id: "s3", Key: "rejected", Name: "Rechazado", IsFinal: true},
		},
		Transitions: []domain.Transition{
			{From: "registered", To: "approved", Permission: "chg.approve"},
			{From: "registered", To: "rejected", Permission: "chg.reject", RequiresReason: true, Rules: []domain.Rule{
				{Id: "r1", Key: "activa", Enabled: true, Blocks: true},
				{Id: "r2", Key: "apagada", Enabled: false, Blocks: true},
			}},
		},
	}, nil
}

func (fakeRepo) Permissions(context.Context) ([]domain.Permission, error)  { return nil, nil }
func (fakeRepo) AllGrants(context.Context) ([]domain.Grant, error)         { return nil, nil }
func (fakeRepo) ReplaceRoleGrants(context.Context, string, []string) error { return nil }
func (fakeRepo) FindRule(context.Context, string) (domain.Rule, error)     { return domain.Rule{}, nil }
func (fakeRepo) UpdateRule(context.Context, domain.Rule, time.Time) error  { return nil }
func (fakeRepo) InsertState(context.Context, domain.EntityType, domain.State, time.Time) error {
	return nil
}
func (fakeRepo) UpdateState(context.Context, domain.EntityType, domain.State, time.Time) error {
	return nil
}
func (fakeRepo) DeleteState(context.Context, domain.EntityType, string) error { return nil }
func (fakeRepo) SaveTransition(context.Context, domain.EntityType, domain.Transition, time.Time) error {
	return nil
}
func (fakeRepo) DeleteTransition(context.Context, domain.EntityType, string, string) error {
	return nil
}

type noAudit struct{}

func (noAudit) Record(context.Context, port.AuditRecord) error { return nil }

type noTx struct{}

func (noTx) Within(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC) }

func newUseCases(t *testing.T) *usecase.UseCases {
	t.Helper()
	uc, err := usecase.New(usecase.Deps{Repository: fakeRepo{}, AuditTrail: noAudit{}, UnitOfWork: noTx{}, Clock: fixedClock{}})
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

func TestNewRequiresRepository(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin Repository")
	}
}

func TestCheckPermission(t *testing.T) {
	uc := newUseCases(t)
	cases := []struct {
		name       string
		roles      []string
		permission string
		wantCode   axis.Code
	}{
		{"el rol tiene el permiso", []string{"change-manager"}, "chg.approve", ""},
		{"basta uno de los roles", []string{"qa", "change-manager"}, "chg.approve", ""},
		{"otro rol no lo tiene", []string{"qa"}, "chg.approve", axis.PermissionDenied},
		{"sin roles", nil, "chg.approve", axis.PermissionDenied},
		{"rol desconocido", []string{"offline_access"}, "chg.approve", axis.PermissionDenied},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := uc.CheckPermission.Execute(context.Background(), usecase.CheckPermissionInput{Roles: tc.roles, Permission: tc.permission})
			if tc.wantCode == "" && err != nil {
				t.Fatalf("no esperaba error: %v", err)
			}
			if tc.wantCode != "" && !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
		})
	}
}

func TestListPermissionsDropsUnknownRoles(t *testing.T) {
	out, err := newUseCases(t).ListPermissions.Execute(context.Background(),
		usecase.ListPermissionsInput{Roles: []string{"offline_access", "qa", "change-manager"}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(out.Roles, []string{"change-manager", "qa"}) {
		t.Errorf("roles = %v", out.Roles)
	}
	if !slices.Equal(out.Permissions, []string{"chg.approve", "chg.reject", "run.create"}) {
		t.Errorf("permisos = %v", out.Permissions)
	}
}

func TestResolveTransition(t *testing.T) {
	uc := newUseCases(t)
	cases := []struct {
		name       string
		in         usecase.ResolveTransitionInput
		wantCode   axis.Code
		wantReason bool
		wantRules  int
	}{
		{"transición permitida", usecase.ResolveTransitionInput{EntityType: "change", From: "registered", To: "approved", Roles: []string{"change-manager"}}, "", false, 0},
		{"exige motivo y solo devuelve reglas activas", usecase.ResolveTransitionInput{EntityType: "change", From: "registered", To: "rejected", Roles: []string{"change-manager"}}, "", true, 1},
		{"el flujo no permite el paso", usecase.ResolveTransitionInput{EntityType: "change", From: "approved", To: "registered", Roles: []string{"change-manager"}}, axis.InvalidAction, false, 0},
		{"estado destino desconocido", usecase.ResolveTransitionInput{EntityType: "change", From: "registered", To: "inventado", Roles: []string{"change-manager"}}, axis.InvalidRequest, false, 0},
		{"el rol no tiene el permiso de la transición", usecase.ResolveTransitionInput{EntityType: "change", From: "registered", To: "approved", Roles: []string{"qa"}}, axis.PermissionDenied, false, 0},
		{"tipo de entidad desconocido", usecase.ResolveTransitionInput{EntityType: "paquete", From: "a", To: "b"}, axis.NotFoundResource, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := uc.ResolveTransition.Execute(context.Background(), tc.in)
			if tc.wantCode != "" {
				if !axis.IsCode(err, tc.wantCode) {
					t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if out.To.Key != tc.in.To || out.RequiresReason != tc.wantReason || len(out.Rules) != tc.wantRules {
				t.Errorf("salida inesperada: %+v", out)
			}
		})
	}
}

func TestGetInitialState(t *testing.T) {
	state, err := newUseCases(t).GetInitialState.Execute(context.Background(), usecase.GetInitialStateInput{EntityType: "change"})
	if err != nil || state.Key != "registered" {
		t.Fatalf("estado inicial = %+v, err = %v", state, err)
	}
}

func TestValidateRolePermissions(t *testing.T) {
	known := []domain.Permission{{Key: "admin.configure"}, {Key: "run.create"}, {Key: "pkg.build"}}
	cases := []struct {
		name        string
		role        string
		permissions []string
		wantCode    axis.Code
		want        []string
	}{
		{"ordena y quita repetidos", "dev", []string{"run.create", "pkg.build", "run.create"}, "", []string{"pkg.build", "run.create"}},
		{"un rol puede quedarse sin permisos", "ops", nil, "", []string{}},
		{"rol desconocido", "root", []string{"run.create"}, axis.NotFoundResource, nil},
		{"permiso desconocido", "dev", []string{"todo"}, axis.InvalidRequest, nil},
		{"qa-admin no pierde el permiso de configurar", "qa-admin", []string{"run.create"}, axis.InvalidAction, nil},
		{"qa-admin lo conserva", "qa-admin", []string{"run.create", "admin.configure"}, "", []string{"admin.configure", "run.create"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.ValidateRolePermissions(tc.role, tc.permissions, known)
			if tc.wantCode != "" {
				if !axis.IsCode(err, tc.wantCode) {
					t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
				}
				return
			}
			if err != nil || !slices.Equal(got, tc.want) {
				t.Errorf("permisos = %v, %v; quiero %v", got, err, tc.want)
			}
		})
	}
}
