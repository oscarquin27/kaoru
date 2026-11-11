// Package axis es el vocabulario de errores: todo error que viaja lleva un código.
package axis

import (
	"context"
	"errors"
)

// Code identifica la clase de un error. problem.StatusOf lo traduce a HTTP.
type Code string

const (
	InvalidFormat       Code = "INVALID_FORMAT"
	InvalidRequest      Code = "INVALID_REQUEST"
	NotAuthenticated    Code = "NOT_AUTHENTICATED"
	InvalidToken        Code = "INVALID_TOKEN"
	PermissionDenied    Code = "PERMISSION_DENIED"
	NotFoundResource    Code = "NOT_FOUND_RESOURCE"
	DuplicateResource   Code = "DUPLICATE_RESOURCE"
	Conflict            Code = "CONFLICT"
	PreconditionFailed  Code = "PRECONDITION_FAILED"
	InvalidAction       Code = "INVALID_ACTION"
	ResourceExhausted   Code = "RESOURCE_EXHAUSTED"
	NotSupported        Code = "NOT_SUPPORTED"
	ServiceUnavailable  Code = "SERVICE_UNAVAILABLE"
	Timeout             Code = "TIMEOUT"
	Canceled            Code = "CANCELED"
	InternalServerError Code = "INTERNAL_SERVER_ERROR"
)

// Códigos de un error de campo.
const (
	FieldRequired = "required"
	FieldInvalid  = "invalid"
)

// FieldError es un error de validación de un campo.
type FieldError struct {
	Field   string
	Code    string
	Message string
}

// Required arma el error de un campo que falta.
func Required(field, message string) FieldError {
	return FieldError{Field: field, Code: FieldRequired, Message: message}
}

// InvalidField arma el error de un campo con un valor que no vale.
func InvalidField(field, message string) FieldError {
	return FieldError{Field: field, Code: FieldInvalid, Message: message}
}

// GeneralError es el error con código. El mensaje es apto para el usuario;
// la causa técnica viaja aparte y nunca se muestra.
type GeneralError struct {
	code    Code
	message string
	fields  []FieldError
	cause   error
}

// New crea un error con código y mensaje para el usuario.
func New(code Code, message string) *GeneralError {
	return &GeneralError{code: code, message: message}
}

// Wrap crea un error con código que conserva la causa técnica.
func Wrap(code Code, message string, cause error) *GeneralError {
	return &GeneralError{code: code, message: message, cause: cause}
}

// Invalid crea un INVALID_REQUEST con la lista de campos que fallaron.
func Invalid(fields ...FieldError) *GeneralError {
	return &GeneralError{code: InvalidRequest, message: "Revisa los datos marcados.", fields: fields}
}

func (e *GeneralError) Error() string { return e.message }

// Unwrap devuelve la causa técnica.
func (e *GeneralError) Unwrap() error { return e.cause }

// Code devuelve el código del error.
func (e *GeneralError) Code() Code { return e.code }

// Fields devuelve los errores de campo de un INVALID_REQUEST.
func (e *GeneralError) Fields() []FieldError { return e.fields }

// CodeOf devuelve el código de err, aunque venga envuelto.
// Un error sin código es INTERNAL_SERVER_ERROR.
func CodeOf(err error) Code {
	var ge *GeneralError
	switch {
	case errors.As(err, &ge):
		return ge.code
	case errors.Is(err, context.Canceled):
		return Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return Timeout
	default:
		return InternalServerError
	}
}

// IsCode dice si err lleva ese código.
func IsCode(err error, code Code) bool {
	return err != nil && CodeOf(err) == code
}

// Fields junta errores de campo mientras se valida una entidad.
type FieldErrors []FieldError

// Add agrega un error de campo.
func (f *FieldErrors) Add(fe FieldError) { *f = append(*f, fe) }

// Err devuelve nil si no hubo errores, o un INVALID_REQUEST con todos.
func (f FieldErrors) Err() error {
	if len(f) == 0 {
		return nil
	}
	return Invalid(f...)
}
