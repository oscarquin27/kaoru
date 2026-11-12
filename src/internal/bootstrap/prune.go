package bootstrap

import (
	"context"
	"fmt"
	"os"
	"slices"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/filestore"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/document"
)

// PruneDocuments lista los archivos del almacén que no tienen registro y, con --apply, los borra.
// Sin --apply no cambia nada.
func PruneDocuments(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	pool, err := corepg.NewPool(ctx, cfg.Postgres.AppURL)
	if err != nil {
		return fmt.Errorf("crear pool de Postgres: %w", err)
	}
	defer pool.Close()
	files, err := filestore.NewLocal(cfg.Documents.Dir)
	if err != nil {
		return err
	}
	pruner, err := document.NewPruner(corepg.NewDatabase(pool), files, clock.System{})
	if err != nil {
		return err
	}
	out, err := pruner.Execute(ctx, slices.Contains(os.Args[2:], "--apply"))
	if err != nil {
		return err
	}
	for _, checksum := range out.Orphans {
		fmt.Println(checksum)
	}
	verb := "sin registro (nada se borró; repite con --apply para borrarlos)"
	if out.Removed {
		verb = "sin registro, borrados"
	}
	fmt.Printf("%d archivos revisados, %d %s\n", out.Scanned, len(out.Orphans), verb)
	return nil
}
