package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/change/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now      = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	start    = now.Add(24 * time.Hour)
	end      = now.Add(26 * time.Hour)
	sypago   = domain.System{Key: "SYPAGO", Name: "SyPago", DefaultRoute: domain.RouteLight}
	initial  = domain.State{Id: "s1", Key: "registered", Name: "Registrado"}
	approved = domain.State{Id: "s2", Key: "approved", Name: "Aprobado"}
)

func params() domain.NewParams {
	return domain.NewParams{
		Id: "c1", Code: shared.NewCode(shared.PrefixChange, 2026, 7), Title: "  Límite diario  ",
		Description: " Ajuste ", System: sypago, State: initial, CreatedBy: "u1", Now: now,
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

func TestNew(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(p *domain.NewParams)
		wantFields []string
		wantRoute  domain.Route
	}{
		{"válido: recorta y toma la ruta del sistema", func(*domain.NewParams) {}, nil, domain.RouteLight},
		{"ruta explícita", func(p *domain.NewParams) { p.Route = "minor_fix" }, nil, domain.RouteMinorFix},
		{"con ventana", func(p *domain.NewParams) { p.WindowStart, p.WindowEnd = &start, &end }, nil, domain.RouteLight},
		{"solo inicio de ventana", func(p *domain.NewParams) { p.WindowStart = &start }, nil, domain.RouteLight},
		{"sin título", func(p *domain.NewParams) { p.Title = "   " }, []string{"title:required"}, ""},
		{"título largo", func(p *domain.NewParams) { p.Title = strings.Repeat("á", 201) }, []string{"title:invalid"}, ""},
		{"título de 200 con acentos", func(p *domain.NewParams) { p.Title = strings.Repeat("á", 200) }, nil, domain.RouteLight},
		{"ruta desconocida", func(p *domain.NewParams) { p.Route = "alta" }, []string{"route:invalid"}, ""},
		{"fin sin inicio", func(p *domain.NewParams) { p.WindowEnd = &end }, []string{"windowStart:required"}, ""},
		{"fin antes del inicio", func(p *domain.NewParams) { p.WindowStart, p.WindowEnd = &end, &start }, []string{"windowEnd:invalid"}, ""},
		{"fin igual al inicio", func(p *domain.NewParams) { p.WindowStart, p.WindowEnd = &start, &start }, []string{"windowEnd:invalid"}, ""},
		{"junta todos los errores", func(p *domain.NewParams) { p.Title, p.Route, p.WindowEnd = "", "x", &end },
			[]string{"title:required", "route:invalid", "windowStart:required"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.mutate(&p)
			change, err := domain.New(p)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := change.Snapshot()
			if s.Route != tc.wantRoute || s.State != initial || s.CreatedAt != now || s.UpdatedAt != now {
				t.Errorf("contenido inesperado: %+v", s)
			}
			if s.Title != strings.TrimSpace(p.Title) || s.Description != "Ajuste" {
				t.Errorf("no recortó los textos: %q, %q", s.Title, s.Description)
			}
		})
	}
}

func ptr(s string) *string { return &s }

func TestApply(t *testing.T) {
	later := now.Add(time.Hour)
	cases := []struct {
		name        string
		edit        domain.Edit
		wantChanged []string
		wantFields  []string
	}{
		{"sin cambios", domain.Edit{}, nil, nil},
		{"mismo valor no cuenta", domain.Edit{Title: ptr("Límite diario")}, nil, nil},
		{"cambia el título", domain.Edit{Title: ptr("Otro")}, []string{"title"}, nil},
		{"cambia ruta y descripción", domain.Edit{Route: ptr("high"), Description: ptr("Nueva")}, []string{"description", "route"}, nil},
		{"quita la ventana", domain.Edit{WindowStart: domain.TimeEdit{Set: true}, WindowEnd: domain.TimeEdit{Set: true}}, []string{"windowStart", "windowEnd"}, nil},
		{"quitar solo el inicio deja un fin sin inicio", domain.Edit{WindowStart: domain.TimeEdit{Set: true}}, nil, []string{"windowStart:required"}},
		{"título vacío", domain.Edit{Title: ptr(" ")}, nil, []string{"title:required"}},
		{"ruta desconocida", domain.Edit{Route: ptr("ligera")}, nil, []string{"route:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			p.WindowStart, p.WindowEnd = &start, &end
			change, err := domain.New(p)
			if err != nil {
				t.Fatal(err)
			}
			before := change.Snapshot()

			changes, err := change.Apply(tc.edit, later)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				if change.Snapshot() != before {
					t.Error("una edición rechazada no debe cambiar el CHG")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var changed []string
			for _, c := range changes {
				changed = append(changed, c.Field)
				if c.Old == c.New {
					t.Errorf("%s aparece como cambio sin cambiar", c.Field)
				}
			}
			if !slices.Equal(changed, tc.wantChanged) {
				t.Errorf("cambios = %v, quiero %v", changed, tc.wantChanged)
			}
			wantUpdated := now
			if len(tc.wantChanged) > 0 {
				wantUpdated = later
			}
			if change.Snapshot().UpdatedAt != wantUpdated {
				t.Errorf("updatedAt = %v, quiero %v", change.Snapshot().UpdatedAt, wantUpdated)
			}
		})
	}
}

func TestMoveTo(t *testing.T) {
	change, err := domain.New(params())
	if err != nil {
		t.Fatal(err)
	}
	later := now.Add(time.Hour)
	moved := change.MoveTo(approved, later)
	if moved != (domain.FieldChange{Field: "state", Old: "registered", New: "approved"}) {
		t.Errorf("cambio de estado = %+v", moved)
	}
	if s := change.Snapshot(); s.State != approved || s.UpdatedAt != later {
		t.Errorf("no quedó en el estado nuevo: %+v", s)
	}
}
