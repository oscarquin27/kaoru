// Package mdpdf escribe un PDF a partir del Markdown que generan los informes de Kaoru: títulos, párrafos,
// listas, tablas y negrita. Usa solo la biblioteca estándar y las fuentes que todo lector de PDF trae
// (Helvetica en WinAnsiEncoding), así que no necesita red, fuentes embebidas ni CGO.
package mdpdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// Options son los datos de la página que no salen del Markdown.
type Options struct {
	// Title y Author van en las propiedades del documento.
	Title  string
	Author string
	// CreatedAt es la fecha del documento. El mismo Markdown con las mismas opciones da los mismos bytes.
	CreatedAt time.Time
	// Brand y Motto van en el encabezado de cada página: la marca en negrita y el lema a su lado.
	Brand string
	Motto string
	// Footer son los renglones del pie de cada página; el número de página va a la derecha del primero.
	Footer []string
}

// Medidas de una página A4 en puntos.
const (
	pageWidth    = 595.28
	pageHeight   = 841.89
	marginX      = 48.0
	marginTop    = 70.0
	marginBottom = 64.0
	contentWidth = pageWidth - 2*marginX

	bodySize  = 9.5
	tableSize = 7.5
	cellPad   = 3.0
)

// Render escribe el PDF del Markdown.
func Render(markdown string, o Options) []byte {
	l := &layout{}
	l.newPage()
	for _, b := range parse(markdown) {
		l.block(b)
	}
	for i, p := range l.pages {
		decorate(p, o, i+1, len(l.pages))
	}
	return write(l.pages, o)
}

// block es un bloque del Markdown: un título, un párrafo, un elemento de lista o una tabla.
type block struct {
	kind  string // "h1", "h2", "h3", "p", "li", "table"
	text  string
	table [][]string
}

// parse separa el Markdown en bloques. Entiende lo mismo que el visor de documentos de Kaoru.
func parse(markdown string) []block {
	var blocks []block
	var para []string
	flush := func() {
		if len(para) > 0 {
			blocks = append(blocks, block{kind: "p", text: strings.Join(para, " ")})
			para = nil
		}
	}
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "### "):
			flush()
			blocks = append(blocks, block{kind: "h3", text: line[4:]})
		case strings.HasPrefix(line, "## "):
			flush()
			blocks = append(blocks, block{kind: "h2", text: line[3:]})
		case strings.HasPrefix(line, "# "):
			flush()
			blocks = append(blocks, block{kind: "h1", text: line[2:]})
		case strings.HasPrefix(line, "- "):
			flush()
			blocks = append(blocks, block{kind: "li", text: line[2:]})
		case strings.HasPrefix(line, "|") && len(para) == 0:
			var rows [][]string
			for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
				cells := splitRow(strings.TrimSpace(lines[i]))
				if !isRule(cells) {
					rows = append(rows, cells)
				}
			}
			i--
			blocks = append(blocks, block{kind: "table", table: rows})
		default:
			para = append(para, line)
		}
	}
	flush()
	return blocks
}

func splitRow(line string) []string {
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	cells := strings.Split(line, "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(c)
	}
	return cells
}

func isRule(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, "-: ") != "" || c == "" {
			return false
		}
	}
	return true
}

// word es una palabra con su fuente.
type word struct {
	text string
	bold bool
}

// words parte un texto en palabras; `**` alterna la negrita y las marcas de código y cursiva se quitan.
func words(s string, bold bool) []word {
	var out []word
	for i, part := range strings.Split(s, "**") {
		b := bold != (i%2 == 1)
		part = strings.NewReplacer("`", "", "*", "").Replace(part)
		for _, f := range strings.Fields(part) {
			out = append(out, word{text: f, bold: b})
		}
	}
	return out
}

