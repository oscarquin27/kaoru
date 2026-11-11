// Package postgres da la unidad de trabajo, la traducción de errores y las migraciones.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"kaoru/internal/core/pkg/uow"
)

// DBTX es lo que necesitan las consultas: lo cumplen el pool y una transacción.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// Database es el pool con su unidad de trabajo.
type Database struct {
	pool *pgxpool.Pool
}

var _ uow.Factory = (*Database)(nil)

// NewDatabase envuelve el pool.
func NewDatabase(pool *pgxpool.Pool) *Database { return &Database{pool: pool} }

// Within ejecuta fn en una transacción. Si el context ya trae una, la reutiliza.
func (d *Database) Within(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return TranslateError(err)
	}
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		// El rollback usa un context propio: el de la petición puede estar cancelado.
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return err
	}
	return TranslateError(tx.Commit(ctx))
}

// Conn devuelve la transacción del context o, si no hay, el pool.
func (d *Database) Conn(ctx context.Context) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return d.pool
}

// Ping comprueba que la base responde.
func (d *Database) Ping(ctx context.Context) error { return d.pool.Ping(ctx) }
