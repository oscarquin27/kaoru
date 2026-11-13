package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/report/application/port"
	"kaoru/internal/feature/report/application/usecase"
	"kaoru/internal/feature/report/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	reports   []domain.Report // en el orden en que se emitieron
	sequences map[string]int
	audit     []port.AuditRecord
	auditErr  error
	locked    []string
	inTx      bool
	evidence  domain.Content
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	reports, sequences, audit, locked := slices.Clone(w.reports), maps.Clone(w.sequences), slices.Clone(w.audit), slices.Clone(w.locked)
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.reports, w.sequences, w.audit, w.locked = reports, sequences, audit, locked
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

var packages = map[string]domain.PackageRef{
	"PKG-2026-0001": {Id: "p1", Code: "PKG-2026-0001", Title: "Mensajería 4.2"},
	"PKG-2026-0002": {Id: "p2", Code: "PKG-2026-0002", Title: "Paquete de noviembre"},
}

func (w *world) NextCode(_ context.Context, prefix string, year int) (shared.Code, error) {
	w.sequences[prefix]++
	return shared.NewCode(prefix, year, w.sequences[prefix]), nil
}

func (w *world) FindPackage(_ context.Context, code string) (domain.PackageRef, error) {
	pkg, ok := packages[code]
	if !ok {
		return domain.PackageRef{}, notFound
	}
	return pkg, nil
}

func (w *world) LockPackage(_ context.Context, packageId string) error {
	if !w.inTx {
		return errors.New("LockPackage fuera de una transacción")
	}
	w.locked = append(w.locked, packageId)
	return nil
}

func (w *world) Latest(_ context.Context, packageId string) (domain.Report, error) {
	for _, r := range slices.Backward(w.reports) {
		if r.Snapshot().Package.Id == packageId {
			return r, nil
		}
	}
	return domain.Report{}, notFound
}

func (w *world) Insert(_ context.Context, r domain.Report) error {
	s := r.Snapshot()
	if !w.inTx || !slices.Contains(w.locked, s.Package.Id) {
		return errors.New("Insert sin bloquear el PKG en una transacción")
	}
	for _, other := range w.reports {
		if o := other.Snapshot(); o.Package.Id == s.Package.Id && o.Version == s.Version {
			return axis.New(axis.DuplicateResource, "Ya existe.")
		}
	}
	w.reports = append(w.reports, r)
	return nil
}

// complete agrega lo que la base calcula: el nombre de quien emitió y el RPT que lo reemplazó.
func (w *world) complete(r domain.Report) domain.Report {
	s := r.Snapshot()
	s.IssuedBy.Name = "Usuario " + string(s.IssuedBy.Id)
	for _, other := range w.reports {
		if other.Snapshot().PreviousId == s.Id {
			s.SupersededBy = other.Snapshot().Code.String()
		}
	}
	return domain.Rehydrate(s)
}

func (w *world) FindByCode(_ context.Context, code shared.Code) (domain.Report, error) {
	for _, r := range w.reports {
		if r.Snapshot().Code == code {
			return w.complete(r), nil
		}
	}
	return domain.Report{}, notFound
}

func (w *world) List(_ context.Context, f port.Filter) ([]domain.Report, int, error) {
	var out []domain.Report
	for _, r := range slices.Backward(w.reports) {
		if f.PackageCode == "" || r.Snapshot().Package.Code == f.PackageCode {
			s := w.complete(r).Snapshot()
			s.Content = ""
			out = append(out, domain.Rehydrate(s))
		}
	}
	return out, len(out), nil
}

func (w *world) UserName(_ context.Context, id shared.UserId) (string, error) {
	return "Usuario " + string(id), nil
}

func (w *world) Gather(_ context.Context, code string) (domain.Content, error) {
	if !w.inTx || !slices.Contains(w.locked, packages[code].Id) {
		return domain.Content{}, errors.New("evidencia reunida sin bloquear el PKG")
	}
	c := w.evidence
	c.Package.Code = code
	return c, nil
}

var granted = map[string][]string{"qa": {usecase.PermissionIssue}, "qa-admin": {usecase.PermissionIssue}}

func (w *world) Require(_ context.Context, roles []string, permission string) error {
	for _, role := range roles {
		if slices.Contains(granted[role], permission) {
			return nil
		}
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

func (w *world) Record(_ context.Context, r port.AuditRecord) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	if !w.inTx {
		return errors.New("auditoría fuera de una transacción")
	}
	w.audit = append(w.audit, r)
	return nil
}

type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	t0      = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	qa      = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	ctx     = context.Background()
)

func setup(t *testing.T) (*usecase.UseCases, *world, *fixedClock) {
	t.Helper()
	w := &world{sequences: map[string]int{}}
	w.evidence = domain.Content{Package: domain.PackageInfo{Title: "Mensajería 4.2", SystemName: "SIMF", Route: "high"}}
	clock := &fixedClock{now: t0}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Evidence: w, Permissions: w, AuditTrail: w, UnitOfWork: w, Clock: clock, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w, clock
}

