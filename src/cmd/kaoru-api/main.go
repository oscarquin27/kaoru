// kaoru-api es el backend de Kaoru.
//
//	kaoru-api              arranca el servidor
//	kaoru-api healthcheck  sale con 0 si el servidor de este contenedor está listo
//	kaoru-api prune-documents [--apply]  lista (o borra) los archivos sin registro
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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			run = bootstrap.CheckHealth
		case "prune-documents":
			run = bootstrap.PruneDocuments
		default:
			fmt.Fprintln(os.Stderr, "kaoru-api: orden desconocida:", os.Args[1])
			os.Exit(2)
		}
	}
	err := run(ctx)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kaoru-api:", err)
		os.Exit(1)
	}
}
