package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"kaoru/internal/feature/user/application/usecase"
	"kaoru/internal/feature/user/domain"
)

var now = time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return now }

type fakeRepository struct {
	saved     []domain.Profile
	savedAt   time.Time
	listQuery string
	listLimit int
	err       error
}

func (r *fakeRepository) Upsert(_ context.Context, p domain.Profile, at time.Time) error {
	if r.err != nil {
		return r.err
	}
	r.saved, r.savedAt = append(r.saved, p), at
	return nil
}

func (r *fakeRepository) List(_ context.Context, query string, limit int) ([]domain.Profile, error) {
	r.listQuery, r.listLimit = query, limit
	return r.saved, r.err
}

type fakePermissions struct{ err error }

func (p fakePermissions) Granted(_ context.Context, roles []string) ([]string, []string, error) {
	if p.err != nil {
		return nil, nil, p.err
	}
	var known []string
	for _, r := range roles {
		if r == "qa" {
			known = append(known, r)
		}
	}
	return known, []string{"requirement.read"}, nil
}

func build(t *testing.T, repo *fakeRepository, perms fakePermissions) *usecase.UseCases {
	t.Helper()
	uc, err := usecase.New(usecase.Deps{
		Repository: repo, Permissions: perms, Clock: fixedClock{},
		LogoutURL: "/oauth2/sign_out", AccountURL: "https://kaoru.lab.invalid/account",
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

var arojas = usecase.ProfileInput{Id: "sub-qa", Username: " arojas ", DisplayName: "", Email: "arojas@lab.invalid"}

func TestNewRequiresDependencies(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New sin dependencias debía fallar")
	}
}

func TestSyncProfileSavesTheTokenIdentity(t *testing.T) {
	repo := &fakeRepository{}
	if err := build(t, repo, fakePermissions{}).SyncProfile.Execute(context.Background(), arojas); err != nil {
		t.Fatal(err)
	}
	want := domain.Profile{Id: "sub-qa", Username: "arojas", DisplayName: "arojas", Email: "arojas@lab.invalid"}
	if len(repo.saved) != 1 || repo.saved[0] != want || !repo.savedAt.Equal(now) {
		t.Fatalf("guardó %+v a las %v, quiero %+v a las %v", repo.saved, repo.savedAt, want, now)
	}
}

func TestSyncProfileRejectsATokenWithoutIdentity(t *testing.T) {
	for _, in := range []usecase.ProfileInput{{Username: "arojas"}, {Id: "sub-qa", Username: "  "}} {
		repo := &fakeRepository{}
		err := build(t, repo, fakePermissions{}).SyncProfile.Execute(context.Background(), in)
		if !errors.Is(err, domain.ErrInvalidProfile) {
			t.Errorf("%+v: quiero ErrInvalidProfile, obtuve %v", in, err)
		}
		if len(repo.saved) != 0 {
			t.Errorf("%+v: no debía guardar nada", in)
		}
	}
}

func TestSyncProfileReturnsTheRepositoryError(t *testing.T) {
	boom := errors.New("sin base")
	err := build(t, &fakeRepository{err: boom}, fakePermissions{}).SyncProfile.Execute(context.Background(), arojas)
	if !errors.Is(err, boom) {
		t.Fatalf("quiero el error del repositorio, obtuve %v", err)
	}
}

func TestGetMeReturnsKnownRolesAndPermissions(t *testing.T) {
	out, err := build(t, &fakeRepository{}, fakePermissions{}).GetMe.Execute(context.Background(),
		usecase.GetMeInput{Profile: arojas, Roles: []string{"qa", "offline_access"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Roles) != 1 || out.Roles[0] != "qa" || len(out.Permissions) != 1 {
		t.Errorf("roles %v y permisos %v: quiero solo los que Kaoru reconoce", out.Roles, out.Permissions)
	}
	if out.Profile.DisplayName != "arojas" || out.LogoutURL != "/oauth2/sign_out" || out.AccountURL == "" {
		t.Errorf("perfil o enlaces de sesión incompletos: %+v", out)
	}
}

func TestGetMeFailsWithoutIdentityOrPermissions(t *testing.T) {
	uc := build(t, &fakeRepository{}, fakePermissions{})
	if _, err := uc.GetMe.Execute(context.Background(), usecase.GetMeInput{}); !errors.Is(err, domain.ErrInvalidProfile) {
		t.Errorf("sin identidad: quiero ErrInvalidProfile, obtuve %v", err)
	}
	boom := errors.New("sin permisos")
	uc = build(t, &fakeRepository{}, fakePermissions{err: boom})
	if _, err := uc.GetMe.Execute(context.Background(), usecase.GetMeInput{Profile: arojas}); !errors.Is(err, boom) {
		t.Errorf("quiero el error de permisos, obtuve %v", err)
	}
}

func TestListUsersTrimsTheQueryAndCapsTheList(t *testing.T) {
	repo := &fakeRepository{saved: []domain.Profile{{Id: "sub-qa", Username: "arojas", DisplayName: "A. Rojas"}}}
	users, err := build(t, repo, fakePermissions{}).ListUsers.Execute(context.Background(), "  roj ")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || repo.listQuery != "roj" || repo.listLimit != 50 {
		t.Fatalf("usuarios %v, consulta %q, tope %d", users, repo.listQuery, repo.listLimit)
	}
}
