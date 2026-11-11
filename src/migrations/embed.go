// Package migrations embebe el SQL versionado en el binario.
package migrations

import "embed"

// FS contiene las migraciones de goose.
//
//go:embed *.sql
var FS embed.FS
