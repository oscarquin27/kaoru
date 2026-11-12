package usecase_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/application/usecase"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// catalog es el catálogo que ve el doble de port.Components.
var catalog = map[string]struct {
	ref     domain.ComponentRef
	system  string
	retired bool
}{
	"COMP-2026-0001": {domain.ComponentRef{Id: "k1", Code: "COMP-2026-0001", Name: "svc-saldos", Kind: "microservice"}, "core", false},
	"COMP-2026-0002": {domain.ComponentRef{Id: "k2", Code: "COMP-2026-0002", Name: "lib-firmas", Kind: "library"}, "core", false},
	"COMP-2026-0003": {domain.ComponentRef{Id: "k3", Code: "COMP-2026-0003", Name: "job-viejo", Kind: "script"}, "core", true},
	"COMP-2026-0004": {domain.ComponentRef{Id: "k4", Code: "COMP-2026-0004", Name: "portal", Kind: "microservice"}, "web", false},
}

// Resolve hace de port.Components: rechaza lo que no existe, lo de otro sistema y lo retirado.
func (w *world) Resolve(_ context.Context, systemKey string, codes []string) ([]domain.ComponentRef, error) {
	refs := make([]domain.ComponentRef, 0, len(codes))
	for _, code := range codes {
		c, ok := catalog[code]
		if !ok || c.system != systemKey || c.retired {
			return nil, axis.Invalid(axis.InvalidField("components", "El componente "+code+" no vale."))
		}
		refs = append(refs, c.ref)
	}
	return refs, nil
}

// LockTestCase no hace nada en memoria: el doble ya ejecuta una transacción a la vez.
func (w *world) LockTestCase(ctx context.Context, code shared.Code) error {
	_, err := w.FindTestCase(ctx, code)
	return err
}

func (w *world) ListTestCaseComponents(_ context.Context, testCaseId string) ([]domain.ComponentRef, error) {
	refs := []domain.ComponentRef{}
	for _, code := range w.components[testCaseId] {
		refs = append(refs, catalog[code].ref)
	}
	slices.SortFunc(refs, func(a, b domain.ComponentRef) int { return strings.Compare(a.Name, b.Name) })
	return refs, nil
}

func (w *world) ReplaceTestCaseComponents(_ context.Context, testCaseId string, componentIds []string) error {
	codes := []string{}
	for code, c := range catalog {
		if slices.Contains(componentIds, c.ref.Id) {
			codes = append(codes, code)
		}
	}
	w.components[testCaseId] = codes
	return nil
}

func namesOf(refs []domain.ComponentRef) string {
	names := make([]string, len(refs))
	for i, r := range refs {
		names[i] = r.Name
	}
	return strings.Join(names, ",")
}

func setComponents(uc *usecase.UseCases, testCaseCode string, codes ...string) ([]domain.ComponentRef, error) {
	if codes == nil {
		codes = []string{}
	}
	return uc.SetTestCaseComponents.Execute(ctx, usecase.SetTestCaseComponentsInput{Actor: qa, Code: testCaseCode, Components: codes})
}

func TestSetTestCaseComponents(t *testing.T) {
	uc, w := setup(t)
	code := saldo(t, uc)
	id := w.testCases[code].Snapshot().Id

	out, err := setComponents(uc, code, "COMP-2026-0001", "COMP-2026-0002")
	if err != nil {
		t.Fatal(err)
	}
	if namesOf(out) != "lib-firmas,svc-saldos" || len(w.components[id]) != 2 {
		t.Errorf("componentes = %+v, guardados = %v", out, w.components[id])
	}
	last := w.audit[len(w.audit)-1]
	if len(w.audit) != 3 || last.Action != port.AuditUpdate || last.EntityType != port.EntityTestCase || last.EntityId != id ||
		last.Code.String() != code || last.Actor != qa.Id || last.Field != "components" ||
		last.OldValue != "" || last.NewValue != "COMP-2026-0001, COMP-2026-0002" {
		t.Errorf("auditoría inesperada: %+v", w.audit[2:])
	}

	// Declarar lo mismo, en otro orden, no escribe ni audita.
	if out, err = setComponents(uc, code, "COMP-2026-0002", "COMP-2026-0001"); err != nil || namesOf(out) != "lib-firmas,svc-saldos" || len(w.audit) != 3 {
		t.Errorf("sin cambios: %+v, %v, auditoría %d", out, err, len(w.audit))
	}

	// Reemplaza la lista: lo que no viene se quita.
	if out, err = setComponents(uc, code, "COMP-2026-0002"); err != nil || namesOf(out) != "lib-firmas" {
		t.Fatalf("reemplazo: %+v, %v", out, err)
	}
	if last = w.audit[len(w.audit)-1]; last.OldValue != "COMP-2026-0001, COMP-2026-0002" || last.NewValue != "COMP-2026-0002" {
		t.Errorf("auditoría del reemplazo: %+v", last)
	}

	// Una lista vacía quita todos.
	if out, err = setComponents(uc, code); err != nil || len(out) != 0 || len(w.components[id]) != 0 {
		t.Errorf("vaciar: %+v, %v", out, err)
	}
	if last = w.audit[len(w.audit)-1]; len(w.audit) != 5 || last.OldValue != "COMP-2026-0002" || last.NewValue != "" {
		t.Errorf("auditoría al vaciar: %+v", last)
	}
}

