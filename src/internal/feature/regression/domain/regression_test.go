package domain_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/regression/domain"
	shared "kaoru/internal/shared/domain"
)

var (
	t0 = time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)
)

func ref(n string) domain.TestCaseRef {
	return domain.TestCaseRef{Id: "id-" + n, Code: "TC-2026-000" + n}
}

func candidate(n string, origin domain.Origin) domain.Candidate {
	return domain.Candidate{TestCase: ref(n), Origin: origin}
}

// proposed es un TC propuesto que QA no tocó.
func proposed(n string, origin domain.Origin) domain.Item {
	return domain.Item{TestCase: ref(n), Origin: origin, Proposed: true, Included: true}
}

// removed es un TC propuesto que QA quitó con su criterio.
func removed(n string, origin domain.Origin) domain.Item {
	return domain.Item{TestCase: ref(n), Origin: origin, Proposed: true, Reason: "No aplica", DecidedBy: "sub-qa", DecidedAt: &t0}
}

// manual es un TC que agregó QA.
func manual(n string) domain.Item {
	return domain.Item{TestCase: ref(n), Origin: domain.OriginManual, Included: true, Reason: "Zona frágil", DecidedBy: "sub-qa", DecidedAt: &t0}
}

func TestProposeMergesTheProposalWithWhatQADecided(t *testing.T) {
	cases := []struct {
		name       string
		current    []domain.Item
		candidates []domain.Candidate
		want       []domain.Item
	}{
		{
			name:       "la primera propuesta entra incluida, por código de TC",
			candidates: []domain.Candidate{candidate("2", domain.OriginCritical), candidate("1", domain.OriginImpact)},
			want:       []domain.Item{proposed("1", domain.OriginImpact), proposed("2", domain.OriginCritical)},
		},
		{
			name: "sin nada que proponer, la selección queda vacía",
			want: []domain.Item{},
		},
		{
			name:       "un candidato repetido cuenta una vez, con su primer origen",
			candidates: []domain.Candidate{candidate("1", domain.OriginImpact), candidate("1", domain.OriginCritical)},
			want:       []domain.Item{proposed("1", domain.OriginImpact)},
		},
		{
			name:       "lo que sigue propuesto conserva el ajuste de QA",
			current:    []domain.Item{proposed("1", domain.OriginImpact), removed("2", domain.OriginImpact)},
			candidates: []domain.Candidate{candidate("1", domain.OriginImpact), candidate("2", domain.OriginImpact)},
			want:       []domain.Item{proposed("1", domain.OriginImpact), removed("2", domain.OriginImpact)},
		},
		{
			name:       "lo que sigue propuesto toma el origen nuevo sin perder el ajuste",
			current:    []domain.Item{removed("1", domain.OriginCritical), proposed("2", domain.OriginImpact)},
			candidates: []domain.Candidate{candidate("1", domain.OriginImpact), candidate("2", domain.OriginCritical)},
			want:       []domain.Item{removed("1", domain.OriginImpact), proposed("2", domain.OriginCritical)},
		},
		{
			name:       "un propuesto nuevo entra incluido junto a los que ya estaban",
			current:    []domain.Item{removed("1", domain.OriginImpact)},
			candidates: []domain.Candidate{candidate("1", domain.OriginImpact), candidate("3", domain.OriginImpact)},
			want:       []domain.Item{removed("1", domain.OriginImpact), proposed("3", domain.OriginImpact)},
		},
		{
			name:       "lo que ya no se propone y QA no tocó sale",
			current:    []domain.Item{proposed("1", domain.OriginImpact), proposed("2", domain.OriginCritical)},
			candidates: []domain.Candidate{candidate("2", domain.OriginCritical)},
			want:       []domain.Item{proposed("2", domain.OriginCritical)},
		},
		{
			name:    "lo que ya no se propone y QA había quitado sale: nadie lo quiere",
			current: []domain.Item{removed("1", domain.OriginImpact)},
			want:    []domain.Item{},
		},
		{
			name: "lo que ya no se propone pero QA decidió conservar pasa a manual con su decisión",
			current: []domain.Item{
				{TestCase: ref("1"), Origin: domain.OriginImpact, Proposed: true, Included: true, Reason: "Imprescindible", DecidedBy: "sub-qa", DecidedAt: &t0},
			},
			want: []domain.Item{
				{TestCase: ref("1"), Origin: domain.OriginManual, Included: true, Reason: "Imprescindible", DecidedBy: "sub-qa", DecidedAt: &t0},
			},
		},
		{
			name: "un ajuste sin criterio no puede quedar como agregado de QA: sale",
			current: []domain.Item{
				{TestCase: ref("1"), Origin: domain.OriginImpact, Proposed: true, Included: true, DecidedBy: "sub-qa", DecidedAt: &t0},
			},
			want: []domain.Item{},
		},
		{
			name:       "lo que agregó QA se conserva aunque no se proponga",
			current:    []domain.Item{manual("5")},
			candidates: []domain.Candidate{candidate("1", domain.OriginImpact)},
			want:       []domain.Item{proposed("1", domain.OriginImpact), manual("5")},
		},
		{
			name:       "lo que agregó QA y ahora se propone deja de ser un ajuste",
			current:    []domain.Item{manual("5")},
			candidates: []domain.Candidate{candidate("5", domain.OriginImpact)},
			want:       []domain.Item{proposed("5", domain.OriginImpact)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]domain.Item(nil), tc.current...)
			got := domain.Propose(tc.current, tc.candidates)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Propose:\n got %+v\nwant %+v", got, tc.want)
			}
			if !reflect.DeepEqual(tc.current, before) {
				t.Errorf("Propose no debe cambiar lo que recibe: %+v", tc.current)
			}
			// Proponer dos veces lo mismo no cambia nada más.
			if again := domain.Propose(got, tc.candidates); !reflect.DeepEqual(again, got) {
				t.Errorf("volver a proponer lo mismo:\n got %+v\nwant %+v", again, got)
			}
			// Lo que queda siempre cumple la restricción de la base.
			for _, item := range got {
				if item.Included != item.Proposed && item.Reason == "" {
					t.Errorf("%s difiere de lo propuesto sin criterio", item.TestCase.Code)
				}
			}
		})
	}
}

