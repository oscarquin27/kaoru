package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/comment/application/usecase"
	"kaoru/internal/feature/comment/domain"
	shared "kaoru/internal/shared/domain"
)

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
type world struct {
	comments []domain.Comment // en el orden en que se escribieron
	audit    []domain.Comment
	auditErr error
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	comments, audit := slices.Clone(w.comments), slices.Clone(w.audit)
	if err := fn(ctx); err != nil {
		w.comments, w.audit = comments, audit
		return err
	}
	return nil
}

var (
	notFound = axis.New(axis.NotFoundResource, "No existe.")
	down     = axis.New(axis.ServiceUnavailable, "Sin servicio.")
)

func (w *world) Insert(_ context.Context, c domain.Comment) error {
	w.comments = append(w.comments, c)
	return nil
}

func (w *world) List(_ context.Context, entity domain.EntityRef) ([]domain.Comment, error) {
	var out []domain.Comment
	for _, c := range w.comments {
		if c.Entity == entity {
			out = append(out, c)
		}
	}
	return out, nil
}

func (w *world) FindUserName(_ context.Context, id shared.UserId) (string, error) {
	if !strings.HasPrefix(string(id), "u") {
		return "", notFound
	}
	return "Usuario " + string(id), nil
}

// Resolve hace de port.Entities; el tipo «down» simula que el otro feature no responde.
func (w *world) Resolve(_ context.Context, entityType, code string) (domain.EntityRef, error) {
	if entityType == "down" {
		return domain.EntityRef{}, down
	}
	entity, ok := entities[entityType+"/"+code]
	if !ok {
		return domain.EntityRef{}, notFound
	}
	return entity, nil
}

func (w *world) RecordComment(_ context.Context, c domain.Comment) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	w.audit = append(w.audit, c)
	return nil
}

