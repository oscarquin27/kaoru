// Package archtest comprueba las reglas de capas de docs/reglas.md (§A.4, §A.6, §A.7).
package archtest

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	module   = "kaoru"
	internal = module + "/internal/"
)

type pkg struct {
	ImportPath string
	Dir        string
	Imports    []string
}

func listPackages(t *testing.T) []pkg {
	t.Helper()
	cmd := exec.Command("go", "list", "-json", module+"/...")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	var pkgs []pkg
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for {
		var p pkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

// location dice dónde vive un paquete del módulo.
type location struct {
	area    string // core, shared, bootstrap, feature, archtest, cmd, migrations
	feature string
	layer   string // domain, port, usecase, adapter, delivery, module, sqlcgen
}

func locate(importPath string) location {
	rest, ok := strings.CutPrefix(importPath, internal)
	if !ok {
		area, _, _ := strings.Cut(strings.TrimPrefix(importPath, module+"/"), "/")
		return location{area: area}
	}
	parts := strings.Split(rest, "/")
	loc := location{area: parts[0]}
	if loc.area != "feature" || len(parts) < 2 {
		return loc
	}
	loc.feature = parts[1]
	tail := strings.Join(parts[2:], "/")
	switch {
	case tail == "":
		loc.layer = "module"
	case tail == "domain":
		loc.layer = "domain"
	case tail == "application/port":
		loc.layer = "port"
	case tail == "application/usecase":
		loc.layer = "usecase"
	case strings.HasSuffix(tail, "/sqlcgen"):
		loc.layer = "sqlcgen"
	case strings.HasPrefix(tail, "adapter/"):
		loc.layer = "adapter"
	case strings.HasPrefix(tail, "delivery/"):
		loc.layer = "delivery"
	default:
		loc.layer = "desconocida:" + tail
	}
	return loc
}

// Librerías que el centro no puede conocer.
var technical = []string{"github.com/gin-gonic/gin", "net/http", "github.com/jackc/pgx", "database/sql"}

func isTechnical(imp string) bool {
	for _, prefix := range technical {
		if imp == prefix || strings.HasPrefix(imp, prefix+"/") {
			return true
		}
	}
	return false
}

// violation devuelve el motivo por el que from no puede importar imp, o vacío si puede.
func violation(from location, imp string) string {
	if imp == "log/slog" && from.area != "bootstrap" {
		return "solo bootstrap importa el logger concreto"
	}
	if from.area == "feature" && (from.layer == "domain" || from.layer == "port" || from.layer == "usecase") && isTechnical(imp) {
		return "dominio, puertos y casos de uso no conocen HTTP ni SQL"
	}
	if from.area == "shared" && isTechnical(imp) {
		return "shared es negocio puro"
	}
	if !strings.HasPrefix(imp, module+"/") {
		return ""
	}
	to := locate(imp)

	switch from.area {
	case "core":
		if to.area != "core" {
			return "core no importa nada de internal fuera de sí mismo"
		}
		return ""
	case "shared":
		if to.area != "shared" && imp != internal+"core/pkg/axis" {
			return "shared solo usa la biblioteca estándar y el vocabulario de errores"
		}
		return ""
	case "cmd", "bootstrap", "archtest", "migrations":
		if to.area == "bootstrap" && from.area != "cmd" && from.area != "bootstrap" {
			return "bootstrap solo lo importa cmd"
		}
		return ""
	}

	// from es un feature.
	if to.area == "bootstrap" {
		return "bootstrap solo lo importa cmd"
	}
	if to.area != "feature" {
		if from.layer == "domain" && to.area == "core" && imp != internal+"core/pkg/axis" {
			return "el dominio solo usa del core el vocabulario de errores"
		}
		return ""
	}
	if to.feature != from.feature {
		switch {
		case to.layer == "domain":
			return "" // identificadores y value objects de lectura
		case to.layer == "usecase" && (from.layer == "adapter" || from.layer == "module"):
			return "" // usar a otro feature: desde un adaptador o desde module.go
		default:
			return "de otro feature solo se importa su domain, o su usecase desde un adaptador o module.go"
		}
	}
	allowed := map[string][]string{
		"domain":   {},
		"port":     {"domain"},
		"usecase":  {"domain", "port"},
		"adapter":  {"domain", "port", "sqlcgen"},
		"delivery": {"domain", "usecase"},
		"sqlcgen":  {},
		"module":   {"domain", "port", "usecase", "adapter", "delivery"},
	}
	for _, layer := range allowed[from.layer] {
		if to.layer == layer {
			return ""
		}
	}
	return from.layer + " no puede importar " + to.layer
}

func TestLayerRules(t *testing.T) {
	for _, p := range listPackages(t) {
		from := locate(p.ImportPath)
		if strings.HasPrefix(from.layer, "desconocida:") {
			t.Errorf("%s: carpeta fuera de la anatomía de un feature (§A.5)", p.ImportPath)
		}
		for _, imp := range p.Imports {
			if why := violation(from, imp); why != "" {
				t.Errorf("%s importa %s: %s", p.ImportPath, imp, why)
			}
		}
	}
}

// Las reglas mismas, sobre casos conocidos: que el test no pase por no mirar nada.
func TestViolationCatchesKnownCases(t *testing.T) {
	f := internal + "feature/"
	cases := []struct {
		from, imp string
		forbidden bool
	}{
		{f + "change/domain", "github.com/gin-gonic/gin", true},
		{f + "change/domain", internal + "core/pkg/axis", false},
		{f + "change/domain", internal + "core/pkg/postgres", true},
		{f + "change/domain", f + "change/application/usecase", true},
		{f + "change/application/usecase", "github.com/jackc/pgx/v5", true},
		{f + "change/application/usecase", "net/http", true},
		{f + "change/application/usecase", f + "change/adapter/postgres", true},
		{f + "change/application/usecase", f + "change/application/port", false},
		{f + "change/application/usecase", f + "audit/application/usecase", true},
		{f + "change/adapter/audit", f + "audit/application/usecase", false},
		{f + "change/adapter/audit", f + "audit/adapter/postgres", true},
		{f + "change/adapter/audit", f + "audit/application/port", true},
		{f + "change/adapter/postgres", f + "change/application/usecase", true},
		{f + "change/adapter/postgres", f + "change/adapter/postgres/sqlcgen", false},
		{f + "change/adapter/postgres", f + "audit/adapter/postgres/sqlcgen", true},
		{f + "change/delivery/api", f + "change/adapter/postgres", true},
		{f + "change/delivery/api", f + "change/application/port", true},
		{f + "change/delivery/api", f + "change/application/usecase", false},
		{f + "change", f + "audit/application/usecase", false},
		{f + "change", internal + "bootstrap", true},
		{f + "change/application/usecase", "log/slog", true},
		{internal + "core/pkg/problem", internal + "shared/domain", true},
		{internal + "core/pkg/problem", internal + "core/pkg/axis", false},
		{internal + "shared/domain", f + "change/domain", true},
		{internal + "bootstrap", "log/slog", false},
		{module + "/cmd/kaoru-api", internal + "bootstrap", false},
	}
	for _, tc := range cases {
		why := violation(locate(tc.from), tc.imp)
		if tc.forbidden && why == "" {
			t.Errorf("%s → %s debía estar prohibido", tc.from, tc.imp)
		}
		if !tc.forbidden && why != "" {
			t.Errorf("%s → %s debía estar permitido: %s", tc.from, tc.imp, why)
		}
	}
}

// Dominio, casos de uso y shared reciben el tiempo por parámetro: no llaman a time.Now.
func TestNoTimeNowInTheCenter(t *testing.T) {
	for _, p := range listPackages(t) {
		loc := locate(p.ImportPath)
		center := loc.area == "shared" || (loc.area == "feature" && (loc.layer == "domain" || loc.layer == "usecase" || loc.layer == "port"))
		if !center {
			continue
		}
		forEachGoFile(t, p.Dir, func(path string, file *ast.File) {
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "time" && sel.Sel.Name == "Now" {
					t.Errorf("%s llama a time.Now: el tiempo llega por Clock", path)
				}
				return true
			})
		})
	}
}

