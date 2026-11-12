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
	"kaoru/internal/feature/component/application/port"
	"kaoru/internal/feature/component/application/usecase"
	"kaoru/internal/feature/component/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria que comparten los dobles. Las escrituras hechas dentro
// de Within se descartan si la función falla, como en una transacción real.
type world struct {
	components map[string]domain.Component // por código
	audit      []port.AuditRecord
	sequence   int
	lastFilter port.Filter

	auditErr error
	inTx     bool
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	components, audit, sequence := maps.Clone(w.components), slices.Clone(w.audit), w.sequence
	w.inTx = true
	defer func() { w.inTx = false }()
	if err := fn(ctx); err != nil {
		w.components, w.audit, w.sequence = components, audit, sequence
		return err
	}
	return nil
}

var notFound = axis.New(axis.NotFoundResource, "No existe.")

func (w *world) NextCode(_ context.Context, year int) (shared.Code, error) {
	w.sequence++
	return shared.NewCode(shared.PrefixComponent, year, w.sequence), nil
}

func (w *world) FindSystem(_ context.Context, key string) (domain.System, error) {
	switch key {
	case "SYPAGO":
		return domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"}, nil
	case "SIMF":
		return domain.System{Key: "SIMF", Name: "SIMF", DefaultRoute: "high"}, nil
	}
	return domain.System{}, notFound
}

// save guarda como la base: el nombre no se repite dentro de un sistema.
func (w *world) save(c domain.Component) error {
	if !w.inTx {
		return errors.New("escritura fuera de una transacción")
	}
	s := c.Snapshot()
	for _, other := range w.components {
		o := other.Snapshot()
		if o.Id != s.Id && o.System.Key == s.System.Key && o.Name == s.Name {
			return domain.ErrNameTaken
		}
	}
	w.components[s.Code.String()] = c
	return nil
}

func (w *world) Insert(_ context.Context, c domain.Component) error { return w.save(c) }
func (w *world) Update(_ context.Context, c domain.Component) error { return w.save(c) }

// Lock no hace nada en memoria: el doble ya ejecuta una transacción a la vez.
func (w *world) Lock(ctx context.Context, code shared.Code) error {
	if !w.inTx {
		return errors.New("Lock fuera de una transacción")
	}
	_, err := w.FindByCode(ctx, code)
	return err
}

func (w *world) FindByCode(_ context.Context, code shared.Code) (port.ComponentView, error) {
	c, ok := w.components[code.String()]
	if !ok {
		return port.ComponentView{}, notFound
	}
	return port.ComponentView{Component: c, CreatedByName: "Ana"}, nil
}

