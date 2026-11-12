package usecase_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/document/application/port"
	"kaoru/internal/feature/document/application/usecase"
	"kaoru/internal/feature/document/domain"
	shared "kaoru/internal/shared/domain"
)

// link es un enlace tal como lo recibió el repositorio.
type link struct {
	documentId string
	entity     domain.EntityRef
	by         shared.UserId
	at         time.Time
}

// world es el estado en memoria de los dobles; lo escrito dentro de Within se descarta si falla.
// Los archivos no: se guardan fuera de la transacción, antes que su registro.
type world struct {
	documents     map[string]domain.Document // por código
	links         []link
	files         map[string][]byte // por checksum
	audit         []port.AuditRecord
	sequence      int
	auditErr      error
	addVersionErr error
}

func (w *world) Within(ctx context.Context, fn func(context.Context) error) error {
	documents, links, audit, sequence := maps.Clone(w.documents), slices.Clone(w.links), slices.Clone(w.audit), w.sequence
	if err := fn(ctx); err != nil {
		w.documents, w.links, w.audit, w.sequence = documents, links, audit, sequence
		return err
	}
	return nil
}

var (
	notFound = axis.New(axis.NotFoundResource, "No existe.")
	down     = axis.New(axis.ServiceUnavailable, "Sin servicio.")
)

func (w *world) NextCode(_ context.Context, year int) (shared.Code, error) {
	w.sequence++
	return shared.NewCode(shared.PrefixDocument, year, w.sequence), nil
}

func (w *world) Insert(_ context.Context, d domain.Document, linkedBy shared.UserId) error {
	s := d.Snapshot()
	w.documents[s.Code.String()] = d
	w.links = append(w.links, link{documentId: s.Id, entity: s.LinkedTo[0], by: linkedBy, at: s.CreatedAt})
	return nil
}

// AddVersion numera la versión y la pone primero: van de la más reciente a la más antigua.
func (w *world) AddVersion(_ context.Context, code shared.Code, version domain.Version) (int, error) {
	if w.addVersionErr != nil {
		return 0, w.addVersionErr
	}
	d, ok := w.documents[code.String()]
	if !ok {
		return 0, notFound
	}
	s := d.Snapshot()
	version.Number = len(s.Versions) + 1
	s.Versions = append([]domain.Version{version}, s.Versions...)
	w.documents[code.String()] = domain.Rehydrate(s)
	return version.Number, nil
}

func (w *world) AddLink(_ context.Context, documentId string, entity domain.EntityRef, by shared.UserId, at time.Time) error {
	for code, d := range w.documents {
		s := d.Snapshot()
		if s.Id != documentId || d.IsLinkedTo(entity) {
			continue
		}
		s.LinkedTo = append(slices.Clone(s.LinkedTo), entity)
		w.documents[code] = domain.Rehydrate(s)
		w.links = append(w.links, link{documentId: documentId, entity: entity, by: by, at: at})
	}
	return nil
}

func (w *world) FindByCode(_ context.Context, code shared.Code) (domain.Document, error) {
	d, ok := w.documents[code.String()]
	if !ok {
		return domain.Document{}, notFound
	}
	return d, nil
}

func (w *world) ListByEntity(_ context.Context, entity domain.EntityRef) ([]domain.Document, error) {
	var out []domain.Document
	for _, d := range w.documents {
		if d.IsLinkedTo(entity) {
			out = append(out, d)
		}
	}
	slices.SortFunc(out, func(a, b domain.Document) int {
		return strings.Compare(a.Snapshot().Code.String(), b.Snapshot().Code.String())
	})
	return out, nil
}

// Put hace de port.FileStore: nombra el contenido por su SHA-256.
func (w *world) Put(_ context.Context, r io.Reader) (port.Stored, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return port.Stored{}, err
	}
	w.files[checksumOf(content)] = content
	return port.Stored{Checksum: checksumOf(content), Size: int64(len(content))}, nil
}

