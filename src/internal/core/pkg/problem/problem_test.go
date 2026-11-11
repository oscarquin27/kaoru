package problem_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/problem"
)

func TestStatusOf(t *testing.T) {
	cases := map[axis.Code]int{
		axis.InvalidFormat:       400,
		axis.InvalidRequest:      422,
		axis.NotAuthenticated:    401,
		axis.InvalidToken:        401,
		axis.PermissionDenied:    403,
		axis.NotFoundResource:    404,
		axis.DuplicateResource:   409,
		axis.Conflict:            409,
		axis.PreconditionFailed:  409,
		axis.InvalidAction:       409,
		axis.ResourceExhausted:   413,
		axis.NotSupported:        415,
		axis.ServiceUnavailable:  503,
		axis.Timeout:             504,
		axis.Canceled:            499,
		axis.InternalServerError: 500,
		axis.Code("DESCONOCIDO"): 500,
	}
	for code, want := range cases {
		if got := problem.StatusOf(code); got != want {
			t.Errorf("StatusOf(%s) = %d, quiero %d", code, got, want)
		}
	}
}

func TestFrom(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantDetail string
		wantFields int
	}{
		{"4xx lleva el mensaje del error", axis.New(axis.InvalidAction, "No se puede."), 409, "INVALID_ACTION", "No se puede.", 0},
		{"422 lleva los campos", axis.Invalid(axis.Required("title", "Escribe un título.")), 422, "INVALID_REQUEST", "Revisa los datos marcados.", 1},
		{"envuelto conserva el código", fmt.Errorf("contexto: %w", axis.New(axis.NotFoundResource, "No existe.")), 404, "NOT_FOUND_RESOURCE", "No existe.", 0},
		{"cancelado es 499", context.Canceled, 499, "CANCELED", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := problem.From(tc.err, "traza")
			if body.Status != tc.wantStatus || body.Code != tc.wantCode || body.Detail != tc.wantDetail ||
				len(body.Errors) != tc.wantFields || body.TraceID != "traza" || body.Type != "about:blank" {
				t.Errorf("cuerpo inesperado: %+v", body)
			}
		})
	}
}

func TestFromNeverLeaksInternalMessages(t *testing.T) {
	secret := "pq: password authentication failed for user kaoru_app"
	for _, err := range []error{
		errors.New(secret),
		axis.Wrap(axis.InternalServerError, secret, errors.New(secret)),
		axis.Wrap(axis.ServiceUnavailable, secret, errors.New(secret)),
	} {
		body := problem.From(err, "traza")
		if body.Status < 500 || body.Detail == secret || body.Detail == "" {
			t.Errorf("un 5xx debe llevar un texto genérico, obtuve %+v", body)
		}
	}
}
