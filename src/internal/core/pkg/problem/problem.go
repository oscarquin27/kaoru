// Package problem traduce errores con código a respuestas RFC 9457.
package problem

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
)

// TraceIDKey es la llave del gin.Context donde vive el id de la petición.
const TraceIDKey = "trace_id"

// StatusClientClosed es el código de una petición que el cliente abandonó.
const StatusClientClosed = 499

type entry struct {
	status int
	title  string
}

var table = map[axis.Code]entry{
	axis.InvalidFormat:       {http.StatusBadRequest, "Formato no válido"},
	axis.InvalidRequest:      {http.StatusUnprocessableEntity, "Datos no válidos"},
	axis.NotAuthenticated:    {http.StatusUnauthorized, "Sesión requerida"},
	axis.InvalidToken:        {http.StatusUnauthorized, "Sesión no válida"},
	axis.PermissionDenied:    {http.StatusForbidden, "Sin permiso"},
	axis.NotFoundResource:    {http.StatusNotFound, "No encontrado"},
	axis.DuplicateResource:   {http.StatusConflict, "Ya existe"},
	axis.Conflict:            {http.StatusConflict, "Conflicto"},
	axis.PreconditionFailed:  {http.StatusConflict, "Condición no cumplida"},
	axis.InvalidAction:       {http.StatusConflict, "Acción no permitida"},
	axis.ResourceExhausted:   {http.StatusRequestEntityTooLarge, "Demasiado grande"},
	axis.NotSupported:        {http.StatusUnsupportedMediaType, "Tipo no admitido"},
	axis.ServiceUnavailable:  {http.StatusServiceUnavailable, "Servicio no disponible"},
	axis.Timeout:             {http.StatusGatewayTimeout, "Tiempo agotado"},
	axis.Canceled:            {StatusClientClosed, "Petición cancelada"},
	axis.InternalServerError: {http.StatusInternalServerError, "Error interno"},
}

// StatusOf es la única tabla de código de error a HTTP.
func StatusOf(code axis.Code) int {
	if e, ok := table[code]; ok {
		return e.status
	}
	return http.StatusInternalServerError
}

// FieldProblem es un error de campo en la respuesta.
type FieldProblem struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Body es el cuerpo application/problem+json.
type Body struct {
	Type    string         `json:"type"`
	Title   string         `json:"title"`
	Status  int            `json:"status"`
	Code    string         `json:"code"`
	Detail  string         `json:"detail,omitempty"`
	TraceID string         `json:"traceId"`
	Errors  []FieldProblem `json:"errors,omitempty"`
}

// From arma el cuerpo. En un 5xx nunca incluye el mensaje interno.
func From(err error, traceID string) Body {
	code := axis.CodeOf(err)
	e, ok := table[code]
	if !ok {
		code, e = axis.InternalServerError, table[axis.InternalServerError]
	}
	body := Body{Type: "about:blank", Title: e.title, Status: e.status, Code: string(code), TraceID: traceID}
	if e.status >= http.StatusInternalServerError {
		body.Detail = "No se pudo completar la operación. Intenta de nuevo; si persiste, avisa con este código de traza."
		return body
	}
	var ge *axis.GeneralError
	if errors.As(err, &ge) {
		body.Detail = ge.Error()
		for _, f := range ge.Fields() {
			body.Errors = append(body.Errors, FieldProblem{Field: f.Field, Code: f.Code, Message: f.Message})
		}
	}
	return body
}

// Write responde con el problema y deja el error en el contexto para el log.
func Write(c *gin.Context, err error) {
	_ = c.Error(err)
	body := From(err, c.GetString(TraceIDKey))
	// gin respeta un Content-Type ya puesto al escribir el JSON.
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(body.Status, body)
}
