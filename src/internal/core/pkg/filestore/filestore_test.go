package filestore_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/filestore"
)

func TestPutAndOpen(t *testing.T) {
	dir := t.TempDir()
	store, err := filestore.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const content = "Especificación del límite diario"
	sum := sha256.Sum256([]byte(content))
	want := hex.EncodeToString(sum[:])

	stored, err := store.Put(ctx, strings.NewReader(content))
	if err != nil || stored.Checksum != want || stored.Size != int64(len(content)) {
		t.Fatalf("Put = %+v, %v; quiero checksum %s", stored, err, want)
	}
	// Guardar lo mismo otra vez deja un solo archivo y ningún temporal.
	if again, err := store.Put(ctx, strings.NewReader(content)); err != nil || again != stored {
		t.Fatalf("segundo Put = %+v, %v", again, err)
	}
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, _ error) error {
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if len(files) != 1 || !strings.HasSuffix(files[0], filepath.Join(want[:2], want)) {
		t.Errorf("archivos en disco = %v", files)
	}

	file, err := store.Open(ctx, want)
	if err != nil {
		t.Fatal(err)
	}
	read, _ := io.ReadAll(file)
	_ = file.Close()
	if string(read) != content {
		t.Errorf("contenido leído = %q", read)
	}
}

func TestOpenRejectsUnknownAndUnsafeNames(t *testing.T) {
	store, err := filestore.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{strings.Repeat("0", 64), "", "../../etc/passwd", strings.Repeat("Z", 64), "../" + strings.Repeat("a", 61)} {
		if _, err := store.Open(context.Background(), name); !axis.IsCode(err, axis.NotFoundResource) {
			t.Errorf("Open(%q) debía ser NOT_FOUND_RESOURCE, obtuve %v", name, err)
		}
	}
}

func TestPutStopsWhenCanceled(t *testing.T) {
	dir := t.TempDir()
	store, _ := filestore.NewLocal(dir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Put(ctx, strings.NewReader("x")); err == nil {
		t.Fatal("Put con el contexto cancelado debía fallar")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "tmp"))
	if len(entries) != 0 {
		t.Errorf("no debe quedar el temporal: %v", entries)
	}
}

func TestDetect(t *testing.T) {
	store, _ := filestore.NewLocal(t.TempDir())
	cases := map[string]string{
		"%PDF-1.7\n":                      "application/pdf",
		"\x89PNG\r\n\x1a\n":               "image/png",
		"# Título\n\nTexto":               "text/plain",
		"MZ\x90\x00\x03":                  "application/octet-stream",
		"<html><script>alert(1)</script>": "text/html",
		"PK\x03\x04\x14\x00\x06\x00":      "application/zip",
	}
	for head, want := range cases {
		if got := store.Detect([]byte(head)); got != want {
			t.Errorf("Detect(%q) = %s, quiero %s", head, got, want)
		}
	}
}
