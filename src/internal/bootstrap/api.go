package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/filestore"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/oidc"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/migrations"
)

const (
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 20 * time.Second
)

// RunAPI arranca kaoru-api y lo mantiene hasta que ctx se cancela o un servidor falla.
func RunAPI(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	log := newLogger(os.Stdout, cfg.Log.Format)

	applied, err := corepg.Migrate(ctx, cfg.Postgres.OwnerURL, migrations.FS)
	if err != nil {
		return err
	}
	log.InfoContext(ctx, "migraciones al día", "applied", applied)

	pool, err := pgxpool.New(ctx, cfg.Postgres.AppURL)
	if err != nil {
		return fmt.Errorf("crear pool de Postgres: %w", err)
	}
	defer pool.Close()
	database := corepg.NewDatabase(pool)
	if err := database.Ping(ctx); err != nil {
		return fmt.Errorf("la base no responde: %w", err)
	}

	files, err := filestore.NewLocal(cfg.Documents.Dir)
	if err != nil {
		return err
	}

	router, err := NewRouter(RouterDeps{
		Config:    cfg,
		Database:  database,
		FileStore: files,
		Verifier: oidc.NewVerifier(ctx, oidc.Config{
			Issuer: cfg.OIDC.Issuer, JWKSURL: cfg.OIDC.JWKSURL, ClientID: cfg.OIDC.ClientID,
		}),
		Logger:      log,
		Clock:       clock.System{},
		IdGenerator: idgen.UUIDv7{},
	})
	if err != nil {
		return err
	}

	health := httpserver.NewHealth(database.Ping)
	public := &http.Server{Addr: cfg.HTTP.Addr, Handler: router, ReadHeaderTimeout: readHeaderTimeout}
	internal := &http.Server{Addr: cfg.HTTP.InternalAddr, Handler: health.Handler(), ReadHeaderTimeout: readHeaderTimeout}

	group, groupCtx := errgroup.WithContext(ctx)
	for _, server := range []*http.Server{public, internal} {
		group.Go(func() error {
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("servidor %s: %w", server.Addr, err)
			}
			return nil
		})
	}
	log.InfoContext(ctx, "kaoru-api escuchando", "addr", cfg.HTTP.Addr, "internal_addr", cfg.HTTP.InternalAddr)

	group.Go(func() error {
		<-groupCtx.Done()
		// Apagado en orden: /readyz pasa a falso, se atiende lo que está en curso y se cierra.
		health.Drain()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		return errors.Join(public.Shutdown(shutdownCtx), internal.Shutdown(shutdownCtx))
	})
	err = group.Wait()
	log.InfoContext(context.WithoutCancel(ctx), "kaoru-api detenido")
	return err
}