func (w *world) Open(_ context.Context, checksum string) (io.ReadCloser, error) {
	content, ok := w.files[checksum]
	if !ok {
		return nil, notFound
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

// Detect reconoce un PDF por su firma; todo lo demás es texto.
func (w *world) Detect(head []byte) string {
	if bytes.HasPrefix(head, []byte("%PDF-")) {
		return "application/pdf"
	}
	return "text/plain"
}

// Resolve hace de port.Entities; el tipo «down» simula que el otro feature no responde.
func (w *world) Resolve(_ context.Context, entityType, code string) (domain.EntityRef, error) {
	if entityType == "down" {
		return domain.EntityRef{}, down
	}
	entity, ok := entities[entityType+"/"+code]
	if !ok {
		return domain.EntityRef{}, notFound
	}
	return entity, nil
}

func (w *world) Record(_ context.Context, r port.AuditRecord) error {
	if w.auditErr != nil {
		return w.auditErr
	}
	w.audit = append(w.audit, r)
	return nil
}

type fakePermissions struct{}

func (fakePermissions) Require(_ context.Context, roles []string, permission string) error {
	if permission == usecase.PermissionUpload && slices.Contains(roles, "qa") {
		return nil
	}
	return axis.New(axis.PermissionDenied, "Sin permiso.")
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return t0 }

type seqIDs struct{ n int }

func (s *seqIDs) NewId() (string, error) {
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

var (
	t0          = time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	qa          = shared.Actor{Id: "u2", Roles: []string{"qa"}}
	dev         = shared.Actor{Id: "u3", Roles: []string{"dev"}}
	ctx         = context.Background()
	requirement = domain.EntityRef{Type: "requirement", Id: "r1", Code: "REQ-2026-0042"}
	defect      = domain.EntityRef{Type: "defect", Id: "d1", Code: "DEF-2026-0001"}
	entities    = map[string]domain.EntityRef{"requirement/REQ-2026-0042": requirement, "defect/DEF-2026-0001": defect}
	// pdf pasa de los 512 bytes que se leen para detectar el tipo.
	pdf  = "%PDF-1.7\n" + strings.Repeat("contenido ", 200)
	pdf2 = "%PDF-1.7\nsegunda versión"
)

func checksumOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func file(name, content string) usecase.FileInput {
	return usecase.FileInput{Name: name, Content: strings.NewReader(content)}
}

func deps(w *world) usecase.Deps {
	return usecase.Deps{
		Repository: w, FileStore: w, Entities: w, Permissions: fakePermissions{}, AuditTrail: w,
		UnitOfWork: w, Clock: fixedClock{}, IdGenerator: &seqIDs{}, AllowedExtensions: []string{".pdf", ".txt"},
	}
}

func setup(t *testing.T) (*usecase.UseCases, *world) {
	t.Helper()
	w := &world{documents: map[string]domain.Document{}, files: map[string][]byte{}}
	uc, err := usecase.New(deps(w))
	if err != nil {
		t.Fatal(err)
	}
	return uc, w
}

// attach adjunta un PDF al REQ y devuelve el código del documento.
func attach(t *testing.T, uc *usecase.UseCases) string {
	t.Helper()
	out, err := uc.CreateDocument.Execute(ctx, usecase.CreateDocumentInput{
		Actor: qa, Title: "Especificación", Kind: "specification", EntityType: "requirement", EntityCode: "REQ-2026-0042",
		File: file("especificacion.pdf", pdf),
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.Code.String()
}

func fieldsOf(err error) []string {
	var ge *axis.GeneralError
	if !errors.As(err, &ge) {
		return nil
	}
	var out []string
	for _, f := range ge.Fields() {
		out = append(out, f.Field+":"+f.Code)
	}
	return out
}

func TestNewRequiresDeps(t *testing.T) {
	if _, err := usecase.New(usecase.Deps{}); err == nil {
		t.Fatal("New debía fallar sin dependencias")
	}
	d := deps(&world{})
	d.AllowedExtensions = nil
	if _, err := usecase.New(d); err == nil {
		t.Fatal("New debía fallar sin la lista de tipos permitidos")
	}
}

func TestCreateDocument(t *testing.T) {
	uc, w := setup(t)
	// Del nombre solo queda el archivo, sin directorios.
	out, err := uc.CreateDocument.Execute(ctx, usecase.CreateDocumentInput{
		Actor: qa, Title: "  Especificación ", Kind: "specification", EntityType: "requirement", EntityCode: "REQ-2026-0042",
		File: file(`C:\docs\Especificación.PDF`, pdf),
	})
	if err != nil {
		t.Fatal(err)
	}
	checksum := checksumOf([]byte(pdf))
	saved := w.documents["DOC-2026-0001"].Snapshot()
	if out.Code.String() != "DOC-2026-0001" || saved.Id != "id-2" || saved.Title != "Especificación" || saved.Kind != "specification" ||
		saved.CreatedBy != "u2" || !saved.CreatedAt.Equal(t0) || !slices.Equal(saved.LinkedTo, []domain.EntityRef{requirement}) {
		t.Errorf("documento guardado inesperado: %+v", saved)
	}
	wantVersion := domain.Version{
		Id: "id-1", Number: 1, FileName: "Especificación.PDF", MediaType: "application/pdf", SizeBytes: int64(len(pdf)),
		Checksum: checksum, UploadedBy: domain.UserRef{Id: "u2"}, CreatedAt: t0,
	}
	if !slices.Equal(saved.Versions, []domain.Version{wantVersion}) || !slices.Equal(out.Versions, saved.Versions) {
		t.Errorf("primera versión: %+v", saved.Versions)
	}
	// El contenido se guarda completo, no solo lo que se leyó para detectar el tipo.
	if len(w.files) != 1 || string(w.files[checksum]) != pdf {
		t.Errorf("el archivo guardado no es el que se subió: %d archivos", len(w.files))
	}
	if !slices.Equal(w.links, []link{{documentId: "id-2", entity: requirement, by: "u2", at: t0}}) {
		t.Errorf("primer enlace: %+v", w.links)
	}
	want := port.AuditRecord{Actor: "u2", DocumentId: "id-2", Code: out.Code, Action: port.AuditCreate, Field: "checksum", NewValue: checksum}
	if len(w.audit) != 1 || w.audit[0] != want {
		t.Errorf("auditoría del alta: %+v", w.audit)
	}
}

func TestCreateDocumentErrors(t *testing.T) {
	type input = usecase.CreateDocumentInput
	cases := []struct {
		name       string
		mutate     func(in *input)
		wantCode   axis.Code
		wantFields []string
		wantErr    error
		// wantFiles: el tipo se valida antes de escribir el archivo; el título y la clase, después.
		wantFiles int
	}{
		{"sin permiso", func(in *input) { in.Actor = dev }, axis.PermissionDenied, nil, nil, 0},
		{"sin tipo de entidad", func(in *input) { in.EntityType = "" }, axis.InvalidRequest, []string{"entityCode:required"}, nil, 0},
		{"sin código de entidad", func(in *input) { in.EntityCode = "" }, axis.InvalidRequest, []string{"entityCode:required"}, nil, 0},
		{"entidad inexistente", func(in *input) { in.EntityCode = "REQ-2026-0099" }, axis.InvalidRequest, []string{"entityCode:invalid"}, nil, 0},
		{"entidad que no admite documentos", func(in *input) { in.EntityType = "user" }, axis.InvalidRequest, []string{"entityCode:invalid"}, nil, 0},
		{"no se pudo resolver la entidad", func(in *input) { in.EntityType = "down" }, axis.ServiceUnavailable, nil, down, 0},
		{"sin archivo", func(in *input) { in.File = usecase.FileInput{Name: "a.pdf"} }, axis.InvalidRequest, []string{"file:required"}, nil, 0},
		{"archivo vacío", func(in *input) { in.File = file("a.pdf", "") }, axis.InvalidRequest, []string{"file:required"}, domain.ErrEmptyFile, 0},
		{"archivo sin nombre", func(in *input) { in.File = file("  ", pdf) }, axis.InvalidRequest, []string{"file:required"}, nil, 0},
		{"nombre demasiado largo", func(in *input) { in.File = file(strings.Repeat("a", 200)+".pdf", pdf) }, axis.InvalidRequest, []string{"file:invalid"}, nil, 0},
		{"tipo desconocido", func(in *input) { in.File = file("a.exe", "MZ") }, axis.NotSupported, nil, domain.ErrTypeNotSupported, 0},
		{"tipo conocido pero no permitido", func(in *input) { in.File = file("a.md", "# Notas") }, axis.NotSupported, nil, domain.ErrTypeNotSupported, 0},
		{"el contenido no es lo que dice la extensión", func(in *input) { in.File = file("a.pdf", "solo texto") }, axis.NotSupported, nil, domain.ErrTypeMismatch, 0},
		{"sin título ni clase", func(in *input) { in.Title, in.Kind = " ", "" }, axis.InvalidRequest, []string{"title:required", "kind:required"}, nil, 1},
		{"título largo y clase inexistente", func(in *input) { in.Title, in.Kind = strings.Repeat("a", 201), "contrato" }, axis.InvalidRequest, []string{"title:invalid", "kind:invalid"}, nil, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, w := setup(t)
			in := input{
				Actor: qa, Title: "Especificación", Kind: "specification", EntityType: "requirement", EntityCode: "REQ-2026-0042",
				File: file("especificacion.pdf", pdf),
			}
			tc.mutate(&in)
			_, err := uc.CreateDocument.Execute(ctx, in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) || (tc.wantErr != nil && !errors.Is(err, tc.wantErr)) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.documents) != 0 || len(w.links) != 0 || len(w.audit) != 0 || w.sequence != 0 {
				t.Error("un alta rechazada no debe dejar registro, ni gastar un código")
			}
			if len(w.files) != tc.wantFiles {
				t.Errorf("archivos guardados = %d, quiero %d", len(w.files), tc.wantFiles)
			}
		})
	}
}

func TestAddVersion(t *testing.T) {
	uc, w := setup(t)
	code := attach(t, uc)

	out, err := uc.AddVersion.Execute(ctx, usecase.AddVersionInput{Actor: qa, Code: code, File: file("v2.pdf", pdf2)})
	if err != nil {
		t.Fatal(err)
	}
	first, second := checksumOf([]byte(pdf)), checksumOf([]byte(pdf2))
	saved := w.documents[code].Snapshot()
	// La versión anterior se conserva; la nueva va primero.
	if len(saved.Versions) != 2 || saved.Versions[0].Number != 2 || saved.Versions[0].Checksum != second ||
		saved.Versions[0].FileName != "v2.pdf" || saved.Versions[0].UploadedBy.Id != "u2" || !saved.Versions[0].CreatedAt.Equal(t0) ||
		saved.Versions[1].Number != 1 || saved.Versions[1].Checksum != first || !slices.Equal(out.Versions, saved.Versions) {
		t.Errorf("versiones guardadas: %+v", saved.Versions)
	}
	if len(w.files) != 2 || string(w.files[second]) != pdf2 || string(w.files[first]) != pdf {
		t.Errorf("deben quedar los dos archivos: %d", len(w.files))
	}
	want := port.AuditRecord{Actor: "u2", DocumentId: saved.Id, Code: saved.Code, Action: port.AuditUpdate, Field: "version.2", NewValue: second}
	if len(w.audit) != 2 || w.audit[1] != want {
		t.Errorf("auditoría de la versión: %+v", w.audit)
	}
}

func TestAddVersionErrors(t *testing.T) {
	uc, w := setup(t)
	code := attach(t, uc)
	cases := []struct {
		name       string
		in         usecase.AddVersionInput
		wantCode   axis.Code
		wantFields []string
		wantErr    error
	}{
		{"sin permiso", usecase.AddVersionInput{Actor: dev, Code: code, File: file("v2.pdf", pdf2)}, axis.PermissionDenied, nil, nil},
		{"documento inexistente", usecase.AddVersionInput{Actor: qa, Code: "DOC-2026-0099", File: file("v2.pdf", pdf2)}, axis.NotFoundResource, nil, usecase.ErrDocumentNotFound},
		{"código mal formado", usecase.AddVersionInput{Actor: qa, Code: "doc 1", File: file("v2.pdf", pdf2)}, axis.NotFoundResource, nil, usecase.ErrDocumentNotFound},
		{"sin archivo", usecase.AddVersionInput{Actor: qa, Code: code}, axis.InvalidRequest, []string{"file:required"}, nil},
		{"archivo vacío", usecase.AddVersionInput{Actor: qa, Code: code, File: file("v2.pdf", "")}, axis.InvalidRequest, []string{"file:required"}, domain.ErrEmptyFile},
		{"tipo no permitido", usecase.AddVersionInput{Actor: qa, Code: code, File: file("v2.md", "# Notas")}, axis.NotSupported, nil, domain.ErrTypeNotSupported},
		{"el contenido no es lo que dice la extensión", usecase.AddVersionInput{Actor: qa, Code: code, File: file("v2.pdf", "solo texto")}, axis.NotSupported, nil, domain.ErrTypeMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.AddVersion.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) || (tc.wantErr != nil && !errors.Is(err, tc.wantErr)) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.documents[code].Snapshot().Versions) != 1 || len(w.audit) != 1 || len(w.files) != 1 {
				t.Error("una versión rechazada no debe dejar nada")
			}
		})
	}

	// Si el documento desaparece entre que se busca y se guarda la versión, tampoco existe.
	w.addVersionErr = notFound
	if _, err := uc.AddVersion.Execute(ctx, usecase.AddVersionInput{Actor: qa, Code: code, File: file("v2.pdf", pdf2)}); !errors.Is(err, usecase.ErrDocumentNotFound) {
		t.Errorf("documento que desaparece: %v", err)
	}
	if len(w.documents[code].Snapshot().Versions) != 1 || len(w.audit) != 1 {
		t.Error("una versión rechazada no debe dejar registro")
	}
}