func (w *world) FindByCodes(_ context.Context, codes []string) ([]domain.Component, error) {
	var out []domain.Component
	for _, code := range codes {
		if c, ok := w.components[code]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// List aplica el filtro en memoria y recuerda el último que recibió.
func (w *world) List(_ context.Context, f port.Filter) ([]port.ComponentView, int, error) {
	w.lastFilter = f
	var views []port.ComponentView
	for _, c := range w.components {
		s := c.Snapshot()
		if (f.SystemKey != "" && s.System.Key != f.SystemKey) || (f.Kind != "" && string(s.Kind) != f.Kind) ||
			!strings.Contains(s.Name, f.Query) || (!f.IncludeRetired && c.Retired()) {
			continue
		}
		views = append(views, port.ComponentView{Component: c, CreatedByName: "Ana"})
	}
	slices.SortFunc(views, func(a, b port.ComponentView) int {
		return strings.Compare(a.Component.Snapshot().Name, b.Component.Snapshot().Name)
	})
	return views, len(views), nil
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

// fakePermissions: dev y qa mantienen el catálogo.
type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	if permission == usecase.PermissionManage && (slices.Contains(roles, "dev") || slices.Contains(roles, "qa")) {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

type fixedClock struct{ t time.Time }

func (c *fixedClock) Now() time.Time { return c.t }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	now     = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	dev     = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	qa      = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	manager = shared.Actor{Id: "u1", Roles: []string{"change-manager"}}
	ctx     = context.Background()
)

const comp1 = "COMP-2026-0001"

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{components: map[string]domain.Component{}}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Permissions: fakePermissions{}, AuditTrail: w,
		UnitOfWork: w, Clock: &fixedClock{now}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

// create agrega un COMP y devuelve su código.
func create(t *testing.T, uc *usecase.UseCases, system, kind, name string) string {
	t.Helper()
	out, err := uc.CreateComponent.Execute(ctx, usecase.CreateComponentInput{
		Actor: dev, SystemKey: system, Kind: kind, Name: name, Description: "Detalle",
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.Component.Code.String()
}

func retire(t *testing.T, uc *usecase.UseCases, code string) {
	t.Helper()
	if _, err := uc.UpdateComponent.Execute(ctx, usecase.UpdateComponentInput{Actor: qa, Code: code, Edit: domain.Edit{Retired: ptr(true)}}); err != nil {
		t.Fatal(err)
	}
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

func ptr[T any](v T) *T { return &v }

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestCreateComponent(t *testing.T) {
	uc, w := setup(t)
	out, err := uc.CreateComponent.Execute(ctx, usecase.CreateComponentInput{
		Actor: dev, SystemKey: "SYPAGO", Kind: "microservice", Name: " svc-pagos ", Description: " Pagos ",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := out.Component
	if s.Code.String() != comp1 || s.Id != "id-1" || s.System.Key != "SYPAGO" || s.System.Name != "SyPago" ||
		s.Kind != domain.KindMicroservice || s.Name != "svc-pagos" || s.Description != "Pagos" || s.RetiredAt != nil ||
		s.CreatedBy != "u3" || s.CreatedAt != now || s.UpdatedAt != now || out.CreatedByName != "Ana" {
		t.Errorf("COMP inesperado: %+v", out)
	}
	if saved, ok := w.components[comp1]; !ok || saved.Snapshot() != s {
		t.Errorf("no guardó lo que devolvió: %+v", saved.Snapshot())
	}
	want := port.AuditRecord{Actor: "u3", ComponentId: "id-1", Code: s.Code, Action: port.AuditCreate, NewValue: "svc-pagos"}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Errorf("auditoría inesperada: %+v", w.audit)
	}
	// El mismo nombre vale en otro sistema, y qa también mantiene el catálogo.
	second, err := uc.CreateComponent.Execute(ctx, usecase.CreateComponentInput{Actor: qa, SystemKey: "SIMF", Kind: "library", Name: "svc-pagos"})
	if err != nil || second.Component.Code.String() != "COMP-2026-0002" {
		t.Errorf("segundo COMP = %+v, %v", second.Component, err)
	}
}

func TestCreateComponentErrors(t *testing.T) {
	cases := []struct {
		name      string
		in        usecase.CreateComponentInput
		wantCode  axis.Code
		wantField string
	}{
		{"sin permiso", usecase.CreateComponentInput{Actor: manager, SystemKey: "SYPAGO", Kind: "script", Name: "x"}, axis.PermissionDenied, ""},
		{"sin sistema", usecase.CreateComponentInput{Actor: dev, Kind: "script", Name: "x"}, axis.InvalidRequest, "system:required"},
		{"sistema desconocido", usecase.CreateComponentInput{Actor: dev, SystemKey: "OTRO", Kind: "script", Name: "x"}, axis.InvalidRequest, "system:invalid"},
		{"sin nombre", usecase.CreateComponentInput{Actor: dev, SystemKey: "SYPAGO", Kind: "script"}, axis.InvalidRequest, "name:required"},
		{"sin clase", usecase.CreateComponentInput{Actor: dev, SystemKey: "SYPAGO", Name: "x"}, axis.InvalidRequest, "kind:required"},
		{"clase desconocida", usecase.CreateComponentInput{Actor: dev, SystemKey: "SYPAGO", Kind: "dll", Name: "x"}, axis.InvalidRequest, "kind:invalid"},
		{"nombre repetido en el sistema", usecase.CreateComponentInput{Actor: dev, SystemKey: "SYPAGO", Kind: "script", Name: " svc-pagos "}, axis.InvalidRequest, "name:invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			create(t, uc, "SYPAGO", "microservice", "svc-pagos")
			_, err := uc.CreateComponent.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if tc.wantField != "" && strings.Join(fieldsOf(err), " ") != tc.wantField {
				t.Errorf("campos = %v, quiero %s", fieldsOf(err), tc.wantField)
			}
			if len(w.components) != 1 || len(w.audit) != 1 || w.sequence != 1 {
				t.Error("un alta rechazada no debe dejar componentes, auditoría ni gastar un código")
			}
		})
	}
}

func TestGetComponent(t *testing.T) {
	uc, _ := setup(t)
	create(t, uc, "SYPAGO", "microservice", "svc-pagos")
	if out, err := uc.GetComponent.Execute(ctx, usecase.GetComponentInput{Code: comp1}); err != nil || out.Component.Name != "svc-pagos" || out.CreatedByName != "Ana" {
		t.Errorf("GetComponent = %+v, %v", out, err)
	}
	for _, code := range []string{"COMP-2026-0099", "no-es-un-codigo", ""} {
		if _, err := uc.GetComponent.Execute(ctx, usecase.GetComponentInput{Code: code}); !errors.Is(err, usecase.ErrComponentNotFound) {
			t.Errorf("GetComponent(%q): esperaba ErrComponentNotFound, obtuve %v", code, err)
		}
	}
}

func TestListComponents(t *testing.T) {
	uc, w := setup(t)
	create(t, uc, "SYPAGO", "microservice", "svc-pagos")
	create(t, uc, "SYPAGO", "library", "lib-firmas")
	create(t, uc, "SIMF", "microservice", "svc-simf")
	retire(t, uc, create(t, uc, "SYPAGO", "script", "job-viejo"))

	cases := []struct {
		name string
		in   usecase.ListComponentsInput
		want string
	}{
		{"sin filtro: vigentes, por nombre", usecase.ListComponentsInput{}, "lib-firmas,svc-pagos,svc-simf"},
		{"con retirados", usecase.ListComponentsInput{IncludeRetired: true}, "job-viejo,lib-firmas,svc-pagos,svc-simf"},
		{"por sistema", usecase.ListComponentsInput{SystemKey: "SIMF"}, "svc-simf"},
		{"por clase", usecase.ListComponentsInput{Kind: "microservice"}, "svc-pagos,svc-simf"},
		{"por nombre", usecase.ListComponentsInput{Query: "firm"}, "lib-firmas"},
		{"todo junto", usecase.ListComponentsInput{SystemKey: "SYPAGO", Kind: "script", Query: "viejo", IncludeRetired: true}, "job-viejo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Page, tc.in.Size = 2, 10
			out, err := uc.ListComponents.Execute(ctx, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			names := make([]string, len(out.Components))
			for i, c := range out.Components {
				names[i] = c.Component.Name
			}
			if got := strings.Join(names, ","); got != tc.want || out.Total != len(names) {
				t.Errorf("lista = %s (total %d), quiero %s", got, out.Total, tc.want)
			}
			want := port.Filter{
				SystemKey: tc.in.SystemKey, Kind: tc.in.Kind, Query: tc.in.Query, IncludeRetired: tc.in.IncludeRetired, Page: 2, Size: 10,
			}
			if w.lastFilter != want {
				t.Errorf("filtro = %+v, quiero %+v", w.lastFilter, want)
			}
		})
	}

	if _, err := uc.ListComponents.Execute(ctx, usecase.ListComponentsInput{Kind: "dll"}); strings.Join(fieldsOf(err), " ") != "kind:invalid" {
		t.Errorf("clase desconocida: %v", err)
	}
}

func TestUpdateComponent(t *testing.T) {
	uc, w := setup(t)
	create(t, uc, "SYPAGO", "microservice", "svc-pagos")

	out, err := uc.UpdateComponent.Execute(ctx, usecase.UpdateComponentInput{
		Actor: qa, Code: comp1, Edit: domain.Edit{Name: ptr("svc-cobros"), Kind: ptr("library")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Component.Name != "svc-cobros" || out.Component.Kind != domain.KindLibrary || out.CreatedByName != "Ana" ||
		w.components[comp1].Snapshot() != out.Component {
		t.Errorf("no aplicó la edición: %+v", out.Component)
	}
	want := []port.AuditRecord{
		{Actor: "u2", ComponentId: "id-1", Code: out.Component.Code, Action: port.AuditUpdate, Field: "kind", OldValue: "microservice", NewValue: "library"},
		{Actor: "u2", ComponentId: "id-1", Code: out.Component.Code, Action: port.AuditUpdate, Field: "name", OldValue: "svc-pagos", NewValue: "svc-cobros"},
	}
	if !slices.Equal(w.audit[1:], want) {
		t.Errorf("debe auditar cada campo que cambió: %+v", w.audit[1:])
	}

	// Editar sin cambiar nada no escribe ni audita.
	if _, err := uc.UpdateComponent.Execute(ctx, usecase.UpdateComponentInput{Actor: qa, Code: comp1, Edit: domain.Edit{Name: ptr("svc-cobros")}}); err != nil {
		t.Fatal(err)
	}
	if len(w.audit) != 3 {
		t.Error("una edición sin cambios no debe auditarse")
	}
}

func TestRetireAndRestoreComponent(t *testing.T) {
	uc, w := setup(t)
	create(t, uc, "SYPAGO", "microservice", "svc-pagos")

	out, err := uc.UpdateComponent.Execute(ctx, usecase.UpdateComponentInput{Actor: dev, Code: comp1, Edit: domain.Edit{Retired: ptr(true)}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Component.RetiredAt == nil || !out.Component.RetiredAt.Equal(now) || !w.components[comp1].Retired() {
		t.Errorf("debía quedar retirado: %+v", out.Component)
	}
	if last := w.audit[len(w.audit)-1]; last.Field != "retired" || last.OldValue != "false" || last.NewValue != "true" || last.Action != port.AuditUpdate {
		t.Errorf("auditoría al retirar: %+v", last)
	}

	out, err = uc.UpdateComponent.Execute(ctx, usecase.UpdateComponentInput{Actor: dev, Code: comp1, Edit: domain.Edit{Retired: ptr(false)}})
	if err != nil || out.Component.RetiredAt != nil || w.components[comp1].Retired() {
		t.Errorf("debía quedar vigente: %+v, %v", out.Component, err)
	}
	if last := w.audit[len(w.audit)-1]; len(w.audit) != 3 || last.OldValue != "true" || last.NewValue != "false" {
		t.Errorf("auditoría al reactivar: %+v", last)
	}
}

func TestUpdateComponentErrors(t *testing.T) {
	cases := []struct {
		name      string
		in        usecase.UpdateComponentInput
		wantCode  axis.Code
		wantField string
	}{
		{"sin permiso", usecase.UpdateComponentInput{Actor: manager, Code: comp1, Edit: domain.Edit{Name: ptr("x")}}, axis.PermissionDenied, ""},
		{"COMP inexistente", usecase.UpdateComponentInput{Actor: dev, Code: "COMP-2026-0099", Edit: domain.Edit{Name: ptr("x")}}, axis.NotFoundResource, ""},
		{"código mal formado", usecase.UpdateComponentInput{Actor: dev, Code: "x", Edit: domain.Edit{Name: ptr("x")}}, axis.NotFoundResource, ""},
		{"nombre vacío", usecase.UpdateComponentInput{Actor: dev, Code: comp1, Edit: domain.Edit{Name: ptr(" ")}}, axis.InvalidRequest, "name:required"},
		{"clase desconocida", usecase.UpdateComponentInput{Actor: dev, Code: comp1, Edit: domain.Edit{Kind: ptr("dll")}}, axis.InvalidRequest, "kind:invalid"},
		{"nombre de otro componente del sistema", usecase.UpdateComponentInput{Actor: dev, Code: comp1, Edit: domain.Edit{Name: ptr("lib-firmas"), Retired: ptr(true)}}, axis.InvalidRequest, "name:invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			create(t, uc, "SYPAGO", "microservice", "svc-pagos")
			create(t, uc, "SYPAGO", "library", "lib-firmas")
			before := w.components[comp1].Snapshot()
			_, err := uc.UpdateComponent.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) {
				t.Fatalf("esperaba %s, obtuve %v", tc.wantCode, err)
			}
			if tc.wantField != "" && strings.Join(fieldsOf(err), " ") != tc.wantField {
				t.Errorf("campos = %v, quiero %s", fieldsOf(err), tc.wantField)
			}
			if w.components[comp1].Snapshot() != before || len(w.audit) != 2 {
				t.Error("una edición rechazada no debe cambiar el COMP ni auditarse")
			}
		})
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	create(t, uc, "SYPAGO", "microservice", "svc-pagos")
	before := w.components[comp1].Snapshot()
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")

	if _, err := uc.CreateComponent.Execute(ctx, usecase.CreateComponentInput{Actor: dev, SystemKey: "SYPAGO", Kind: "script", Name: "otro"}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("crear: %v", err)
	}
	if _, err := uc.UpdateComponent.Execute(ctx, usecase.UpdateComponentInput{Actor: dev, Code: comp1, Edit: domain.Edit{Name: ptr("otro")}}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("editar: %v", err)
	}
	if _, err := uc.UpdateComponent.Execute(ctx, usecase.UpdateComponentInput{Actor: dev, Code: comp1, Edit: domain.Edit{Retired: ptr(true)}}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("retirar: %v", err)
	}
	if len(w.components) != 1 || w.components[comp1].Snapshot() != before || w.sequence != 1 || len(w.audit) != 1 {
		t.Error("si la auditoría falla, nada debe quedar guardado")
	}
}

func TestResolveComponents(t *testing.T) {
	uc, _ := setup(t)
	pagos := create(t, uc, "SYPAGO", "microservice", "svc-pagos")
	firmas := create(t, uc, "SYPAGO", "library", "lib-firmas")
	simf := create(t, uc, "SIMF", "microservice", "svc-simf")
	viejo := create(t, uc, "SYPAGO", "script", "job-viejo")
	retire(t, uc, viejo)

	// Devuelve una referencia por código distinto, en el orden en que llegaron.
	refs, err := uc.ResolveComponents.Execute(ctx, usecase.ResolveComponentsInput{SystemKey: "SYPAGO", Codes: []string{firmas, pagos, firmas}})
	want := []usecase.ComponentRef{
		{Id: "id-2", Code: firmas, Name: "lib-firmas", Kind: domain.KindLibrary},
		{Id: "id-1", Code: pagos, Name: "svc-pagos", Kind: domain.KindMicroservice},
	}
	if err != nil || !slices.Equal(refs, want) {
		t.Errorf("refs = %+v, %v", refs, err)
	}
	if refs, err := uc.ResolveComponents.Execute(ctx, usecase.ResolveComponentsInput{SystemKey: "SYPAGO"}); err != nil || len(refs) != 0 {
		t.Errorf("sin códigos: %+v, %v", refs, err)
	}

	cases := []struct {
		name    string
		codes   []string
		message string
	}{
		{"no existe", []string{pagos, "COMP-2026-0099"}, "El componente COMP-2026-0099 no existe."},
		{"código mal formado", []string{"x"}, "El componente x no existe."},
		{"de otro sistema", []string{simf}, "El componente " + simf + " es de otro sistema."},
		{"retirado", []string{pagos, viejo}, "El componente " + viejo + " está retirado."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.ResolveComponents.Execute(ctx, usecase.ResolveComponentsInput{SystemKey: "SYPAGO", Codes: tc.codes})
			var ge *axis.GeneralError
			if !errors.As(err, &ge) || ge.Code() != axis.InvalidRequest || len(ge.Fields()) != 1 {
				t.Fatalf("esperaba un INVALID_REQUEST con un campo, obtuve %v", err)
			}
			if f := ge.Fields()[0]; f.Field != "components" || f.Code != axis.FieldInvalid || f.Message != tc.message {
				t.Errorf("campo = %+v", f)
			}
		})
	}
}
