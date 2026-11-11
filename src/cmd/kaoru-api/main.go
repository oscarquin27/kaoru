// kaoru-api es el backend de Kaoru.
//
//	kaoru-api              arranca el servidor
//	kaoru-api healthcheck  sale con 0 si el servidor de este contenedor está listo
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kaoru/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	run := bootstrap.RunAPI
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		run = bootstrap.CheckHealth
	}
	err := run(ctx)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kaoru-api:", err)
		os.Exit(1)
	}
}