func TestLinkDocument(t *testing.T) {
	uc, w := setup(t)
	code := attach(t, uc)

	out, err := uc.LinkDocument.Execute(ctx, usecase.LinkDocumentInput{Actor: qa, Code: code, EntityType: "defect", EntityCode: "DEF-2026-0001"})
	if err != nil {
		t.Fatal(err)
	}
	saved := w.documents[code].Snapshot()
	if !slices.Equal(saved.LinkedTo, []domain.EntityRef{requirement, defect}) || !slices.Equal(out.LinkedTo, saved.LinkedTo) {
		t.Errorf("enlaces guardados: %+v", saved.LinkedTo)
	}
	if len(w.links) != 2 || w.links[1] != (link{documentId: saved.Id, entity: defect, by: "u2", at: t0}) {
		t.Errorf("quién enlazó y cuándo: %+v", w.links)
	}
	want := port.AuditRecord{Actor: "u2", DocumentId: saved.Id, Code: saved.Code, Action: port.AuditUpdate, Field: "link", NewValue: "DEF-2026-0001"}
	if len(w.audit) != 2 || w.audit[1] != want {
		t.Errorf("auditoría del enlace: %+v", w.audit)
	}

	// Enlazar dos veces a lo mismo no cambia nada ni se audita.
	for _, entity := range []domain.EntityRef{requirement, defect} {
		out, err := uc.LinkDocument.Execute(ctx, usecase.LinkDocumentInput{Actor: qa, Code: code, EntityType: entity.Type, EntityCode: entity.Code})
		if err != nil || len(out.LinkedTo) != 2 || len(w.links) != 2 || len(w.audit) != 2 {
			t.Errorf("enlazar otra vez a %s: %+v, %v", entity.Code, out.LinkedTo, err)
		}
	}
}

