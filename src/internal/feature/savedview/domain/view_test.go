package domain_test

import (
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/savedview/domain"
)

func TestNew(t *testing.T) {
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	v, err := domain.New("v1", "u1", "requirements", "  Mis REQ en pruebas ", "?state=in_testing&owner=u1", now)
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "Mis REQ en pruebas" || v.Query != "owner=u1&state=in_testing" || v.OwnerId != "u1" || !v.CreatedAt.Equal(now) {
		t.Errorf("vista inesperada: %+v", v)
	}

	cases := []struct {
		name                 string
		screen, title, query string
		fields               int
	}{
		{"lista desconocida", "paquetes", "x", "a=1", 1},
		{"sin nombre", "defects", "  ", "a=1", 1},
		{"nombre largo", "defects", strings.Repeat("ñ", 61), "a=1", 1},
		{"sin filtros", "defects", "x", "", 1},
		{"filtros ilegibles", "defects", "x", "a=%zz", 1},
		{"filtros enormes", "defects", "x", "a=" + strings.Repeat("b", 2000), 1},
		{"todo mal", "nope", "", "", 3},
	}
	for _, tc := range cases {
		_, err := domain.New("v1", "u1", tc.screen, tc.title, tc.query, now)
		if !axis.IsCode(err, axis.InvalidRequest) {
			t.Errorf("%s: quiero INVALID_REQUEST, obtuve %v", tc.name, err)
		}
	}
}
