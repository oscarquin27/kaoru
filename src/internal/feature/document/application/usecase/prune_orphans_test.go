package usecase_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"kaoru/internal/feature/document/application/port"
	"kaoru/internal/feature/document/application/usecase"
)

type pruneClock struct{}

func (pruneClock) Now() time.Time { return time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC) }

// inventory es un almacén en memoria: checksum → fecha del archivo.
type inventory struct {
	files      map[string]time.Time
	referenced []string
	listErr    error
	removeErr  error
}

func (i *inventory) List(context.Context) ([]port.StoredFile, error) {
	if i.listErr != nil {
		return nil, i.listErr
	}
	var out []port.StoredFile
	for checksum, at := range i.files {
		out = append(out, port.StoredFile{Checksum: checksum, ModifiedAt: at})
	}
	slices.SortFunc(out, func(a, b port.StoredFile) int { return a.ModifiedAt.Compare(b.ModifiedAt) })
	return out, nil
}

func (i *inventory) Remove(_ context.Context, checksum string) error {
	if i.removeErr != nil {
		return i.removeErr
	}
	delete(i.files, checksum)
	return nil
}

func (i *inventory) Referenced(_ context.Context, checksums []string) ([]string, error) {
	var out []string
	for _, c := range checksums {
		if slices.Contains(i.referenced, c) {
			out = append(out, c)
		}
	}
	return out, nil
}

func newInventory() *inventory {
	now := pruneClock{}.Now()
	return &inventory{
		files: map[string]time.Time{
			"registrado":      now.Add(-72 * time.Hour),
			"huerfano-viejo":  now.Add(-48 * time.Hour),
			"subida-en-curso": now.Add(-time.Minute),
		},
		referenced: []string{"registrado"},
	}
}

func TestPruneOrphansOnlyListsByDefault(t *testing.T) {
	inv := newInventory()
	uc, err := usecase.NewPruneOrphans(inv, inv, pruneClock{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := uc.Execute(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Scanned != 3 || !slices.Equal(out.Orphans, []string{"huerfano-viejo"}) || out.Removed || len(inv.files) != 3 {
		t.Fatalf("sin apply solo lista el huérfano viejo y no borra: %+v, quedan %d", out, len(inv.files))
	}
}

func TestPruneOrphansRemovesOnlyOldUnreferencedFiles(t *testing.T) {
	inv := newInventory()
	uc, _ := usecase.NewPruneOrphans(inv, inv, pruneClock{})
	out, err := uc.Execute(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	_, registered := inv.files["registrado"]
	_, uploading := inv.files["subida-en-curso"]
	if !out.Removed || len(out.Orphans) != 1 || len(inv.files) != 2 || !registered || !uploading {
		t.Fatalf("debía borrar solo el huérfano viejo: %+v, quedan %v", out, inv.files)
	}
}

func TestPruneOrphansErrors(t *testing.T) {
	if _, err := usecase.NewPruneOrphans(nil, nil, nil); err == nil {
		t.Error("sin dependencias debía fallar")
	}
	boom := errors.New("disco")
	inv := newInventory()
	inv.listErr = boom
	uc, _ := usecase.NewPruneOrphans(inv, inv, pruneClock{})
	if _, err := uc.Execute(context.Background(), true); !errors.Is(err, boom) {
		t.Errorf("quiero el error de listar, obtuve %v", err)
	}
	inv = newInventory()
	inv.removeErr = boom
	uc, _ = usecase.NewPruneOrphans(inv, inv, pruneClock{})
	if _, err := uc.Execute(context.Background(), true); !errors.Is(err, boom) {
		t.Errorf("quiero el error de borrar, obtuve %v", err)
	}
}