// wrap acomoda las palabras en renglones de un ancho; la palabra que no cabe sola se parte por caracteres.
func wrap(ws []word, size, width float64) [][]word {
	var lines [][]word
	var line []word
	used := 0.0
	space := textWidth(" ", size, false)
	for _, w := range ws {
		for _, piece := range split(w, size, width) {
			pw := textWidth(piece.text, size, piece.bold)
			if len(line) > 0 && used+space+pw > width {
				lines, line, used = append(lines, line), nil, 0
			}
			if len(line) > 0 {
				used += space
			}
			line, used = append(line, piece), used+pw
		}
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}

func split(w word, size, width float64) []word {
	if textWidth(w.text, size, w.bold) <= width {
		return []word{w}
	}
	var out []word
	var cur []rune
	for _, r := range w.text {
		if len(cur) > 0 && textWidth(string(append(cur, r)), size, w.bold) > width {
			out, cur = append(out, word{text: string(cur), bold: w.bold}), nil
		}
		cur = append(cur, r)
	}
	return append(out, word{text: string(cur), bold: w.bold})
}

// layout coloca los bloques en páginas. y es la distancia desde el borde de arriba.
type layout struct {
	pages []*bytes.Buffer
	page  *bytes.Buffer
	y     float64
}

func (l *layout) newPage() {
	l.page = &bytes.Buffer{}
	l.pages = append(l.pages, l.page)
	l.y = marginTop
}

// ensure pasa a otra página si lo que sigue no cabe.
func (l *layout) ensure(height float64) {
	if l.y+height > pageHeight-marginBottom && l.y > marginTop {
		l.newPage()
	}
}

func (l *layout) block(b block) {
	switch b.kind {
	case "h1":
		l.heading(b.text, 16, 10)
	case "h2":
		l.heading(b.text, 12.5, 14)
	case "h3":
		l.heading(b.text, 10.5, 8)
	case "p":
		l.paragraph(words(b.text, false), marginX, contentWidth)
	case "li":
		lines := wrap(words(b.text, false), bodySize, contentWidth-12)
		l.ensure(bodySize * 1.4)
		text(l.page, marginX+3, l.y+bodySize, bodySize, false, "•")
		l.lines(lines, marginX+12, bodySize)
		l.y += 2
	case "table":
		l.table(b.table)
	}
}

func (l *layout) heading(s string, size, before float64) {
	l.y += before
	lines := wrap(words(s, true), size, contentWidth)
	// Un título no queda solo al pie de la página.
	l.ensure(float64(len(lines))*size*1.3 + 3*bodySize)
	l.lines(lines, marginX, size)
	l.y += size * 0.4
}

func (l *layout) paragraph(ws []word, x, width float64) {
	l.lines(wrap(ws, bodySize, width), x, bodySize)
	l.y += bodySize * 0.7
}

// lines escribe renglones ya acomodados; pasa de página entre renglones si hace falta.
func (l *layout) lines(lines [][]word, x, size float64) {
	leading := size * 1.35
	for _, line := range lines {
		l.ensure(leading)
		drawLine(l.page, x, l.y+size, size, line)
		l.y += leading
	}
}

// table reparte el ancho entre las columnas según lo que ocupa cada una y repite el encabezado en cada página.
func (l *layout) table(rows [][]string) {
	if len(rows) == 0 {
		return
	}
	columns := len(rows[0])
	widths := columnWidths(rows, columns)
	leading := tableSize * 1.3
	cellLines := func(row []string, bold bool) ([][][]word, float64) {
		out := make([][][]word, columns)
		height := 0.0
		for c := range columns {
			cell := ""
			if c < len(row) {
				cell = row[c]
			}
			out[c] = wrap(words(cell, bold), tableSize, widths[c]-2*cellPad)
			height = max(height, float64(max(len(out[c]), 1))*leading)
		}
		return out, height + 2*cellPad
	}
	drawRow := func(row []string, header bool) {
		cells, height := cellLines(row, header)
		x := marginX
		if header {
			fmt.Fprintf(l.page, "0.93 g %.2f %.2f %.2f %.2f re f 0 g\n", x, pageHeight-l.y-height, contentWidth, height)
		}
		fmt.Fprintf(l.page, "0.75 G 0.5 w %.2f %.2f %.2f %.2f re S 0 G\n", x, pageHeight-l.y-height, contentWidth, height)
		for c, lines := range cells {
			if c > 0 {
				fmt.Fprintf(l.page, "0.75 G 0.5 w %.2f %.2f m %.2f %.2f l S 0 G\n", x, pageHeight-l.y, x, pageHeight-l.y-height)
			}
			y := l.y + cellPad
			for _, line := range lines {
				drawLine(l.page, x+cellPad, y+tableSize, tableSize, line)
				y += leading
			}
			x += widths[c]
		}
		l.y += height
	}

	_, headerHeight := cellLines(rows[0], true)
	l.y += 2
	for i, row := range rows {
		if i == 0 {
			_, first := cellLines(rows[min(1, len(rows)-1)], false)
			l.ensure(headerHeight + first)
			drawRow(row, true)
			continue
		}
		_, height := cellLines(row, false)
		if l.y+height > pageHeight-marginBottom {
			l.newPage()
			drawRow(rows[0], true)
		}
		drawRow(row, false)
	}
	l.y += bodySize
}

// columnWidths da a cada columna lo que pide su texto más largo. Si no caben todas, ninguna baja de su
// palabra más larga (un código no se parte) y las demás ceden en proporción a lo que piden de más.
func columnWidths(rows [][]string, columns int) []float64 {
	natural, floor := make([]float64, columns), make([]float64, columns)
	for r, row := range rows {
		for c := 0; c < columns && c < len(row); c++ {
			w := 0.0
			for _, wd := range words(row[c], r == 0) {
				ww := textWidth(wd.text, tableSize, wd.bold)
				w += ww + textWidth(" ", tableSize, false)
				floor[c] = max(floor[c], ww)
			}
			natural[c] = max(natural[c], min(w, 220))
		}
	}
	sumNatural, sumFloor := 0.0, 0.0
	for c := range columns {
		floor[c] = min(max(floor[c]+2*cellPad+1, 30), 140)
		natural[c] = max(natural[c]+2*cellPad+1, floor[c])
		sumNatural, sumFloor = sumNatural+natural[c], sumFloor+floor[c]
	}
	widths := make([]float64, columns)
	for c := range columns {
		switch {
		case sumNatural <= contentWidth:
			widths[c] = natural[c] + (contentWidth-sumNatural)/float64(columns)
		case sumFloor >= contentWidth:
			widths[c] = floor[c] * contentWidth / sumFloor
		default:
			widths[c] = floor[c] + (natural[c]-floor[c])*(contentWidth-sumFloor)/(sumNatural-sumFloor)
		}
	}
	return widths
}

// decorate dibuja el encabezado y el pie de una página.
func decorate(page *bytes.Buffer, o Options, number, total int) {
	y := 34.0
	text(page, marginX, y, 12, true, o.Brand)
	text(page, marginX+textWidth(o.Brand, 12, true)+8, y, 8.5, false, o.Motto)
	if o.Title != "" {
		text(page, pageWidth-marginX-textWidth(o.Title, 8, false), y, 8, false, o.Title)
	}
	fmt.Fprintf(page, "0.6 G 0.6 w %.2f %.2f m %.2f %.2f l S 0 G\n", marginX, pageHeight-y-8, pageWidth-marginX, pageHeight-y-8)

	y = pageHeight - marginBottom + 22
	fmt.Fprintf(page, "0.6 G 0.6 w %.2f %.2f m %.2f %.2f l S 0 G\n", marginX, pageHeight-y+10, pageWidth-marginX, pageHeight-y+10)
	page.WriteString("0.35 g\n")
	for i, line := range o.Footer {
		text(page, marginX, y+float64(i)*10, 7, false, line)
	}
	label := fmt.Sprintf("Página %d de %d", number, total)
	text(page, pageWidth-marginX-textWidth(label, 7, false), y, 7, false, label)
	page.WriteString("0 g\n")
}

// drawLine escribe un renglón; las palabras seguidas con la misma fuente van en un solo texto,
// para que copiar del PDF dé la frase entera.
func drawLine(page *bytes.Buffer, x, y, size float64, line []word) {
	for i := 0; i < len(line); {
		j := i + 1
		for j < len(line) && line[j].bold == line[i].bold {
			j++
		}
		parts := make([]string, j-i)
		for k := i; k < j; k++ {
			parts[k-i] = line[k].text
		}
		run := strings.Join(parts, " ")
		if j < len(line) {
			run += " "
		}
		text(page, x, y, size, line[i].bold, run)
		x += textWidth(run, size, line[i].bold)
		i = j
	}
}

// text escribe un texto con la línea base a y puntos del borde de arriba.
func text(page *bytes.Buffer, x, y, size float64, bold bool, s string) {
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(page, "BT /%s %.2f Tf %.2f %.2f Td (%s) Tj ET\n", font, size, x, pageHeight-y, escape(encode(s)))
}

func escape(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c == '\\' || c == '(' || c == ')' {
			out = append(out, '\\')
		}
		out = append(out, c)
	}
	return out
}

// write arma el archivo: catálogo, páginas, las dos fuentes, las propiedades y una tabla de referencias.
func write(pages []*bytes.Buffer, o Options) []byte {
	var out bytes.Buffer
	var offsets []int
	object := func(body string) {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", len(offsets), body)
	}
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	const firstPage = 6
	kids := make([]string, len(pages))
	for i := range pages {
		kids[i] = fmt.Sprintf("%d 0 R", firstPage+2*i)
	}
	object("<< /Type /Catalog /Pages 2 0 R >>")
	object(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	object("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	object("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	object(fmt.Sprintf("<< /Title (%s) /Author (%s) /Producer (%s) /CreationDate (D:%s) >>",
		escape(encode(o.Title)), escape(encode(o.Author)), escape(encode(o.Brand)), o.CreatedAt.UTC().Format("20060102150405Z")))
	for i, p := range pages {
		object(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>",
			pageWidth, pageHeight, firstPage+2*i+1))
		object(fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", p.Len(), p.Bytes()))
	}

	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R /Info 5 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xref)
	return out.Bytes()
}
