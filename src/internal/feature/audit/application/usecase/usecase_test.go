package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/audit/application/port"
	"kaoru/internal/feature/audit/application/usecase"
	"kaoru/internal/feature/audit/domain"
	shared "kaoru/internal/shared/domain"
)

// fakeRepo guarda la cadena en memoria.
type fakeRepo struct {
	entries   []domain.Entry
	locked    int
	insertErr error
}

func (r *fakeRepo) LockChain(context.Context) error { r.locked++; return nil }

func (r *fakeRepo) LastHash(context.Context) (string, error) {
	if len(r.entries) == 0 {
		return "", nil
	}
	return r.entries[len(r.entries)-1].Hash, nil
}

func (r *fakeRepo) Insert(_ context.Context, e domain.Entry) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	e.Seq = int64(len(r.entries) + 1)
	r.entries = append(r.entries, e)
	return nil
}

func (r *fakeRepo) List(context.Context, port.Filter) ([]port.EntryView, int, error) {
	views := make([]port.EntryView, len(r.entries))
	for i, e := range r.entries {
		views[i] = port.EntryView{Entry: e, ActorName: "Lab"}
	}
	return views, len(views), nil
}

func (r *fakeRepo) ChainAfter(_ context.Context, afterSeq int64, limit int) ([]domain.Entry, error) {
	var out []domain.Entry
	for _, e := range r.entries {
		if e.Seq > afterSeq && len(out) < limit {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *fakeRepo) LastTransitions(_ context.Context, ids []string) (map[string]time.Time, error) {
	out := map[string]time.Time{}
	for _, e := range r.entries {
		if e.Action == domain.ActionTransition && slices.Contains(ids, e.EntityId) {
			out[e.EntityId] = e.OccurredAt
		}
	}
	return out, nil
}

type fakePermissions struct{ allowed bool }

func (p fakePermissions) Require(context.Context, []string, string) error {
	if p.allowed {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

type fakeUoW struct{}

func (fakeUoW) Within(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC) }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("01930000-0000-7000-8000-%012d", s.n), nil
}

func newUseCases(t *testing.T, repo *fakeRepo, allowed bool) *usecase.UseCases {
	t.Helper()
	uc, err := usecase.New(usecase.Deps{
		Repository: repo, Permissions: fakePermissions{allowed: allowed},
		UnitOfWork: fakeUoW{}, Clock: fixedClock{}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

func record(t *testing.T, uc *usecase.UseCases, field, value string) {
	t.Helper()
	err := uc.Record.Execute(context.Background(), usecase.RecordInput{
		ActorId: "u1", EntityType: "change", EntityId: "01930000-0000-7000-8000-00000000aaaa",
		EntityCode: "CHG-2026-0001", Action: domain.ActionUpdate, Field: field, NewValue: value,
	})
	if err != nil {
		t.Fatal(err)
	}
}

var auditor = shared.Actor{Id: "u1", Roles: []string{"auditor"}}

func TestRecordChainsEntries(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUseCases(t, repo, true)
	record(t, uc, "title", "uno")
	record(t, uc, "title", "dos")
	record(t, uc, "title", "tres")

	if repo.locked != 3 {
		t.Errorf("cada inserción debe bloquear la cadena; bloqueos = %d", repo.locked)
	}
	if repo.entries[0].PrevHash != "" {
		t.Error("el primer registro no debe tener hash anterior")
	}
	for i := 1; i < len(repo.entries); i++ {
		if repo.entries[i].PrevHash != repo.entries[i-1].Hash {
			t.Errorf("el registro %d no apunta al hash del anterior", i+1)
		}
	}
	out, err := uc.VerifyChain.Execute(context.Background(), usecase.VerifyChainInput{Actor: auditor})
	if err != nil || !out.Intact || out.Entries != 3 || out.FirstBrokenSeq != 0 {
		t.Errorf("cadena recién escrita debe estar íntegra: %+v, err = %v", out, err)
	}
}

func TestRecordRejectsIncompleteEntry(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUseCases(t, repo, true)
	cases := map[string]usecase.RecordInput{
		"sin actor":         {EntityType: "change", EntityId: "x", Action: domain.ActionCreate},
		"sin entidad":       {ActorId: "u1", EntityType: "change", Action: domain.ActionCreate},
		"acción desconocid": {ActorId: "u1", EntityType: "change", EntityId: "x", Action: "borrar"},
	}
	for name, in := range cases {
		if err := uc.Record.Execute(context.Background(), in); !errors.Is(err, domain.ErrInvalidEntry) {
			t.Errorf("%s: esperaba ErrInvalidEntry, obtuve %v", name, err)
		}
	}
	if len(repo.entries) != 0 {
		t.Error("un registro incompleto no debe guardarse")
	}
}

func TestRecordPropagatesInsertError(t *testing.T) {
	repo := &fakeRepo{insertErr: axis.New(axis.ServiceUnavailable, "sin base")}
	uc := newUseCases(t, repo, true)
	err := uc.Record.Execute(context.Background(), usecase.RecordInput{
		ActorId: "u1", EntityType: "change", EntityId: "x", Action: domain.ActionCreate,
	})
	if !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Fatalf("la operación debe fallar si la auditoría falla; obtuve %v", err)
	}
}

func TestVerifyChainDetectsTampering(t *testing.T) {
	tamper := map[string]func(e *domain.Entry){
		"valor cambiado":    func(e *domain.Entry) { e.NewValue = "alterado" },
		"actor cambiado":    func(e *domain.Entry) { e.ActorId = "otro" },
		"fecha cambiada":    func(e *domain.Entry) { e.OccurredAt = e.OccurredAt.Add(time.Second) },
		"motivo agregado":   func(e *domain.Entry) { e.Reason = "inventado" },
		"hash recalculado":  func(e *domain.Entry) { *e = e.Chained("0000") },
		"enlace reapuntado": func(e *domain.Entry) { e.PrevHash = "" },
	}
	for name, alter := range tamper {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepo{}
			uc := newUseCases(t, repo, true)
			record(t, uc, "title", "uno")
			record(t, uc, "title", "dos")
			record(t, uc, "title", "tres")
			alter(&repo.entries[1])

			out, err := uc.VerifyChain.Execute(context.Background(), usecase.VerifyChainInput{Actor: auditor})
			if err != nil {
				t.Fatal(err)
			}
			if out.Intact || out.FirstBrokenSeq != 2 || out.Entries != 3 {
				t.Errorf("debía detectar la alteración en el registro 2: %+v", out)
			}
		})
	}
}

func TestVerifyChainDetectsDeletion(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUseCases(t, repo, true)
	record(t, uc, "title", "uno")
	record(t, uc, "title", "dos")
	record(t, uc, "title", "tres")
	repo.entries = append(repo.entries[:1], repo.entries[2:]...)

	out, err := uc.VerifyChain.Execute(context.Background(), usecase.VerifyChainInput{Actor: auditor})
	if err != nil || out.Intact || out.FirstBrokenSeq != 3 {
		t.Errorf("debía detectar el registro borrado: %+v, err = %v", out, err)
	}
}

func TestReadingRequiresPermission(t *testing.T) {
	uc := newUseCases(t, &fakeRepo{}, false)
	if _, err := uc.ListEntries.Execute(context.Background(), usecase.ListEntriesInput{Actor: auditor}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("ListEntries sin permiso: %v", err)
	}
	if _, err := uc.VerifyChain.Execute(context.Background(), usecase.VerifyChainInput{Actor: auditor}); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("VerifyChain sin permiso: %v", err)
	}
}