func TestLinkDocumentErrors(t *testing.T) {
	uc, w := setup(t)
	code := attach(t, uc)
	cases := []struct {
		name       string
		in         usecase.LinkDocumentInput
		wantCode   axis.Code
		wantFields []string
	}{
		{"sin permiso", usecase.LinkDocumentInput{Actor: dev, Code: code, EntityType: "defect", EntityCode: "DEF-2026-0001"}, axis.PermissionDenied, nil},
		{"documento inexistente", usecase.LinkDocumentInput{Actor: qa, Code: "DOC-2026-0099", EntityType: "defect", EntityCode: "DEF-2026-0001"}, axis.NotFoundResource, nil},
		{"código mal formado", usecase.LinkDocumentInput{Actor: qa, Code: "doc 1", EntityType: "defect", EntityCode: "DEF-2026-0001"}, axis.NotFoundResource, nil},
		{"sin entidad", usecase.LinkDocumentInput{Actor: qa, Code: code}, axis.InvalidRequest, []string{"entityCode:required"}},
		{"entidad inexistente", usecase.LinkDocumentInput{Actor: qa, Code: code, EntityType: "defect", EntityCode: "DEF-2026-0099"}, axis.InvalidRequest, []string{"entityCode:invalid"}},
		{"no se pudo resolver la entidad", usecase.LinkDocumentInput{Actor: qa, Code: code, EntityType: "down", EntityCode: "DEF-2026-0001"}, axis.ServiceUnavailable, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.LinkDocument.Execute(ctx, tc.in)
			if !axis.IsCode(err, tc.wantCode) || !slices.Equal(fieldsOf(err), tc.wantFields) {
				t.Fatalf("esperaba %s %v, obtuve %v %v", tc.wantCode, tc.wantFields, err, fieldsOf(err))
			}
			if len(w.documents[code].Snapshot().LinkedTo) != 1 || len(w.links) != 1 || len(w.audit) != 1 {
				t.Error("un enlace rechazado no debe dejar nada")
			}
		})
	}
}

