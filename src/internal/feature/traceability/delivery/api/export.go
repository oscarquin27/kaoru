package api

import (
	"archive/zip"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"kaoru/internal/feature/traceability/domain"
)

var matrixHeader = []string{
	"REQ", "Título", "Sistema", "Estado", "CHG", "RISK", "Riesgo", "Responsable", "PKG", "RPT", "UC",
	"TC", "Título del TC", "Crítico", "Último RUN", "Resultado", "BUILD", "Ambiente", "Ejecutado", "DEF",
}

var (
	riskLabels     = map[string]string{"high": "Alto", "medium": "Medio", "low": "Bajo"}
	resultLabels   = map[string]string{"passed": "Aprobado", "failed": "Fallido", "blocked": "Bloqueado"}
	severityLabels = map[string]string{"high": "alta", "medium": "media", "low": "baja"}
)

// matrixCells arma las celdas de una fila de la matriz, con textos en español.
func matrixCells(r domain.MatrixRow) []string {
	critical, executed := "", ""
	if r.TestCase.Code != "" {
		critical = "No"
		if r.TestCase.IsCritical {
			critical = "Sí"
		}
	}
	if r.TestCase.LastRun.Code != "" {
		executed = r.TestCase.LastRun.ExecutedAt.UTC().Format(time.RFC3339)
	}
	defects := make([]string, len(r.Defects))
	for i, d := range r.Defects {
		defects[i] = fmt.Sprintf("%s (%s, %s)", d.Code, severityLabels[d.Severity], d.State.Name)
	}
	cells := []string{
		r.RequirementCode, r.RequirementTitle, r.SystemKey, r.State.Name, r.ChangeCode, r.RiskCode,
		riskLabels[r.RiskLevel], r.OwnerName, strings.Join(r.Packages, ", "), strings.Join(r.Reports, ", "),
		strings.Join(r.FunctionalCases, ", "),
		r.TestCase.Code, r.TestCase.Title, critical, r.TestCase.LastRun.Code, resultLabels[r.TestCase.LastRun.Result],
		r.TestCase.LastRun.BuildVersion, r.TestCase.LastRun.EnvironmentName, executed, strings.Join(defects, "; "),
	}
	for i, cell := range cells {
		cells[i] = safeCell(cell)
	}
	return cells
}

// safeCell impide que una hoja de cálculo interprete como fórmula un texto escrito por un usuario.
func safeCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// writeCSV escribe la matriz en CSV, con BOM para que Excel lea bien los acentos.
func writeCSV(w io.Writer, rows []domain.MatrixRow) error {
	if _, err := w.Write([]byte("\xEF\xBB\xBF")); err != nil {
		return err
	}
	out := csv.NewWriter(w)
	if err := out.Write(matrixHeader); err != nil {
		return err
	}
	for _, r := range rows {
		if err := out.Write(matrixCells(r)); err != nil {
			return err
		}
	}
	out.Flush()
	return out.Error()
}

// Las partes fijas de un XLSX mínimo: una sola hoja con texto en línea.
const (
	xlsxContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`
	xlsxRootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`
	xlsxWorkbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Matriz" sheetId="1" r:id="rId1"/></sheets></workbook>`
	xlsxWorkbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`
	xlsxSheetOpen = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`
	xlsxSheetClose = `</sheetData></worksheet>`
)

// writeXLSX escribe la matriz como un libro de Excel con la biblioteca estándar: un zip con
// cinco partes XML. No hace falta una librería para una sola hoja de texto.
func writeXLSX(w io.Writer, rows []domain.MatrixRow) error {
	archive := zip.NewWriter(w)
	for name, content := range map[string]string{
		"[Content_Types].xml":        xlsxContentTypes,
		"_rels/.rels":                xlsxRootRels,
		"xl/workbook.xml":            xlsxWorkbook,
		"xl/_rels/workbook.xml.rels": xlsxWorkbookRels,
	} {
		part, err := archive.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(part, content); err != nil {
			return err
		}
	}
	sheet, err := archive.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return err
	}
	if _, err := io.WriteString(sheet, xlsxSheetOpen); err != nil {
		return err
	}
	if err := writeXLSXRow(sheet, matrixHeader); err != nil {
		return err
	}
	for _, r := range rows {
		if err := writeXLSXRow(sheet, matrixCells(r)); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(sheet, xlsxSheetClose); err != nil {
		return err
	}
	return archive.Close()
}

func writeXLSXRow(w io.Writer, cells []string) error {
	var b strings.Builder
	b.WriteString("<row>")
	for _, cell := range cells {
		b.WriteString(`<c t="inlineStr"><is><t xml:space="preserve">`)
		if err := xml.EscapeText(&b, []byte(cell)); err != nil {
			return err
		}
		b.WriteString("</t></is></c>")
	}
	b.WriteString("</row>")
	_, err := io.WriteString(w, b.String())
	return err
}
