package postgres

import (
	"context"
	"errors"
	"net"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"kaoru/internal/core/pkg/axis"
)

// TranslateError convierte un error de Postgres en un error con código.
// Un error que ya tiene código, o nil, pasa igual.
func TranslateError(err error) error {
	if err == nil {
		return nil
	}
	var ge *axis.GeneralError
	if errors.As(err, &ge) {
		return err
	}
	var pgErr *pgconn.PgError
	var netErr net.Error
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return axis.Wrap(axis.NotFoundResource, "No existe.", err)
	case errors.Is(err, context.Canceled):
		return axis.Wrap(axis.Canceled, "Petición cancelada.", err)
	case errors.Is(err, context.DeadlineExceeded):
		return axis.Wrap(axis.Timeout, "La operación tardó demasiado.", err)
	case errors.As(err, &pgErr):
		return translatePgError(pgErr)
	case errors.As(err, &netErr):
		return axis.Wrap(axis.ServiceUnavailable, "La base de datos no responde.", err)
	default:
		return err
	}
}

func translatePgError(pgErr *pgconn.PgError) error {
	switch pgErr.Code {
	case "23505":
		return axis.Wrap(axis.DuplicateResource, "Ya existe un registro con esos datos.", pgErr)
	case "23503", "23514", "23502":
		return axis.Wrap(axis.InvalidRequest, "Los datos no cumplen una regla de la base.", pgErr)
	case "40001", "40P01":
		return axis.Wrap(axis.ServiceUnavailable, "La operación chocó con otra. Intenta de nuevo.", pgErr)
	case "57014":
		return axis.Wrap(axis.Timeout, "La operación tardó demasiado.", pgErr)
	default:
		return pgErr
	}
}