func TestAdjust(t *testing.T) {
	active := domain.TestCase{Ref: ref("1"), SystemKey: "SYPAGO"}
	decided := func(item domain.Item, included bool, reason string) domain.Item {
		item.Included, item.Reason, item.DecidedBy, item.DecidedAt = included, reason, "sub-admin", &t1
		return item
	}
	undecided := func(item domain.Item) domain.Item {
		item.Included, item.Reason, item.DecidedBy, item.DecidedAt = true, "", "", nil
		return item
	}
	item := func(i domain.Item) *domain.Item { return &i }
	cases := []struct {
		name      string
		current   *domain.Item
		testCase  domain.TestCase
		included  bool
		reason    string
		want      domain.Adjustment
		wantField string
		wantErr   error
	}{
		{
			name: "quitar un TC propuesto exige el criterio", current: item(proposed("1", domain.OriginImpact)),
			testCase: active, included: false, reason: "  ", wantField: "reason",
		},
		{
			name: "quitar un TC propuesto guarda criterio, autor y fecha", current: item(proposed("1", domain.OriginImpact)),
			testCase: active, included: false, reason: "  Cubierto por TC-2026-0002  ",
			want: domain.Adjustment{Item: decided(proposed("1", domain.OriginImpact), false, "Cubierto por TC-2026-0002"), Changed: true},
		},
		{
			name: "incluir un TC propuesto que ya estaba incluido no cambia nada", current: item(proposed("1", domain.OriginCritical)),
			testCase: active, included: true, reason: "da igual",
			want: domain.Adjustment{Item: proposed("1", domain.OriginCritical)},
		},
		{
			name: "volver a incluir un TC propuesto borra el criterio y el autor", current: item(removed("1", domain.OriginImpact)),
			testCase: active, included: true, reason: "ya no hace falta",
			want: domain.Adjustment{Item: undecided(removed("1", domain.OriginImpact)), Changed: true},
		},
		{
			name: "repetir la misma exclusión con el mismo criterio no cambia nada", current: item(removed("1", domain.OriginImpact)),
			testCase: active, included: false, reason: " No aplica ",
			want: domain.Adjustment{Item: removed("1", domain.OriginImpact)},
		},
		{
			name: "cambiar el criterio de una exclusión se guarda con su nuevo autor", current: item(removed("1", domain.OriginImpact)),
			testCase: active, included: false, reason: "Duplicado",
			want: domain.Adjustment{Item: decided(removed("1", domain.OriginImpact), false, "Duplicado"), Changed: true},
		},
		{
			name: "agregar un TC que no estaba exige el criterio", testCase: active, included: true, wantField: "reason",
		},
		{
			name: "agregar un TC que no estaba entra como manual", testCase: active, included: true, reason: "Zona frágil",
			want: domain.Adjustment{
				Item:    domain.Item{TestCase: ref("1"), Origin: domain.OriginManual, Included: true, Reason: "Zona frágil", DecidedBy: "sub-admin", DecidedAt: &t1},
				Changed: true,
			},
		},
		{
			name: "excluir un TC que no estaba no hace nada", testCase: active, included: false, reason: "x",
			want: domain.Adjustment{},
		},
		{
			name: "quitar un TC agregado por QA lo saca de la selección", current: item(manual("1")),
			testCase: active, included: false,
			want: domain.Adjustment{Changed: true, Removed: true},
		},
		{
			name: "un TC agregado por QA no se queda sin criterio", current: item(manual("1")),
			testCase: active, included: true, reason: "", wantField: "reason",
		},
		{
			name:     "un TC de otro sistema no se incluye",
			testCase: domain.TestCase{Ref: ref("1"), SystemKey: "SIMF"}, included: true, reason: "x", wantErr: domain.ErrOtherSystem,
		},
		{
			name:     "un TC retirado no se incluye",
			testCase: domain.TestCase{Ref: ref("1"), SystemKey: "SYPAGO", Retired: true}, included: true, reason: "x", wantErr: domain.ErrTestCaseRetired,
		},
		{
			name: "un TC retirado que estaba propuesto sí se puede quitar", current: item(proposed("1", domain.OriginImpact)),
			testCase: domain.TestCase{Ref: ref("1"), SystemKey: "SYPAGO", Retired: true}, included: false, reason: "Retirado",
			want: domain.Adjustment{Item: decided(proposed("1", domain.OriginImpact), false, "Retirado"), Changed: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.Adjust(tc.current, tc.testCase, "SYPAGO", domain.Decision{
				Included: tc.included, Reason: tc.reason, By: "sub-admin", At: t1,
			})
			switch {
			case tc.wantField != "":
				var general *axis.GeneralError
				if !errors.As(err, &general) || general.Code() != axis.InvalidRequest || len(general.Fields()) != 1 ||
					general.Fields()[0].Field != tc.wantField || general.Fields()[0].Code != "required" {
					t.Fatalf("quiero un error de campo %s required, obtuve %v", tc.wantField, err)
				}
			case tc.wantErr != nil:
				if !errors.Is(err, tc.wantErr) || axis.CodeOf(err) != axis.InvalidAction {
					t.Fatalf("error = %v, quiero %v", err, tc.wantErr)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("Adjust:\n got %+v\nwant %+v", got, tc.want)
				}
			}
		})
	}
}