func fieldsOf(err error) []string {
	var ge *axis.GeneralError
	if !errors.As(err, &ge) {
		return nil
	}
	var out []string
	for _, f := range ge.Fields() {
		out = append(out, f.Field+":"+f.Code)
	}
	return out
}

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestIssueReport(t *testing.T) {
	uc, w, clock := setup(t)
	issue := func(actor shared.Actor, pkg string) (domain.Snapshot, error) {
		return uc.IssueReport.Execute(ctx, usecase.IssueReportInput{Actor: actor, PackageCode: pkg})
	}

	if _, err := issue(manager, "PKG-2026-0001"); !axis.IsCode(err, axis.PermissionDenied) {
		t.Errorf("change management no emite informes: %v", err)
	}
	for pkg, want := range map[string]string{"": "package:required", "PKG-2026-9999": "package:invalid"} {
		if _, err := issue(qa, pkg); !slices.Equal(fieldsOf(err), []string{want}) {
			t.Errorf("PKG %q: %v", pkg, err)
		}
	}

	first, err := issue(qa, "PKG-2026-0001")
	if err != nil {
		t.Fatal(err)
	}
	if first.Code.String() != "RPT-2026-0001" || first.Version != 1 || first.PreviousCode != "" || first.IssuedBy.Name != "Usuario u2" {
		t.Errorf("primer informe = %+v", first)
	}
	if first.Checksum != domain.Checksum(first.Content) {
		t.Errorf("la huella no es el SHA-256 del contenido")
	}
	for _, want := range []string{"# Informe de conformidad RPT-2026-0001", "| Emitido por | Usuario u2 |", "| Emitido el | 2026-11-12 15:00 UTC |", "| PKG | PKG-2026-0001 — Mensajería 4.2 |"} {
		if !strings.Contains(first.Content, want) {
			t.Errorf("falta %q en el contenido", want)
		}
	}
	if len(w.audit) != 1 || w.audit[0].Code != "RPT-2026-0001" || w.audit[0].NewValue != first.Checksum {
		t.Errorf("auditoría = %+v", w.audit)
	}

	// Algo cambió: la versión siguiente referencia a la vigente y la deja reemplazada.
	clock.now = t0.Add(time.Hour)
	w.evidence.Package.StateName = "Verificado"
	second, err := issue(qa, "PKG-2026-0001")
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 || second.PreviousCode != "RPT-2026-0001" || !strings.Contains(second.Content, "| Reemplaza a | RPT-2026-0001 |") {
		t.Errorf("segundo informe = %+v", second)
	}
	if second.Checksum == first.Checksum {
		t.Errorf("dos contenidos distintos con la misma huella")
	}
	again, err := uc.GetReport.Execute(ctx, usecase.GetReportInput{Code: "RPT-2026-0001"})
	if err != nil || again.SupersededBy != "RPT-2026-0002" || again.Content != first.Content {
		t.Errorf("el primer informe cambió o no se ve reemplazado: %+v %v", again, err)
	}

	// Otro PKG empieza su propia cadena.
	other, err := issue(qa, "PKG-2026-0002")
	if err != nil || other.Version != 1 {
		t.Errorf("informe de otro PKG = %+v %v", other, err)
	}
}

func TestIssueReportSavesNothingIfTheAuditFails(t *testing.T) {
	uc, w, _ := setup(t)
	w.auditErr = errors.New("auditoría caída")
	if _, err := uc.IssueReport.Execute(ctx, usecase.IssueReportInput{Actor: qa, PackageCode: "PKG-2026-0001"}); err == nil {
		t.Fatal("debía fallar")
	}
	if len(w.reports) != 0 || w.sequences["RPT"] != 0 {
		t.Errorf("quedó algo guardado: %d informes", len(w.reports))
	}
}

func TestGetAndListReports(t *testing.T) {
	uc, _, _ := setup(t)
	for _, pkg := range []string{"PKG-2026-0001", "PKG-2026-0002", "PKG-2026-0001"} {
		if _, err := uc.IssueReport.Execute(ctx, usecase.IssueReportInput{Actor: qa, PackageCode: pkg}); err != nil {
			t.Fatal(err)
		}
	}
	for _, code := range []string{"RPT-2026-9999", "nada"} {
		if _, err := uc.GetReport.Execute(ctx, usecase.GetReportInput{Code: code}); !errors.Is(err, usecase.ErrReportNotFound) {
			t.Errorf("%s: %v", code, err)
		}
	}
	page, err := uc.ListReports.Execute(ctx, usecase.ListReportsInput{PackageCode: "PKG-2026-0001"})
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, r := range page.Reports {
		codes = append(codes, r.Code.String())
		if r.Content != "" {
			t.Errorf("la lista trae el contenido de %s", r.Code)
		}
	}
	if !slices.Equal(codes, []string{"RPT-2026-0003", "RPT-2026-0001"}) || page.Total != 2 {
		t.Errorf("lista = %v (%d)", codes, page.Total)
	}
}