func TestNothingIsSavedIfAuditFails(t *testing.T) {
	uc, w := setup(t)
	code := attach(t, uc)
	w.auditErr = axis.New(axis.ServiceUnavailable, "sin auditoría")

	if _, err := uc.CreateDocument.Execute(ctx, usecase.CreateDocumentInput{
		Actor: qa, Title: "Otro", Kind: "manual", EntityType: "requirement", EntityCode: "REQ-2026-0042", File: file("otro.pdf", pdf2),
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("adjuntar: %v", err)
	}
	if _, err := uc.AddVersion.Execute(ctx, usecase.AddVersionInput{Actor: qa, Code: code, File: file("v2.pdf", pdf2)}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("subir versión: %v", err)
	}
	if _, err := uc.LinkDocument.Execute(ctx, usecase.LinkDocumentInput{
		Actor: qa, Code: code, EntityType: "defect", EntityCode: "DEF-2026-0001",
	}); !axis.IsCode(err, axis.ServiceUnavailable) {
		t.Errorf("enlazar: %v", err)
	}

	saved := w.documents[code].Snapshot()
	if len(w.documents) != 1 || len(saved.Versions) != 1 || len(saved.LinkedTo) != 1 || len(w.links) != 1 || w.sequence != 1 || len(w.audit) != 1 {
		t.Error("si la auditoría falla, no debe quedar ningún registro")
	}
}

func TestGetAndListDocuments(t *testing.T) {
	uc, _ := setup(t)
	first, second := attach(t, uc), attach(t, uc)
	if _, err := uc.LinkDocument.Execute(ctx, usecase.LinkDocumentInput{Actor: qa, Code: second, EntityType: "defect", EntityCode: "DEF-2026-0001"}); err != nil {
		t.Fatal(err)
	}

	// Leer no exige permiso: ni siquiera recibe un actor.
	out, err := uc.GetDocument.Execute(ctx, usecase.GetDocumentInput{Code: first})
	if err != nil || out.Code.String() != first || out.Title != "Especificación" || len(out.Versions) != 1 {
		t.Errorf("GetDocument = %+v, %v", out, err)
	}
	for _, code := range []string{"DOC-2026-0099", "no es un código"} {
		if _, err := uc.GetDocument.Execute(ctx, usecase.GetDocumentInput{Code: code}); !errors.Is(err, usecase.ErrDocumentNotFound) {
			t.Errorf("documento %q: %v", code, err)
		}
	}

	cases := []struct {
		name       string
		entityType string
		entityCode string
		want       []string
	}{
		{"los del REQ", "requirement", "REQ-2026-0042", []string{first, second}},
		{"los del DEF", "defect", "DEF-2026-0001", []string{second}},
		{"entidad inexistente", "defect", "DEF-2026-0099", []string{}},
	}
	for _, tc := range cases {
		list, err := uc.ListDocuments.Execute(ctx, usecase.ListDocumentsInput{EntityType: tc.entityType, EntityCode: tc.entityCode})
		got := []string{}
		for _, s := range list {
			got = append(got, s.Code.String())
		}
		if err != nil || list == nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v, %v; quiero %v", tc.name, got, err, tc.want)
		}
	}
	if _, err := uc.ListDocuments.Execute(ctx, usecase.ListDocumentsInput{EntityType: "down", EntityCode: "x"}); !errors.Is(err, down) {
		t.Errorf("no se pudo resolver la entidad: %v", err)
	}
}

