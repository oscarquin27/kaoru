package usecase_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/requirement/domain"
)

// catalog es el catálogo que ve el doble de port.Components.
var catalog = map[string]struct {
	ref     domain.ComponentRef
	system  string
	retired bool
}{
	"COMP-2026-0001": {domain.ComponentRef{Id: "k1", Code: "COMP-2026-0001", Name: "svc-pagos", Kind: "microservice"}, "SYPAGO", false},
	"COMP-2026-0002": {domain.ComponentRef{Id: "k2", Code: "COMP-2026-0002", Name: "lib-firmas", Kind: "library"}, "SYPAGO", false},
	"COMP-2026-0003": {domain.ComponentRef{Id: "k3", Code: "COMP-2026-0003", Name: "job-viejo", Kind: "script"}, "SYPAGO", true},
	"COMP-2026-0004": {domain.ComponentRef{Id: "k4", Code: "COMP-2026-0004", Name: "svc-simf", Kind: "microservice"}, "SIMF", false},
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

func (w *world) ListComponents(_ context.Context, requirementId string) ([]domain.ComponentRef, error) {
	refs := []domain.ComponentRef{}
	for _, code := range w.components[requirementId] {
		refs = append(refs, catalog[code].ref)
	}
	slices.SortFunc(refs, func(a, b domain.ComponentRef) int { return strings.Compare(a.Name, b.Name) })
	return refs, nil
}

func (w *world) ReplaceComponents(_ context.Context, requirementId string, componentIds []string) error {
	codes := []string{}
	for code, c := range catalog {
		if slices.Contains(componentIds, c.ref.Id) {
			codes = append(codes, code)
		}
	}
	w.components[requirementId] = codes
	return nil
}

func namesOf(refs []domain.ComponentRef) string {
	names := make([]string, len(refs))
	for i, r := range refs {
		names[i] = r.Name
	}
	return strings.Join(names, ",")
}

func setComponents(uc *usecase.UseCases, codes ...string) ([]domain.ComponentRef, error) {
	if codes == nil {
		codes = []string{}
	}
	return uc.SetRequirementComponents.Execute(ctx, usecase.SetRequirementComponentsInput{Actor: tester, Code: req1, Components: codes})
}

func TestSetRequirementComponents(t *testing.T) {
	uc, w := setup(t)
	id := w.requirements[req1].Snapshot().Id

	out, err := setComponents(uc, "COMP-2026-0001", "COMP-2026-0002")
	if err != nil {
		t.Fatal(err)
	}
	if namesOf(out) != "lib-firmas,svc-pagos" || len(w.components[id]) != 2 {
		t.Errorf("componentes = %+v, guardados = %v", out, w.components[id])
	}
	last := w.audit[len(w.audit)-1]
	if len(w.audit) != 2 || last.Action != port.AuditUpdate || last.Field != "components" || last.Actor != tester.Id ||
		last.RequirementId != id || last.Code.String() != req1 ||
		last.OldValue != "" || last.NewValue != "COMP-2026-0001, COMP-2026-0002" {
		t.Errorf("auditoría inesperada: %+v", w.audit[1:])
	}

	// Declarar lo mismo, en otro orden, no escribe ni audita.
	if out, err = setComponents(uc, "COMP-2026-0002", "COMP-2026-0001"); err != nil || namesOf(out) != "lib-firmas,svc-pagos" || len(w.audit) != 2 {
		t.Errorf("sin cambios: %+v, %v, auditoría %d", out, err, len(w.audit))
	}

	// Reemplaza la lista: lo que no viene se quita.
	if out, err = setComponents(uc, "COMP-2026-0002"); err != nil || namesOf(out) != "lib-firmas" {
		t.Fatalf("reemplazo: %+v, %v", out, err)
	}
	if last = w.audit[len(w.audit)-1]; last.OldValue != "COMP-2026-0001, COMP-2026-0002" || last.NewValue != "COMP-2026-0002" {
		t.Errorf("auditoría del reemplazo: %+v", last)
	}

	// Una lista vacía quita todos.
	if out, err = setComponents(uc); err != nil || len(out) != 0 || len(w.components[id]) != 0 {
		t.Errorf("vaciar: %+v, %v", out, err)
	}
	if last = w.audit[len(w.audit)-1]; len(w.audit) != 4 || last.OldValue != "COMP-2026-0002" || last.NewValue != "" {
		t.Errorf("auditoría al vaciar: %+v", last)
	}

	listed, err := uc.ListRequirementComponents.Execute(ctx, usecase.ListRequirementComponentsInput{Code: req1})
	if err != nil || listed == nil || len(listed) != 0 {
		t.Errorf("lista vacía = %#v, %v", listed, err)
	}
}

func TestSetRequirementComponentsErrors(t *testing.T) {
	valid := []string{"COMP-2026-0001"}
	cases := []struct {
		name      string
		in        usecase.SetRequirementComponentsInput
		wantCode  axis.Code
		wantField string
	}{
		{"sin permiso", usecase.SetRequirementComponentsInput{Actor: auditor, Code: req1, Components: valid}, axis.PermissionDenied, ""},
		{"sin lista", usecase.SetRequirementComponentsInput{Actor: tester, Code: req1}, axis.InvalidRequest, "components:required"},
		{"REQ inexistente", usecase.SetRequirementComponentsInput{Actor: tester, Code: "REQ-2026-0099", Components: valid}, axis.NotFoundResource, ""},
		{"código de REQ mal formado", usecase.SetRequirementComponentsInput{Actor: tester, Code: "x", Components: valid}, axis.NotFoundResource, ""},
		{"componente inexistente", usecase.SetRequirementComponentsInput{Actor: tester, Code: req1, Components: []string{"COMP-2026-0099"}}, axis.InvalidRequest, "components:invalid"},
		{"componente de otro sistema", usecase.SetRequirementComponentsInput{Actor: tester, Code: req1, Components: []string{"COMP-2026-0001", "COMP-2026-0004"}}, axis.InvalidRequest, "components:invalid"},
		{"componente retirado", usecase.SetRequirementComponentsInput{Actor: tester, Code: req1, Components: []string{"COMP-2026-0003"}}, axis.InvalidRequest, "components:invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			if _, err := setComponents(uc, "COMP-2026-0002"); err != nil {
				t.Fatal(err)
			}
			_, err := uc.SetRequirementComponents.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if tc.wantField != "" && strings.Join(fieldsOf(err), " ") != tc.wantField {
				t.Errorf("campos = %v, quiero %s", fieldsOf(err), tc.wantField)
			}
			id := w.requirements[req1].Snapshot().Id
			if !slices.Equal(w.components[id], []string{"COMP-2026-0002"}) || len(w.audit) != 2 {
				t.Error("una declaración rechazada no debe cambiar los componentes ni auditarse")
			}
		})
	}
}

