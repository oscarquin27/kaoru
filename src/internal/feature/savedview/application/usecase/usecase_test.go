package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/savedview/application/usecase"
	"kaoru/internal/feature/savedview/domain"
	shared "kaoru/internal/shared/domain"
)

var now = time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return now }

// store es el repositorio en memoria; también hace de unidad de trabajo y de generador de ids.
type store struct {
	views     []domain.View
	sequence  int
	insertErr error
}

func (s *store) Within(ctx context.Context, fn func(context.Context) error) error {
	saved := slices.Clone(s.views)
	if err := fn(ctx); err != nil {
		s.views = saved
		return err
	}
	return nil
}

func (s *store) NewId() (string, error) {
	s.sequence++
	return fmt.Sprintf("v%d", s.sequence), nil
}

func (s *store) Insert(_ context.Context, v domain.View) error {
	if s.insertErr != nil {
		return s.insertErr
	}
	if slices.ContainsFunc(s.views, func(o domain.View) bool {
		return o.OwnerId == v.OwnerId && o.Screen == v.Screen && o.Name == v.Name
	}) {
		return axis.New(axis.DuplicateResource, "Ya existe.")
	}
	s.views = append(s.views, v)
	return nil
}

func (s *store) List(_ context.Context, owner shared.UserId, screen string) ([]domain.View, error) {
	var out []domain.View
	for _, v := range s.views {
		if v.OwnerId == owner && v.Screen == screen {
			out = append(out, v)
		}
	}
	return out, nil
}

func (s *store) Count(ctx context.Context, owner shared.UserId, screen string) (int, error) {
	views, err := s.List(ctx, owner, screen)
	return len(views), err
}

func (s *store) Delete(_ context.Context, owner shared.UserId, id string) error {
	before := len(s.views)
	s.views = slices.DeleteFunc(s.views, func(v domain.View) bool { return v.Id == id && v.OwnerId == owner })
	if len(s.views) == before {
		return domain.ErrNotFound
	}
	return nil
}

func build(t *testing.T, s *store) *usecase.UseCases {
	t.Helper()
	uc, err := usecase.New(usecase.Deps{Repository: s, UnitOfWork: s, Clock: fixedClock{}, IdGenerator: s})
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

var ctx = context.Background()

func TestNewRequiresDependencies(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New sin dependencias debía fallar")
	}
}

func TestSaveView(t *testing.T) {
	s := &store{}
	uc := build(t, s)
	in := usecase.SaveViewInput{Owner: "ana", Screen: "requirements", Name: "En pruebas", Query: "?state=in_testing"}
	view, err := uc.SaveView.Execute(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if view.Id != "v1" || view.Query != "state=in_testing" || !view.CreatedAt.Equal(now) || len(s.views) != 1 {
		t.Fatalf("vista guardada inesperada: %+v", view)
	}

	// El nombre no se repite en la misma lista de la misma persona; sí en otra lista o de otra persona.
	if _, err := uc.SaveView.Execute(ctx, in); !axis.IsCode(err, axis.InvalidRequest) || len(s.views) != 1 {
		t.Errorf("nombre repetido: quiero INVALID_REQUEST en el campo, obtuve %v", err)
	}
	other := in
	other.Screen = "defects"
	if _, err := uc.SaveView.Execute(ctx, other); err != nil {
		t.Errorf("mismo nombre en otra lista: %v", err)
	}
	other = in
	other.Owner = "luis"
	if _, err := uc.SaveView.Execute(ctx, other); err != nil {
		t.Errorf("mismo nombre de otra persona: %v", err)
	}
}

func TestSaveViewRejectsInvalidInputAndTheLimit(t *testing.T) {
	s := &store{}
	uc := build(t, s)
	if _, err := uc.SaveView.Execute(ctx, usecase.SaveViewInput{Owner: "ana", Screen: "nope"}); !axis.IsCode(err, axis.InvalidRequest) || len(s.views) != 0 {
		t.Errorf("entrada inválida: quiero INVALID_REQUEST sin guardar, obtuve %v", err)
	}
	for i := range domain.MaxPerScreen {
		in := usecase.SaveViewInput{Owner: "ana", Screen: "matrix", Name: fmt.Sprintf("Vista %d", i), Query: "risk=high"}
		if _, err := uc.SaveView.Execute(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	_, err := uc.SaveView.Execute(ctx, usecase.SaveViewInput{Owner: "ana", Screen: "matrix", Name: "Una más", Query: "risk=high"})
	if !errors.Is(err, domain.ErrTooMany) || len(s.views) != domain.MaxPerScreen {
		t.Errorf("tope por lista: quiero ErrTooMany, obtuve %v", err)
	}
	boom := errors.New("sin base")
	s = &store{insertErr: boom}
	if _, err := build(t, s).SaveView.Execute(ctx, usecase.SaveViewInput{Owner: "ana", Screen: "matrix", Name: "x", Query: "a=1"}); !errors.Is(err, boom) {
		t.Errorf("quiero el error del repositorio, obtuve %v", err)
	}
}

func TestListAndDeleteAreScopedToTheOwner(t *testing.T) {
	s := &store{}
	uc := build(t, s)
	mine, _ := uc.SaveView.Execute(ctx, usecase.SaveViewInput{Owner: "ana", Screen: "defects", Name: "Altos", Query: "severity=high"})
	_, _ = uc.SaveView.Execute(ctx, usecase.SaveViewInput{Owner: "luis", Screen: "defects", Name: "Míos", Query: "assignee=luis"})

	views, err := uc.ListViews.Execute(ctx, "ana", "defects")
	if err != nil || len(views) != 1 || views[0].Name != "Altos" {
		t.Fatalf("cada quien ve solo sus vistas: %+v, %v", views, err)
	}
	if _, err := uc.ListViews.Execute(ctx, "ana", "nope"); !axis.IsCode(err, axis.InvalidRequest) {
		t.Errorf("lista desconocida: quiero INVALID_REQUEST, obtuve %v", err)
	}
	if err := uc.DeleteView.Execute(ctx, "luis", mine.Id); !errors.Is(err, domain.ErrNotFound) || len(s.views) != 2 {
		t.Errorf("nadie quita la vista de otro: %v", err)
	}
	if err := uc.DeleteView.Execute(ctx, "ana", mine.Id); err != nil || len(s.views) != 1 {
		t.Errorf("el dueño quita la suya: %v", err)
	}
}
