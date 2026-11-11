package domain_test

import (
	"errors"
	"testing"

	"kaoru/internal/shared/domain"
)

func TestParseCode(t *testing.T) {
	cases := []struct {
		raw   string
		valid bool
	}{
		{"REQ-2026-0042", true},
		{"CHG-2026-0007", true},
		{"RUN-2026-12345", true},
		{"", false},
		{"req-2026-0042", false},
		{"REQ-26-0042", false},
		{"REQ-2026-42", false},
		{"REQ-2026-0042 ", false},
		{"REQ-2026-0042; DROP TABLE", false},
	}
	for _, tc := range cases {
		code, err := domain.ParseCode(tc.raw)
		if tc.valid && (err != nil || code.String() != tc.raw) {
			t.Errorf("ParseCode(%q) debía valer, obtuve %v", tc.raw, err)
		}
		if !tc.valid && !errors.Is(err, domain.ErrInvalidCode) {
			t.Errorf("ParseCode(%q) debía fallar con ErrInvalidCode, obtuve %v", tc.raw, err)
		}
	}
}

func TestNewCode(t *testing.T) {
	cases := []struct {
		seq  int
		want string
	}{{7, "CHG-2026-0007"}, {42, "CHG-2026-0042"}, {12345, "CHG-2026-12345"}}
	for _, tc := range cases {
		got := domain.NewCode(domain.PrefixChange, 2026, tc.seq)
		if got.String() != tc.want {
			t.Errorf("NewCode(%d) = %s, quiero %s", tc.seq, got, tc.want)
		}
		if _, err := domain.ParseCode(got.String()); err != nil {
			t.Errorf("NewCode(%d) produjo un código que ParseCode rechaza", tc.seq)
		}
	}
}
