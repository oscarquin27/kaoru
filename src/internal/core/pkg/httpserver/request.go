package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
)

// DecodeJSON lee el cuerpo en dst. Un JSON ilegible es INVALID_FORMAT
// y un cuerpo que pasa el límite es RESOURCE_EXHAUSTED.
func DecodeJSON(c *gin.Context, dst any) error {
	if err := c.ShouldBindJSON(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return axis.Wrap(axis.ResourceExhausted, "La petición es demasiado grande.", err)
		}
		return axis.Wrap(axis.InvalidFormat, "No se pudo leer la petición.", err)
	}
	return nil
}

// Paginación de las listas.
const (
	DefaultPageSize = 25
	MaxPageSize     = 200
)

// PageParams lee `page` y `size` de la consulta.
func PageParams(c *gin.Context) (page, size int, err error) {
	page, size = 1, DefaultPageSize
	if raw := c.Query("page"); raw != "" {
		if page, err = strconv.Atoi(raw); err != nil || page < 1 {
			return 0, 0, axis.Invalid(axis.InvalidField("page", "La página debe ser un número desde 1."))
		}
	}
	if raw := c.Query("size"); raw != "" {
		if size, err = strconv.Atoi(raw); err != nil || size < 1 || size > MaxPageSize {
			return 0, 0, axis.Invalid(axis.InvalidField("size", "El tamaño debe estar entre 1 y 200."))
		}
	}
	return page, size, nil
}