func TestSetTestCaseComponentsErrors(t *testing.T) {
	const tc1 = "TC-2026-0001"
	valid := []string{"COMP-2026-0001"}
	cases := []struct {
		name      string
		in        usecase.SetTestCaseComponentsInput
		wantCode  axis.Code
		wantField string
	}{
		{"sin permiso", usecase.SetTestCaseComponentsInput{Actor: dev, Code: tc1, Components: valid}, axis.PermissionDenied, ""},
		{"sin lista", usecase.SetTestCaseComponentsInput{Actor: qa, Code: tc1}, axis.InvalidRequest, "components:required"},
		{"TC inexistente", usecase.SetTestCaseComponentsInput{Actor: qa, Code: "TC-2026-0099", Components: valid}, axis.NotFoundResource, ""},
		{"código de TC mal formado", usecase.SetTestCaseComponentsInput{Actor: qa, Code: "x", Components: valid}, axis.NotFoundResource, ""},
		{"componente inexistente", usecase.SetTestCaseComponentsInput{Actor: qa, Code: tc1, Components: []string{"COMP-2026-0099"}}, axis.InvalidRequest, "components:invalid"},
		{"componente de otro sistema", usecase.SetTestCaseComponentsInput{Actor: qa, Code: tc1, Components: []string{"COMP-2026-0001", "COMP-2026-0004"}}, axis.InvalidRequest, "components:invalid"},
		{"componente retirado", usecase.SetTestCaseComponentsInput{Actor: qa, Code: tc1, Components: []string{"COMP-2026-0003"}}, axis.InvalidRequest, "components:invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			code := saldo(t, uc)
			if _, err := setComponents(uc, code, "COMP-2026-0002"); err != nil {
				t.Fatal(err)
			}
			_, err := uc.SetTestCaseComponents.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if tc.wantField != "" && strings.Join(fieldsOf(err), " ") != tc.wantField {
				t.Errorf("campos = %v, quiero %s", fieldsOf(err), tc.wantField)
			}
			id := w.testCases[code].Snapshot().Id
			if !slices.Equal(w.components[id], []string{"COMP-2026-0002"}) || len(w.audit) != 3 {
				t.Error("una declaración rechazada no debe cambiar los componentes ni auditarse")
			}
		})
	}
}

func TestListTestCaseComponents(t *testing.T) {
	uc, _ := setup(t)
	code := saldo(t, uc)
	out, err := uc.ListTestCaseComponents.Execute(ctx, usecase.ListTestCaseComponentsInput{Code: code})
	if err != nil || out == nil || len(out) != 0 {
		t.Errorf("lista vacía = %#v, %v", out, err)
	}
	if _, err := setComponents(uc, code, "COMP-2026-0001", "COMP-2026-0002"); err != nil {
		t.Fatal(err)
	}
	out, err = uc.ListTestCaseComponents.Execute(ctx, usecase.ListTestCaseComponentsInput{Code: code})
	if err != nil || namesOf(out) != "lib-firmas,svc-saldos" || out[0].Code != "COMP-2026-0002" || out[0].Kind != "library" {
		t.Errorf("lista = %+v, %v", out, err)
	}
	for _, missing := range []string{"TC-2026-0099", "x"} {
		if _, err := uc.ListTestCaseComponents.Execute(ctx, usecase.ListTestCaseComponentsInput{Code: missing}); !errors.Is(err, usecase.ErrTestCaseNotFound) {
			t.Errorf("lista de %q: %v", missing, err)
		}
	}
}

func TestComponentsAreNotSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	code := saldo(t, uc)
	if _, err := setComponents(uc, code, "COMP-2026-0002"); err != nil {
		t.Fatal(err)
	}
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")
	if _, err := setComponents(uc, code, "COMP-2026-0001"); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Fatalf("esperaba SERVICE_UNAVAILABLE, obtuve %v", err)
	}
	id := w.testCases[code].Snapshot().Id
	if !slices.Equal(w.components[id], []string{"COMP-2026-0002"}) || len(w.audit) != 3 {
		t.Error("si la auditoría falla, los componentes deben quedar como estaban")
	}
}

// saldo define un TC del sistema core, con su UC, y devuelve su código. Deja dos hechos en la auditoría.
func saldo(t *testing.T, uc *usecase.UseCases) string {
	t.Helper()
	return testCase(t, uc, "Saldo", false, functionalCase(t, uc, "REQ-2026-0042", "Consultar saldo"))
}
