package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// CheckHealth pregunta a /readyz del proceso que corre en este contenedor.
// Es el healthcheck de Docker: la imagen no trae curl ni shell.
func CheckHealth(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(cfg.HTTP.InternalAddr)
	if err != nil {
		return fmt.Errorf("http.internalAddr: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort("127.0.0.1", port)+"/readyz", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/readyz respondió %d", resp.StatusCode)
	}
	return nil
}
