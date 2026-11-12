package mdpdf

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

var options = Options{
	Title: "Informe de conformidad RPT-2026-0001", Author: "Quinn QA", CreatedAt: time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC),
	Brand: "kaoru", Motto: "Trazabilidad a prueba de errores",
	Footer: []string{"kaoru · Informe de conformidad RPT-2026-0001 · versión 1", "SHA-256 " + strings.Repeat("ab", 32)},
}

const sample = `**kaoru** · Trazabilidad a prueba de errores

# Informe de conformidad RPT-2026-0001

| Dato | Valor |
|---|---|
| PKG | PKG-2026-0001 — Pagos (noviembre) |

## 1. Contenido

Un párrafo con **negrita** y ` + "`código`" + `
que sigue en otra línea.

- Propuestos: 3
- Final: 2
`

// objects valida la tabla de referencias: cada desplazamiento apunta al comienzo de su objeto.
func objects(t *testing.T, pdf []byte) int {
	t.Helper()
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4\n")) || !bytes.HasSuffix(pdf, []byte("%%EOF\n")) {
		t.Fatal("no empieza ni termina como un PDF")
	}
	m := regexp.MustCompile(`startxref\n(\d+)\n`).FindSubmatch(pdf)
	if m == nil {
		t.Fatal("sin startxref")
	}
	xref, _ := strconv.Atoi(string(m[1]))
	table := string(pdf[xref:])
	if !strings.HasPrefix(table, "xref\n0 ") {
		t.Fatalf("startxref no apunta a la tabla: %q", table[:min(20, len(table))])
	}
	entries := regexp.MustCompile(`(\d{10}) 00000 n `).FindAllStringSubmatch(table, -1)
	for i, e := range entries {
		off, _ := strconv.Atoi(e[1])
		if want := fmt.Sprintf("%d 0 obj\n", i+1); !bytes.HasPrefix(pdf[off:], []byte(want)) {
			t.Errorf("la referencia %d no apunta a su objeto", i+1)
		}
	}
	for _, m := range regexp.MustCompile(`/Length (\d+) >>\nstream\n`).FindAllSubmatchIndex(pdf, -1) {
		n, _ := strconv.Atoi(string(pdf[m[2]:m[3]]))
		if !bytes.HasPrefix(pdf[m[1]+n:], []byte("endstream")) {
			t.Errorf("un /Length no coincide con su stream")
		}
	}
	return len(entries)
}

func pages(pdf []byte) int {
	m := regexp.MustCompile(`/Count (\d+)`).FindSubmatch(pdf)
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

func TestRenderWritesAValidPDF(t *testing.T) {
	pdf := Render(sample, options)
	if n := objects(t, pdf); n != 7 {
		t.Errorf("objetos = %d, se esperaban 7 (5 fijos y 2 por página)", n)
	}
	if pages(pdf) != 1 {
		t.Errorf("páginas = %d", pages(pdf))
	}
	for _, want := range []string{
		"(Informe de conformidad RPT-2026-0001) Tj",     // título y encabezado
		"(PKG-2026-0001 \x97 Pagos \\(noviembre\\)) Tj", // WinAnsi y paréntesis escapados
		"/F2 9.50 Tf", "(negrita ) Tj", "(y c\xf3digo que sigue en otra l\xednea.) Tj", // negrita; sin la marca de código
		"(SHA-256 " + strings.Repeat("ab", 32) + ") Tj", // la huella en el pie
		"(P\xe1gina 1 de 1) Tj", "(\x95) Tj", "/CreationDate (D:20261112150000Z)",
	} {
		if !bytes.Contains(pdf, []byte(want)) {
			t.Errorf("falta %q", want)
		}
	}
	if bytes.Contains(pdf, []byte("**")) || bytes.Contains(pdf, []byte("|---|")) {
		t.Error("quedaron marcas de Markdown en el PDF")
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	if !bytes.Equal(Render(sample, options), Render(sample, options)) {
		t.Fatal("el mismo Markdown dio dos PDF distintos")
	}
}

func TestLongTablesBreakPagesAndRepeatTheirHeader(t *testing.T) {
	var b strings.Builder
	b.WriteString("# Largo\n\n| TC | Resultado | Notas |\n|---|---|---|\n")
	for i := range 200 {
		fmt.Fprintf(&b, "| TC-2026-%04d | Aprobado | Una nota que ocupa algo de espacio en la celda |\n", i)
	}
	pdf := Render(b.String(), options)
	objects(t, pdf)
	n := pages(pdf)
	if n < 3 {
		t.Fatalf("páginas = %d", n)
	}
	if got := bytes.Count(pdf, []byte("(Resultado) Tj")); got != n {
		t.Errorf("el encabezado de la tabla salió %d veces en %d páginas", got, n)
	}
	if !bytes.Contains(pdf, []byte(fmt.Sprintf("(P\xe1gina %d de %d) Tj", n, n))) {
		t.Error("falta el número de la última página")
	}
}

func TestWrapKeepsEveryLineInsideItsWidth(t *testing.T) {
	ws := words("una palabra **larguísima** "+strings.Repeat("0123456789abcdef", 8), false)
	for _, line := range wrap(ws, tableSize, 80) {
		width := 0.0
		for i, w := range line {
			if i > 0 {
				width += textWidth(" ", tableSize, false)
			}
			width += textWidth(w.text, tableSize, w.bold)
		}
		if width > 80 {
			t.Errorf("un renglón mide %.1f de 80", width)
		}
	}
}

func TestEncodeUsesWinAnsi(t *testing.T) {
	if got := string(encode("ñ—€→")); got != "\xf1\x97\x80?" {
		t.Errorf("encode = %q", got)
	}
}