type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	if permission == usecase.PermissionCreate && slices.Contains(roles, "qa") {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return t0 }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	t0          = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	qa          = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	viewer      = shared.Actor{Id: "u3", Roles: []string{"viewer"}}
	ghost       = shared.Actor{Id: "x1", Roles: []string{"qa"}} // con permiso, pero sin usuario
	ctx         = context.Background()
	requirement = domain.EntityRef{Type: "requirement", Id: "r1", Code: "REQ-2026-0042"}
	defect      = domain.EntityRef{Type: "defect", Id: "d1", Code: "DEF-2026-0001"}
	entities    = map[string]domain.EntityRef{"requirement/REQ-2026-0042": requirement, "defect/DEF-2026-0001": defect}
)

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{}
	uc, err := usecase.New(usecase.Deps{
		Repository: w, Entities: w, Permissions: fakePermissions{}, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{}, IdGenerator: &seqIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

func comment(uc *usecase.UseCases, actor shared.Actor, entity domain.EntityRef, body string) (domain.Comment, error) {
	return uc.CreateComment.Execute(ctx, usecase.CreateCommentInput{Actor: actor, EntityType: entity.Type, EntityCode: entity.Code, Body: body})
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

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
}

func TestCreateComment(t *testing.T) {
	uc, w := setup(t)
	out, err := comment(uc, qa, requirement, "  Falta el criterio de aceptación. ")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Comment{
		Id: "id-1", Entity: requirement, Body: "Falta el criterio de aceptación.",
		AuthorId: "u2", AuthorName: "Usuario u2", CreatedAt: t0,
	}
	if out != want || !slices.Equal(w.comments, []domain.Comment{want}) {
		t.Errorf("comentario guardado inesperado: %+v / %+v", out, w.comments)
	}
	if !slices.Equal(w.audit, []domain.Comment{want}) {
		t.Errorf("la auditoría debe dejar constancia del comentario: %+v", w.audit)
	}
}

func TestCreateCommentErrors(t *testing.T) {
	cases := []struct {
		name       string
		actor      shared.Actor
		entityType string
		entityCode string
		body       string
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", viewer, "requirement", "REQ-2026-0042", "Hola", axis.PermissionDenied, nil},
		{"entidad inexistente", qa, "requirement", "REQ-2026-0099", "Hola", axis.InvalidRequest, []string{"entityCode:invalid"}},
		{"entidad que no admite comentarios", qa, "user", "REQ-2026-0042", "Hola", axis.InvalidRequest, []string{"entityCode:invalid"}},
		{"sin entidad", qa, "", "", "Hola", axis.InvalidRequest, []string{"entityCode:invalid"}},
		{"no se pudo resolver la entidad", qa, "down", "REQ-2026-0042", "Hola", axis.ServiceUnavailable, nil},
		{"actor sin usuario", ghost, "requirement", "REQ-2026-0042", "Hola", axis.NotFoundResource, nil},
		{"sin texto", qa, "requirement", "REQ-2026-0042", " \n ", axis.InvalidRequest, []string{"body:required"}},
		{"texto demasiado largo", qa, "requirement", "REQ-2026-0042", strings.Repeat("ñ", 5001), axis.InvalidRequest, []string{"body:invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			_, err := uc.CreateComment.Execute(ctx, usecase.CreateCommentInput{
				Actor: tc.actor, EntityType: tc.entityType, EntityCode: tc.entityCode, Body: tc.body,
			})
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.comments) != 0 || len(w.audit) != 0 {
				t.Error("un comentario rechazado no se guarda ni se audita")
			}
		})
	}

	// El tope se cuenta en caracteres, no en bytes.
	uc, _ := setup(t)
	if _, err := comment(uc, qa, requirement, strings.Repeat("ñ", 5000)); err != nil {
		t.Errorf("5000 caracteres caben: %v", err)
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	if _, err := comment(uc, qa, requirement, "Primero"); err != nil {
		t.Fatal(err)
	}
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")
	if out, err := comment(uc, qa, requirement, "Segundo"); !axis.IsCode(err, axis.ServiceUnavailable) || out != (domain.Comment{}) {
		t.Fatalf("comentar: %+v, %v", out, err)
	}
	if len(w.comments) != 1 || w.comments[0].Body != "Primero" || len(w.audit) != 1 {
		t.Error("si la auditoría falla, el comentario no se guarda")
	}
}

func TestListComments(t *testing.T) {
	uc, _ := setup(t)
	for _, c := range []struct {
		entity domain.EntityRef
		body   string
	}{{requirement, "Primero"}, {defect, "Del defecto"}, {requirement, "Segundo"}} {
		if _, err := comment(uc, qa, c.entity, c.body); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name       string
		entityType string
		entityCode string
		want       []string
	}{
		{"los del REQ, del más antiguo al más reciente", "requirement", "REQ-2026-0042", []string{"Primero", "Segundo"}},
		{"los del DEF", "defect", "DEF-2026-0001", []string{"Del defecto"}},
		{"entidad inexistente", "requirement", "REQ-2026-0099", []string{}},
		{"entidad que no admite comentarios", "user", "u2", []string{}},
	}
	for _, tc := range cases {
		// Leer no exige permiso: ni siquiera recibe un actor.
		list, err := uc.ListComments.Execute(ctx, usecase.ListCommentsInput{EntityType: tc.entityType, EntityCode: tc.entityCode})
		got := []string{}
		for _, c := range list {
			got = append(got, c.Body)
		}
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v, %v; quiero %v", tc.name, got, err, tc.want)
		}
		if tc.name == "entidad inexistente" && list == nil {
			t.Error("sin entidad la lista es vacía, no nil")
		}
	}
	if _, err := uc.ListComments.Execute(ctx, usecase.ListCommentsInput{EntityType: "down", EntityCode: "x"}); !errors.Is(err, down) {
		t.Errorf("no se pudo resolver la entidad: %v", err)
	}
}
