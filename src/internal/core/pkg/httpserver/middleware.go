// Package httpserver reúne los middlewares globales y el servidor interno de salud.
package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/logging"
	"kaoru/internal/core/pkg/problem"
)

const requestIDHeader = "X-Request-Id"

// RequestID acepta el id que manda Nginx o genera uno, y lo devuelve en la respuesta.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" || len(id) > 64 {
			buf := make([]byte, 16)
			_, _ = rand.Read(buf) // crypto/rand.Read no falla desde Go 1.24.
			id = hex.EncodeToString(buf)
		}
		c.Set(problem.TraceIDKey, id)
		c.Header(requestIDHeader, id)
		ctx := logging.WithFields(c.Request.Context(), logging.Fields{TraceID: id})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Recovery convierte un pánico en un 500 problem+json y registra la pila.
func Recovery(log logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(c.Request.Context(), "pánico en la petición",
					"panic", fmt.Sprint(r), "stack", string(debug.Stack()))
				problem.Write(c, axis.New(axis.InternalServerError, "pánico"))
			}
		}()
		c.Next()
	}
}

// AccessLog registra cada petición. Un 5xx lleva el detalle técnico; un 4xx, solo su código,
// porque su causa puede traer datos del usuario. Nunca registra cuerpos ni cabeceras.
func AccessLog(log logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		args := []any{
			"method", c.Request.Method,
			"route", c.FullPath(),
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		var err error
		if last := c.Errors.Last(); last != nil {
			err = last.Err
			args = append(args, "code", string(axis.CodeOf(err)))
		}
		ctx := c.Request.Context()
		switch {
		case status >= http.StatusInternalServerError:
			log.ErrorContext(ctx, "petición fallida", append(args, "error", detail(err))...)
		case status >= http.StatusBadRequest && status != problem.StatusClientClosed:
			log.WarnContext(ctx, "petición rechazada", args...)
		default:
			log.InfoContext(ctx, "petición atendida", args...)
		}
	}
}

// detail arma el texto técnico de un error con toda su cadena de causas.
func detail(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	for cause := errors.Unwrap(err); cause != nil; cause = errors.Unwrap(cause) {
		text += ": " + cause.Error()
	}
	return text
}

// Origin exige que toda petición que escribe traiga `Origin` igual al origen público (CSRF).
func Origin(publicOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if c.GetHeader("Origin") != publicOrigin {
				problem.Write(c, axis.New(axis.PermissionDenied, "La petición no viene de la aplicación."))
				return
			}
		}
		c.Next()
	}
}

// MaxBody limita el tamaño del cuerpo de toda petición.
func MaxBody(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