func TestListRequirementComponents(t *testing.T) {
	uc, _ := setup(t)
	if _, err := setComponents(uc, "COMP-2026-0001", "COMP-2026-0002"); err != nil {
		t.Fatal(err)
	}
	out, err := uc.ListRequirementComponents.Execute(ctx, usecase.ListRequirementComponentsInput{Code: req1})
	if err != nil || namesOf(out) != "lib-firmas,svc-pagos" || out[0].Code != "COMP-2026-0002" || out[0].Kind != "library" {
		t.Errorf("lista = %+v, %v", out, err)
	}
	for _, code := range []string{"REQ-2026-0099", "x"} {
		if _, err := uc.ListRequirementComponents.Execute(ctx, usecase.ListRequirementComponentsInput{Code: code}); !errors.Is(err, usecase.ErrRequirementNotFound) {
			t.Errorf("lista de %q: %v", code, err)
		}
	}
}

func TestComponentsAreNotSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	if _, err := setComponents(uc, "COMP-2026-0002"); err != nil {
		t.Fatal(err)
	}
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")
	if _, err := setComponents(uc, "COMP-2026-0001"); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Fatalf("esperaba SERVICE_UNAVAILABLE, obtuve %v", err)
	}
	id := w.requirements[req1].Snapshot().Id
	if !slices.Equal(w.components[id], []string{"COMP-2026-0002"}) || len(w.audit) != 2 {
		t.Error("si la auditoría falla, los componentes deben quedar como estaban")
	}
}
