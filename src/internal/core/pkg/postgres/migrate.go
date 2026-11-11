package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	// Registra el driver "pgx" de database/sql, que es el que usa goose.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Migrate aplica las migraciones pendientes con la conexión del dueño del esquema.
// Un bloqueo de sesión evita que dos procesos migren a la vez.
func Migrate(ctx context.Context, ownerURL string, migrations fs.FS) (applied int, err error) {
	db, err := sql.Open("pgx", ownerURL)
	if err != nil {
		return 0, fmt.Errorf("abrir conexión de migraciones: %w", err)
	}
	defer func() { _ = db.Close() }()

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return 0, fmt.Errorf("crear bloqueo de migraciones: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations, goose.WithSessionLocker(locker))
	if err != nil {
		return 0, fmt.Errorf("leer migraciones: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return len(results), fmt.Errorf("aplicar migraciones: %w", err)
	}
	return len(results), nil
}
