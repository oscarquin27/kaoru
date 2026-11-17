package domain_test

import (
	"errors"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/system/domain"
)

var now = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

func newSystem(t *testing.T) domain.System {
	t.Helper()
	s, err := domain.New(domain.NewParams{Id: "id-1", Key: " sysaf ", Name: " SysAF ", DefaultRoute: "high", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewNormalizesKeyAndName(t *testing.T) {
	s := newSystem(t).Snapshot()
	if s.Key != "SYSAF" || s.Name != "SysAF" || s.DefaultRoute != domain.RouteHigh || !s.CreatedAt.Equal(now) {
		t.Fatalf("sistema inesperado: %+v", s)
	}
}

func TestNewRejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name   string
		params domain.NewParams
		field  string
	}{
		{"sin clave", domain.NewParams{Name: "X", DefaultRoute: "high"}, "key"},
		{"clave con espacio", domain.NewParams{Key: "SYS AF", Name: "X", DefaultRoute: "high"}, "key"},
		{"clave de una letra", domain.NewParams{Key: "S", Name: "X", DefaultRoute: "high"}, "key"},
		{"clave que empieza con dígito", domain.NewParams{Key: "1SYS", Name: "X", DefaultRoute: "high"}, "key"},
		{"clave larga", domain.NewParams{Key: "ABCDEFGHIJKLM", Name: "X", DefaultRoute: "high"}, "key"},
		{"sin nombre", domain.NewParams{Key: "SYSAF", Name: "  ", DefaultRoute: "high"}, "name"},
		{"sin ruta", domain.NewParams{Key: "SYSAF", Name: "X"}, "defaultRoute"},
		{"fix menor no es ruta de sistema", domain.NewParams{Key: "SYSAF", Name: "X", DefaultRoute: "minor_fix"}, "defaultRoute"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := domain.New(c.params)
			if !hasField(err, c.field) {
				t.Fatalf("se esperaba error en %s, llegó %v", c.field, err)
			}
		})
	}
}

func TestApplyReportsOnlyRealChanges(t *testing.T) {
	s := newSystem(t)
	later := now.Add(time.Hour)
	same, light := "SysAF", "light"
	changes, err := s.Apply(domain.Edit{Name: &same, DefaultRoute: &light}, later)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0] != (domain.FieldChange{Field: "defaultRoute", Old: "high", New: "light"}) {
		t.Fatalf("cambios inesperados: %+v", changes)
	}
	if got := s.Snapshot(); got.DefaultRoute != domain.RouteLight || !got.UpdatedAt.Equal(later) || got.Key != "SYSAF" {
		t.Fatalf("sistema inesperado: %+v", got)
	}

	changes, err = s.Apply(domain.Edit{DefaultRoute: &light}, later.Add(time.Hour))
	if err != nil || len(changes) != 0 || !s.Snapshot().UpdatedAt.Equal(later) {
		t.Fatalf("repetir el mismo valor no es un cambio: %+v %v", changes, err)
	}
}

func TestApplyRejectsInvalidEditWithoutChanging(t *testing.T) {
	s := newSystem(t)
	empty := ""
	if _, err := s.Apply(domain.Edit{Name: &empty}, now); !hasField(err, "name") {
		t.Fatalf("se esperaba error en name, llegó %v", err)
	}
	if s.Snapshot().Name != "SysAF" {
		t.Fatalf("una edición inválida no cambia nada: %+v", s.Snapshot())
	}
}

// hasField dice si err es un INVALID_REQUEST con un error en ese campo.
func hasField(err error, field string) bool {
	var ge *axis.GeneralError
	if !errors.As(err, &ge) || ge.Code() != axis.InvalidRequest {
		return false
	}
	for _, f := range ge.Fields() {
		if f.Field == field {
			return true
		}
	}
	return false
}
