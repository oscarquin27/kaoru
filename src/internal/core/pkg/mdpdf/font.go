package mdpdf

// Anchos de las fuentes estándar Helvetica y Helvetica-Bold en milésimas del tamaño, de la 32 a la 126.
// Vienen de sus métricas AFM: son las fuentes que todo lector de PDF trae, así que el archivo no embebe ninguna.
var (
	regularWidths = [95]int{
		278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278,
		556, 556, 556, 556, 556, 556, 556, 556, 556, 556,
		278, 278, 584, 584, 584, 556, 1015,
		667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778, 667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611,
		278, 278, 278, 469, 556, 333,
		556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556, 556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500,
		334, 260, 334, 584,
	}
	boldWidths = [95]int{
		278, 333, 474, 556, 556, 889, 722, 238, 333, 333, 389, 584, 278, 333, 278, 278,
		556, 556, 556, 556, 556, 556, 556, 556, 556, 556,
		333, 333, 584, 584, 584, 611, 975,
		722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778, 667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611,
		333, 278, 333, 584, 556, 333,
		556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611, 611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500,
		389, 280, 389, 584,
	}
	// base mide una letra acentuada como la letra sin acento; la diferencia no llega a mover un renglón.
	base = map[rune]rune{
		'á': 'a', 'à': 'a', 'â': 'a', 'ä': 'a', 'ã': 'a', 'å': 'a', 'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
		'í': 'l', 'ì': 'l', 'î': 'l', 'ï': 'l', 'ó': 'o', 'ò': 'o', 'ô': 'o', 'ö': 'o', 'õ': 'o',
		'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u', 'ñ': 'n', 'ç': 'c', 'ý': 'y', 'ÿ': 'y',
		'Á': 'A', 'À': 'A', 'Â': 'A', 'Ä': 'A', 'Ã': 'A', 'Å': 'A', 'É': 'E', 'È': 'E', 'Ê': 'E', 'Ë': 'E',
		'Í': 'I', 'Ì': 'I', 'Î': 'I', 'Ï': 'I', 'Ó': 'O', 'Ò': 'O', 'Ô': 'O', 'Ö': 'O', 'Õ': 'O',
		'Ú': 'U', 'Ù': 'U', 'Û': 'U', 'Ü': 'U', 'Ñ': 'N', 'Ç': 'C', 'Ý': 'Y',
	}
	special = map[rune]int{
		'—': 1000, '…': 1000, '–': 556, '€': 556, '«': 556, '»': 556, '¿': 611, '¡': 333, '·': 278, '•': 350,
		'¦': 280, '°': 400, 'º': 365, 'ª': 370, '“': 500, '”': 500, '‘': 278, '’': 278, ' ': 278,
	}
	// cp1252 son los caracteres de WinAnsiEncoding que no están en la misma posición que en Unicode.
	cp1252 = map[rune]byte{
		'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86, '‡': 0x87, 'ˆ': 0x88, '‰': 0x89,
		'Š': 0x8A, '‹': 0x8B, 'Œ': 0x8C, 'Ž': 0x8E, '‘': 0x91, '’': 0x92, '“': 0x93, '”': 0x94, '•': 0x95,
		'–': 0x96, '—': 0x97, '˜': 0x98, '™': 0x99, 'š': 0x9A, '›': 0x9B, 'œ': 0x9C, 'ž': 0x9E, 'Ÿ': 0x9F,
	}
)

// runeWidth es el ancho de un carácter en milésimas del tamaño de la fuente.
func runeWidth(r rune, bold bool) int {
	if b, ok := base[r]; ok {
		r = b
	}
	if r >= 32 && r <= 126 {
		if bold {
			return boldWidths[r-32]
		}
		return regularWidths[r-32]
	}
	if w, ok := special[r]; ok {
		return w
	}
	return 600
}

// textWidth es el ancho de un texto en puntos.
func textWidth(s string, size float64, bold bool) float64 {
	total := 0
	for _, r := range s {
		total += runeWidth(r, bold)
	}
	return float64(total) * size / 1000
}

// encode pasa un texto a WinAnsiEncoding. Lo que esa codificación no tiene sale como «?».
func encode(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch b, ok := cp1252[r]; {
		case ok:
			out = append(out, b)
		case r < 0x80 || (r >= 0xA0 && r <= 0xFF):
			out = append(out, byte(r)) //nolint:gosec // r cabe en un byte: es menor que 0x100
		default:
			out = append(out, '?')
		}
	}
	return out
}