func TestOpenContent(t *testing.T) {
	uc, w := setup(t)
	code := attach(t, uc)
	if _, err := uc.AddVersion.Execute(ctx, usecase.AddVersionInput{Actor: qa, Code: code, File: file("v2.pdf", pdf2)}); err != nil {
		t.Fatal(err)
	}

	for number, want := range map[int]string{1: pdf, 2: pdf2} {
		out, err := uc.OpenContent.Execute(ctx, usecase.OpenContentInput{Code: code, Version: number})
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(out.Content)
		if closeErr := out.Content.Close(); err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
		if string(content) != want || out.Version.Number != number || out.Version.Checksum != checksumOf([]byte(want)) ||
			out.Version.MediaType != "application/pdf" {
			t.Errorf("versión %d: %+v", number, out.Version)
		}
	}

	cases := []struct {
		name string
		in   usecase.OpenContentInput
	}{
		{"versión inexistente", usecase.OpenContentInput{Code: code, Version: 3}},
		{"versión cero", usecase.OpenContentInput{Code: code}},
		{"documento inexistente", usecase.OpenContentInput{Code: "DOC-2026-0099", Version: 1}},
		{"código mal formado", usecase.OpenContentInput{Code: "doc 1", Version: 1}},
	}
	for _, tc := range cases {
		if _, err := uc.OpenContent.Execute(ctx, tc.in); !errors.Is(err, usecase.ErrDocumentNotFound) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}

	// Un registro cuyo archivo ya no está en el almacén.
	delete(w.files, checksumOf([]byte(pdf)))
	if _, err := uc.OpenContent.Execute(ctx, usecase.OpenContentInput{Code: code, Version: 1}); !axis.IsCode(err, axis.NotFoundResource) {
		t.Errorf("archivo perdido: %v", err)
	}
}
