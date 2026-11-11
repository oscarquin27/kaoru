package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/risk/application/port"
	"kaoru/internal/feature/risk/application/usecase"
	"kaoru/internal/feature/risk/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	risks    map[string]domain.Assessment // por id de REQ
	audit    []port.AuditRecord
	sequence int
	auditErr error
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	risks, audit, sequence := map[string]domain.Assessment{}, slices.Clone(w.audit), w.sequence
	for k, v := range w.risks {
		risks[k] = v
	}
	if err := fn(ctx); err != nil {
		w.risks, w.audit, w.sequence = risks, audit, sequence
		return err
	}
	return nil
}

func (w *world) NextCode(_ context.Context, year int) (shared.Code, error) {
	w.sequence++
	return shared.NewCode(shared.PrefixRisk, year, w.sequence), nil
}

func (w *world) FindByRequirement(_ context.Context, requirementId string) (domain.Assessment, error) {
	a, ok := w.risks[requirementId]
	if !ok {
		return domain.Assessment{}, axis.New(axis.NotFoundResource, "No existe.")
	}
	return a, nil
}

func (w *world) Insert(_ context.Context, a domain.Assessment) error {
	w.risks[a.Snapshot().RequirementId] = a
	return nil
}

func (w *world) Update(_ context.Context, a domain.Assessment) error {
	w.risks[a.Snapshot().RequirementId] = a
	return nil
}

func (w *world) Record(_ context.Context, r port.AuditRecord) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	w.audit = append(w.audit, r)
	return nil
}

type fakeRequirements struct{}

func (fakeRequirements) Find(_ context.Context, code string) (port.RequirementRef, error) {
	if code != "REQ-2026-0042" {
		return port.RequirementRef{}, axis.New(axis.NotFoundResource, "El REQ no existe.")
	}
	return port.RequirementRef{Id: "r1", Code: code}, nil
}

type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, _ string) error {
	if slices.Contains(roles, "qa") {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC) }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	qa  = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	dev = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	ctx = context.Background()
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{risks: map[string]domain.Assessment{}}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Requirements: fakeRequirements{}, Permissions: fakePermissions{}, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

func assess(uc *usecase.UseCases, actor shared.Actor, code, level string, chars ...string) (domain.Snapshot, error) {
	return uc.AssessRisk.Execute(ctx, usecase.AssessRiskInput{
		Actor: actor, RequirementCode: code, Level: level, Justification: "Toca saldos", Characteristics: chars,
	})
}

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestAssessRisk(t *testing.T) {
	uc, w := setup(t)

	first, err := assess(uc, qa, "REQ-2026-0042", "medium")
	if err != nil {
		t.Fatal(err)
	}
	if first.Code.String() != "RISK-2026-0001" || first.Level != domain.LevelMedium || first.AssessedBy != "u2" ||
		len(w.audit) != 1 || w.audit[0].Action != port.AuditCreate || w.audit[0].NewValue != "medium" {
		t.Fatalf("primera evaluación: %+v / %+v", first, w.audit)
	}

	// Reevaluar actualiza el mismo RISK y audita cada campo que cambió.
	second, err := assess(uc, qa, "REQ-2026-0042", "high", "security")
	if err != nil {
		t.Fatal(err)
	}
	if second.Code != first.Code || second.Level != domain.LevelHigh || w.sequence != 1 || len(w.risks) != 1 {
		t.Errorf("reevaluar no crea otro RISK: %+v", second)
	}
	var fields []string
	for _, a := range w.audit[1:] {
		fields = append(fields, a.Field+":"+a.OldValue+"→"+a.NewValue)
	}
	if !slices.Equal(fields, []string{"level:medium→high", "characteristics:→security"}) {
		t.Errorf("auditoría de la reevaluación: %v", fields)
	}

	// Reevaluar con lo mismo no escribe ni audita.
	if _, err := assess(uc, qa, "REQ-2026-0042", "high", "security"); err != nil || len(w.audit) != 3 {
		t.Errorf("una reevaluación sin cambios no se audita: %v, %d registros", err, len(w.audit))
	}
}

func TestAssessRiskErrors(t *testing.T) {
	cases := []struct {
		name     string
		actor    shared.Actor
		code     string
		level    string
		wantCode axis.Code
	}{
		{"sin permiso", dev, "REQ-2026-0042", "low", axis.PermissionDenied},
		{"REQ inexistente", qa, "REQ-2026-0099", "low", axis.NotFoundResource},
		{"nivel inválido", qa, "REQ-2026-0042", "alto", axis.InvalidRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			if _, err := assess(uc, tc.actor, tc.code, tc.level); !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if len(w.risks) != 0 || len(w.audit) != 0 || w.sequence != 0 {
				t.Error("una evaluación rechazada no debe dejar nada, ni gastar un código")
			}
		})
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")
	if _, err := assess(uc, qa, "REQ-2026-0042", "low"); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Fatalf("crear: %v", err)
	}
	if len(w.risks) != 0 || w.sequence != 0 {
		t.Error("si la auditoría falla, el RISK no se guarda")
	}
	w.auditErr = nil
	if _, err := assess(uc, qa, "REQ-2026-0042", "low"); err != nil {
		t.Fatal(err)
	}
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")
	if _, err := assess(uc, qa, "REQ-2026-0042", "high"); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Fatalf("reevaluar: %v", err)
	}
	if w.risks["r1"].Snapshot().Level != domain.LevelLow {
		t.Error("si la auditoría falla, la reevaluación no se guarda")
	}
}

func TestGetRisk(t *testing.T) {
	uc, _ := setup(t)
	if _, err := uc.GetRisk.Execute(ctx, usecase.GetRiskInput{RequirementCode: "REQ-2026-0042"}); !errors.Is(err, usecase.ErrRiskNotFound) {
		t.Errorf("sin evaluar: %v", err)
	}
	if _, err := assess(uc, qa, "REQ-2026-0042", "low"); err != nil {
		t.Fatal(err)
	}
	out, err := uc.GetRisk.Execute(ctx, usecase.GetRiskInput{RequirementCode: "REQ-2026-0042"})
	if err != nil || out.Level != domain.LevelLow {
		t.Errorf("GetRisk = %+v, %v", out, err)
	}
	if _, err := uc.GetRisk.Execute(ctx, usecase.GetRiskInput{RequirementCode: "REQ-2026-0099"}); !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("REQ inexistente: %v", err)
	}
}
