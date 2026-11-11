package domain_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/risk/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	now  = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	code = shared.NewCode(shared.PrefixRisk, 2026, 19)
)

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
		in         domain.Input
		wantFields []string
		wantChars  []domain.Characteristic
	}{
		{"simplificado, sin características", domain.Input{Level: "low", Justification: " Módulo estable "}, nil, []domain.Characteristic{}},
		{"ordena y quita repetidas", domain.Input{Level: "high", Justification: "x", Characteristics: []string{"security", "reliability", "security"}},
			nil, []domain.Characteristic{"reliability", "security"}},
		{"sin nivel ni justificación", domain.Input{}, []string{"level:required", "justification:required"}, nil},
		{"nivel en español", domain.Input{Level: "alto", Justification: "x"}, []string{"level:invalid"}, nil},
		{"característica desconocida", domain.Input{Level: "low", Justification: "x", Characteristics: []string{"seguridad"}}, []string{"characteristics:invalid"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.AssessedBy, tc.in.Now = "u1", now
			assessment, err := domain.New("k1", code, "r1", tc.in)
			if tc.wantFields != nil {
				if got := fieldsOf(t, err); !slices.Equal(got, tc.wantFields) {
					t.Fatalf("campos = %v, quiero %v", got, tc.wantFields)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := assessment.Snapshot()
			if !slices.Equal(s.Characteristics, tc.wantChars) || s.Level != domain.Level(tc.in.Level) || s.RequirementId != "r1" || s.Code != code {
				t.Errorf("RISK inesperado: %+v", s)
			}
		})
	}
}

func TestReassess(t *testing.T) {
	assessment, err := domain.New("k1", code, "r1", domain.Input{Level: "low", Justification: "Estable", AssessedBy: "u1", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	later := now.Add(time.Hour)

	changes, err := assessment.Reassess(domain.Input{Level: "low", Justification: " Estable ", AssessedBy: "u2", Now: later})
	if err != nil || len(changes) != 0 || assessment.Snapshot().AssessedBy != "u1" || assessment.Snapshot().UpdatedAt != now {
		t.Errorf("reevaluar sin cambios no toca nada: %+v, %v", changes, err)
	}

	changes, err = assessment.Reassess(domain.Input{
		Level: "high", Justification: "Toca saldos", Characteristics: []string{"security"}, AssessedBy: "u2", Now: later,
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields []string
	for _, c := range changes {
		fields = append(fields, c.Field+":"+c.Old+"→"+c.New)
	}
	if !slices.Equal(fields, []string{"level:low→high", "justification:Estable→Toca saldos", "characteristics:→security"}) {
		t.Errorf("cambios = %v", fields)
	}
	if s := assessment.Snapshot(); s.AssessedBy != "u2" || s.UpdatedAt != later || s.CreatedAt != now {
		t.Errorf("debe quedar quién y cuándo reevaluó: %+v", s)
	}

	if _, err := assessment.Reassess(domain.Input{Level: "", Justification: ""}); len(fieldsOf(t, err)) != 2 {
		t.Error("una reevaluación inválida se rechaza")
	}
	if assessment.Snapshot().Level != domain.LevelHigh {
		t.Error("una reevaluación rechazada no cambia el RISK")
	}
}
