package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/component/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now    = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	later  = now.Add(time.Hour)
	sypago = domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: "light"}
)

func params() domain.NewParams {
	return domain.NewParams{
		Id: "k1", Code: shared.NewCode(shared.PrefixComponent, 2026, 7), System: sypago, Kind: "microservice",
		Name: "  svc-pagos  ", Description: " Servicio de pagos ", CreatedBy: "u1", Now: now,
	}
}

// fieldsOf devuelve "campo:código" de cada error de campo.
func fieldsOf(t *testing.T, err error) []string {
	t.Helper()
	var ge *axis.GeneralError
	if !errors.As(err, &ge) || ge.Code() != axis.InvalidRequest {
		t.Fatalf("esperaba INVALID_REQUEST, obtuve %v", err)
	}
	var out []string
	for _, f := range ge.Fields() {
		out = append(out, f.Field+":"+f.Code)
	}
	return out
}

func ptr[T any](v T) *T { return &v }

func TestParseKind(t *testing.T) {
	for _, raw := range []string{"microservice", "library", "script", "kafka_topic", "db_schema", "other"} {
		if kind, ok := domain.ParseKind(raw); !ok || string(kind) != raw {
			t.Errorf("ParseKind(%q) debía valer", raw)
		}
	}
	for _, raw := range []string{"", "Microservice", "dll", "kafka-topic"} {
		if _, ok := domain.ParseKind(raw); ok {
			t.Errorf("ParseKind(%q) no debía valer", raw)
		}
	}
}

func TestNew(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewParams)
		wantFields []string
	}{
		{"válido: recorta nombre y descripción", func(*domain.NewParams) {}, nil},
		{"sin descripción", func(p *domain.NewParams) { p.Description = "" }, nil},
		{"nombre de 120 con acentos", func(p *domain.NewParams) { p.Name = strings.Repeat("á", 120) }, nil},
		{"sin nombre", func(p *domain.NewParams) { p.Name = "   " }, []string{"name:required"}},
		{"nombre largo", func(p *domain.NewParams) { p.Name = strings.Repeat("á", 121) }, []string{"name:invalid"}},
		{"sin clase", func(p *domain.NewParams) { p.Kind = "" }, []string{"kind:required"}},
		{"clase desconocida", func(p *domain.NewParams) { p.Kind = "dll" }, []string{"kind:invalid"}},
		{"junta todos los errores", func(p *domain.NewParams) { p.Kind, p.Name = "x", "" }, []string{"kind:invalid", "name:required"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			component, err := domain.New(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Errorf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := component.Snapshot()
			if s.Id != "k1" || s.Code.String() != "COMP-2026-0007" || s.System != sypago || s.Name != strings.TrimSpace(p.Name) ||
				s.Description != strings.TrimSpace(p.Description) || string(s.Kind) != p.Kind || s.RetiredAt != nil ||
				component.Retired() || s.CreatedBy != "u1" || s.CreatedAt != now || s.UpdatedAt != now {
				t.Errorf("COMP inesperado: %+v", s)
			}
		})
	}
}

func TestApply(t *testing.T) {
	cases := []struct {
		name        string
		edit        domain.Edit
		wantFields  []string
		wantChanges []domain.FieldChange
	}{
		{"sin campos no cambia nada", domain.Edit{}, nil, nil},
		{"los mismos valores no cambian nada", domain.Edit{Kind: ptr("microservice"), Name: ptr(" svc-pagos "), Retired: ptr(false)}, nil, nil},
		{"cambia nombre, clase y descripción",
			domain.Edit{Kind: ptr("library"), Name: ptr(" lib-pagos "), Description: ptr(" Otra ")}, nil,
			[]domain.FieldChange{
				{Field: "kind", Old: "microservice", New: "library"},
				{Field: "name", Old: "svc-pagos", New: "lib-pagos"},
				{Field: "description", Old: "Servicio de pagos", New: "Otra"},
			}},
		{"vacía la descripción", domain.Edit{Description: ptr("")}, nil,
			[]domain.FieldChange{{Field: "description", Old: "Servicio de pagos", New: ""}}},
		{"retira", domain.Edit{Retired: ptr(true)}, nil, []domain.FieldChange{{Field: "retired", Old: "false", New: "true"}}},
		{"nombre vacío", domain.Edit{Name: ptr(" ")}, []string{"name:required"}, nil},
		{"nombre largo", domain.Edit{Name: ptr(strings.Repeat("x", 121))}, []string{"name:invalid"}, nil},
		{"clase vacía", domain.Edit{Kind: ptr("")}, []string{"kind:required"}, nil},
		{"clase desconocida", domain.Edit{Kind: ptr("dll"), Retired: ptr(true)}, []string{"kind:invalid"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			component, err := domain.New(params())
			if err != nil {
				t.Fatal(err)
			}
			before := component.Snapshot()
			changes, err := component.Apply(tc.edit, later)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Errorf("campos = %v, quiero %v", got, tc.wantFields)
				}
				if component.Snapshot() != before {
					t.Error("una edición rechazada no debe cambiar el COMP")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(changes, tc.wantChanges) {
				t.Errorf("cambios = %+v, quiero %+v", changes, tc.wantChanges)
			}
			wantUpdated := now
			if len(tc.wantChanges) > 0 {
				wantUpdated = later
			}
			if component.Snapshot().UpdatedAt != wantUpdated {
				t.Errorf("updatedAt = %s, quiero %s", component.Snapshot().UpdatedAt, wantUpdated)
			}
		})
	}
}

func TestRetireAndRestore(t *testing.T) {
	component, err := domain.New(params())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := component.Apply(domain.Edit{Retired: ptr(true)}, later); err != nil {
		t.Fatal(err)
	}
	if s := component.Snapshot(); !component.Retired() || s.RetiredAt == nil || !s.RetiredAt.Equal(later) {
		t.Fatalf("debía quedar retirado en %s: %+v", later, s)
	}
	// Retirar lo ya retirado no cambia la fecha ni cuenta como cambio.
	again := later.Add(time.Hour)
	if changes, err := component.Apply(domain.Edit{Retired: ptr(true)}, again); err != nil || len(changes) != 0 ||
		!component.Snapshot().RetiredAt.Equal(later) || component.Snapshot().UpdatedAt != later {
		t.Errorf("retirar dos veces: %+v, %v", changes, err)
	}
	changes, err := component.Apply(domain.Edit{Retired: ptr(false)}, again)
	if err != nil || !slices.Equal(changes, []domain.FieldChange{{Field: "retired", Old: "true", New: "false"}}) {
		t.Fatalf("reactivar: %+v, %v", changes, err)
	}
	if component.Retired() || component.Snapshot().RetiredAt != nil || component.Snapshot().UpdatedAt != again {
		t.Errorf("debía quedar vigente: %+v", component.Snapshot())
	}
}

func TestRehydrate(t *testing.T) {
	s := domain.Snapshot{Id: "k9", Name: "x", Kind: domain.KindOther, RetiredAt: &now}
	if c := domain.Rehydrate(s); c.Snapshot() != s || !c.Retired() {
		t.Error("Rehydrate debe conservar el contenido")
	}
}

func TestErrNameTaken(t *testing.T) {
	if got := fieldsOf(t, domain.ErrNameTaken); !slices.Equal(got, []string{"name:invalid"}) {
		t.Errorf("campos = %v", got)
	}
}