func forEachGoFile(t *testing.T, dir string, fn func(path string, file *ast.File)) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		fn(path, file)
	}
}

// Cada feature escribe solo en sus tablas. Las consultas son la única vía de escritura.
func TestQueriesOnlyWriteTheirOwnTables(t *testing.T) {
	owned := map[string][]string{
		"workflow": {"workflow_state", "workflow_transition", "permission", "role_permission", "blocking_rule"},
		"audit":    {"audit_entry"},
		"user":     {"app_user"},
		// code_sequence es compartida: cada feature reserva ahí sus IDs legibles.
		"change":      {"change", "code_sequence"},
		"requirement": {"requirement", "requirement_link", "requirement_entry_check", "requirement_component", "code_sequence"},
		"risk":        {"risk_assessment", "risk_quality_characteristic", "code_sequence"},
		"testdesign":  {"functional_case", "test_case", "test_case_functional_case", "test_case_component", "code_sequence"},
		"execution":   {"test_run", "exploratory_session", "code_sequence"},
		"defect":      {"defect", "code_sequence"},
		"document":    {"document", "document_version", "document_link", "code_sequence"},
		"comment":     {"comment"},
		"savedview":   {"saved_view"},
		"component":   {"component", "code_sequence"},
		"build":       {"build", "build_requirement", "code_sequence"},
		"environment": {"environment", "deployment", "code_sequence"},
		"system":      {"system_under_test"},
		"bundle":      {"package", "package_build", "package_requirement", "code_sequence"},
		"regression":  {"regression_selection", "regression_item"},
		"production": {
			"checklist_template", "checklist_template_step", "checklist", "checklist_signature",
			"diagnostic", "diagnostic_review", "smoke_test", "code_sequence",
		},
		"report":   {"conformance_report", "code_sequence"},
		"planning": {"milestone", "activity", "activity_operator", "code_sequence"},
		// traceability y metrics son de solo lectura: no pueden escribir en ninguna tabla.
		"traceability": {},
		"metrics":      {},
	}
	root := filepath.Join("..", "feature")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".sql" {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		feature := strings.Split(filepath.ToSlash(rel), "/")[0]
		raw, err := os.ReadFile(path) //nolint:gosec // ruta del propio repositorio
		if err != nil {
			return err
		}
		tables, known := owned[feature]
		if !known {
			t.Errorf("%s: falta declarar en archtest las tablas del feature %s", path, feature)
		}
		for _, table := range writtenTables(string(raw)) {
			ok := false
			for _, own := range tables {
				ok = ok || own == table
			}
			if !ok {
				t.Errorf("%s escribe en %s, que no es una tabla de %s", path, table, feature)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// writtenTables devuelve las tablas que siguen a INSERT INTO, UPDATE o DELETE FROM.
func writtenTables(sql string) []string {
	var tables []string
	words := strings.Fields(strings.ToLower(sql))
	for i := 0; i+1 < len(words); i++ {
		var table string
		switch {
		case words[i] == "insert" && words[i+1] == "into" && i+2 < len(words):
			table = words[i+2]
		case words[i] == "delete" && words[i+1] == "from" && i+2 < len(words):
			table = words[i+2]
		case words[i] == "update" && words[i+1] != "set":
			// "DO UPDATE SET" de un ON CONFLICT no nombra tabla.
			table = words[i+1]
		}
		if table != "" {
			tables = append(tables, strings.Trim(table, "(;"))
		}
	}
	return tables
}