func TestSelectionIsProposedConfirmedAndReopened(t *testing.T) {
	s := domain.NewSelection("pkg-1", "sub-qa", t0)
	if s.Confirmed() || s.ProposedBy != "sub-qa" || !s.ProposedAt.Equal(t0) || s.PackageId != "pkg-1" {
		t.Fatalf("selección nueva: %+v", s)
	}
	if s.Reopen() {
		t.Error("no hay confirmación que quitar")
	}
	if err := s.Propose("sub-admin", t1); err != nil || s.ProposedBy != "sub-admin" || !s.ProposedAt.Equal(t1) {
		t.Errorf("volver a proponer: %v %+v", err, s)
	}
	if err := s.Confirm("sub-qa", t1); err != nil || !s.Confirmed() || s.ConfirmedBy != "sub-qa" || !s.ConfirmedAt.Equal(t1) {
		t.Fatalf("confirmar: %v %+v", err, s)
	}
	if err := s.Confirm("sub-qa", t1); !errors.Is(err, domain.ErrAlreadyConfirmed) || axis.CodeOf(err) != axis.InvalidAction {
		t.Errorf("confirmar dos veces: %v", err)
	}
	if err := s.Propose("sub-admin", t1.Add(time.Hour)); !errors.Is(err, domain.ErrConfirmed) || s.ProposedBy != "sub-admin" || !s.ProposedAt.Equal(t1) {
		t.Errorf("confirmada no se vuelve a proponer: %v %+v", err, s)
	}
	if !s.Reopen() || s.Confirmed() || s.ConfirmedBy != shared.UserId("") {
		t.Errorf("reabrir quita la confirmación: %+v", s)
	}
	if err := s.Confirm("sub-admin", t1); err != nil {
		t.Errorf("reabierta se puede confirmar otra vez: %v", err)
	}
}

func TestSummarizeAndUnpassed(t *testing.T) {
	items := []domain.Progress{
		{Item: proposed("1", domain.OriginImpact), LastResult: "passed"},
		{Item: proposed("2", domain.OriginImpact), LastResult: "failed"},
		{Item: proposed("3", domain.OriginCritical), LastResult: "blocked"},
		{Item: proposed("4", domain.OriginCritical)},
		{Item: removed("5", domain.OriginImpact), LastResult: "failed"}, // quitado: no cuenta en el avance
		{Item: manual("6"), LastResult: "passed"},
		{Item: manual("7")},
	}
	want := domain.Summary{Proposed: 5, Included: 6, Added: 2, Removed: 1, Passed: 2, Failed: 2, Pending: 2}
	if got := domain.Summarize(items); got != want {
		t.Errorf("Summarize = %+v, quiero %+v", got, want)
	}
	if got := domain.Summarize(nil); got != (domain.Summary{}) {
		t.Errorf("sin TC = %+v", got)
	}
	wantCodes := []string{"TC-2026-0002", "TC-2026-0003", "TC-2026-0004", "TC-2026-0007"}
	if got := domain.Unpassed(items); !reflect.DeepEqual(got, wantCodes) {
		t.Errorf("Unpassed = %v, quiero %v", got, wantCodes)
	}
	if got := domain.Unpassed(items[:1]); got != nil {
		t.Errorf("todo aprobado = %v", got)
	}
}
