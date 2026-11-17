package usecase_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/system/application/port"
	"kaoru/internal/feature/system/application/usecase"
	"kaoru/internal/feature/system/domain"
	shared "kaoru/internal/shared/domain"
)

var t0 = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	systems  map[string]domain.System // por clave
	audit    []port.AuditRecord
	auditErr error
	locked   []string
	inTx     bool
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	systems, audit := maps.Clone(w.systems), slices.Clone(w.audit)
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.systems, w.audit = systems, audit
		return err
	}
	return nil
}

func (w *world) Insert(_ context.Context, s domain.System) error {
	if !w.inTx {
		return errors.New("Insert fuera de una transacción")
	}
	if _, taken := w.systems[s.Snapshot().Key]; taken {
		return axis.New(axis.DuplicateResource, "Ya existe.")
	}
	w.systems[s.Snapshot().Key] = s
	return nil
}

func (w *world) Update(_ context.Context, s domain.System) error {
	if !w.inTx || !slices.Contains(w.locked, s.Snapshot().Key) {
		return errors.New("Update sin bloquear el sistema en una transacción")
	}
	w.systems[s.Snapshot().Key] = s
	return nil
}

func (w *world) Lock(_ context.Context, key string) (domain.System, error) {
	s, ok := w.systems[key]
	if !ok {
		return domain.System{}, axis.New(axis.NotFoundResource, "No existe.")
	}
	w.locked = append(w.locked, key)
	return s, nil
}

func (w *world) Record(_ context.Context, r port.AuditRecord) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	w.audit = append(w.audit, r)
	return nil
}

// fakePermissions: solo qa-admin configura catálogos.
type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	if permission == usecase.PermissionConfigure && slices.Contains(roles, "qa-admin") {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return t0 }

type seqIds struct{ n int }

func (s *seqIds) NewId() (string, error) {
	s.n++
	return []string{"", "id-1", "id-2", "id-3"}[s.n], nil
}

var (
	admin = shared.Actor{Id: "u-admin", Roles: []string{"qa-admin"}}
	qa    = shared.Actor{Id: "u-qa", Roles: []string{"qa"}}
)

func setup(t *testing.T) (*world, *usecase.UseCases) {
	t.Helper()
	w := &world{systems: map[string]domain.System{}}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Permissions: fakePermissions{}, AuditTrail: w, UnitOfWork: w,
		Clock: fixedClock{}, IdGenerator: &seqIds{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return w, uc
}

func create(t *testing.T, uc *usecase.UseCases, key string) domain.Snapshot {
	t.Helper()
	s, err := uc.CreateSystem.Execute(context.Background(), usecase.CreateSystemInput{
		Actor: admin, Key: key, Name: "SysAF", DefaultRoute: "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCreateSystemSavesAndAudits(t *testing.T) {
	w, uc := setup(t)
	s := create(t, uc, "sysaf")
	if s.Key != "SYSAF" || s.Id != "id-1" {
		t.Fatalf("sistema inesperado: %+v", s)
	}
	if _, ok := w.systems["SYSAF"]; !ok {
		t.Fatal("el sistema no se guardó")
	}
	want := port.AuditRecord{Actor: "u-admin", EntityId: "id-1", Key: "SYSAF", Action: port.AuditCreate, NewValue: "SysAF"}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Fatalf("auditoría inesperada: %+v", w.audit)
	}
}

func TestCreateSystemNeedsConfigurePermission(t *testing.T) {
	w, uc := setup(t)
	_, err := uc.CreateSystem.Execute(context.Background(), usecase.CreateSystemInput{
		Actor: qa, Key: "SYSAF", Name: "SysAF", DefaultRoute: "high",
	})
	if !axis.IsCode(err, axis.PermissionDenied) || len(w.systems) != 0 {
		t.Fatalf("qa no configura catálogos: %v", err)
	}
}

func TestCreateSystemRejectsRepeatedKey(t *testing.T) {
	w, uc := setup(t)
	create(t, uc, "SYSAF")
	_, err := uc.CreateSystem.Execute(context.Background(), usecase.CreateSystemInput{
		Actor: admin, Key: " sysaf", Name: "Otro", DefaultRoute: "light",
	})
	var ge *axis.GeneralError
	if !errors.As(err, &ge) || ge.Code() != axis.InvalidRequest || len(ge.Fields()) != 1 || ge.Fields()[0].Field != "key" {
		t.Fatalf("la clave repetida es error del campo key: %v", err)
	}
	if len(w.audit) != 1 || w.systems["SYSAF"].Snapshot().Name != "SysAF" {
		t.Fatalf("el intento fallido no cambia nada: %+v", w.audit)
	}
}

func TestCreateSystemRollsBackIfAuditFails(t *testing.T) {
	w, uc := setup(t)
	w.auditErr = errors.New("auditoría caída")
	_, err := uc.CreateSystem.Execute(context.Background(), usecase.CreateSystemInput{
		Actor: admin, Key: "SYSAF", Name: "SysAF", DefaultRoute: "high",
	})
	if err == nil || len(w.systems) != 0 {
		t.Fatalf("sin auditoría no se guarda: %v %v", err, w.systems)
	}
}

func TestUpdateSystemAuditsEachChangedField(t *testing.T) {
	w, uc := setup(t)
	create(t, uc, "SYSAF")
	name, route := "SysAF 2", "light"
	s, err := uc.UpdateSystem.Execute(context.Background(), usecase.UpdateSystemInput{
		Actor: admin, Key: "sysaf", Edit: domain.Edit{Name: &name, DefaultRoute: &route},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "SysAF 2" || s.DefaultRoute != domain.RouteLight || w.systems["SYSAF"].Snapshot().Name != "SysAF 2" {
		t.Fatalf("sistema inesperado: %+v", s)
	}
	if len(w.audit) != 3 || w.audit[1].Field != "name" || w.audit[2].Field != "defaultRoute" ||
		w.audit[2].OldValue != "high" || w.audit[2].NewValue != "light" {
		t.Fatalf("auditoría inesperada: %+v", w.audit)
	}

	// Sin cambios no se escribe ni se audita.
	if _, err := uc.UpdateSystem.Execute(context.Background(), usecase.UpdateSystemInput{
		Actor: admin, Key: "SYSAF", Edit: domain.Edit{Name: &name},
	}); err != nil || len(w.audit) != 3 {
		t.Fatalf("repetir el valor no audita: %v %+v", err, w.audit)
	}
}

func TestUpdateSystemErrors(t *testing.T) {
	_, uc := setup(t)
	create(t, uc, "SYSAF")
	name := "Otro"
	if _, err := uc.UpdateSystem.Execute(context.Background(), usecase.UpdateSystemInput{
		Actor: qa, Key: "SYSAF", Edit: domain.Edit{Name: &name},
	}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Fatalf("qa no edita sistemas: %v", err)
	}
	if _, err := uc.UpdateSystem.Execute(context.Background(), usecase.UpdateSystemInput{
		Actor: admin, Key: "NOEXISTE", Edit: domain.Edit{Name: &name},
	}); !errors.Is(err, usecase.ErrSystemNotFound) {
		t.Fatalf("una clave que no existe es NOT_FOUND: %v", err)
	}
}

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("faltan dependencias y no falló")
	}
}
