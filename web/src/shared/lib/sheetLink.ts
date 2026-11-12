// Regla `sheetLink`: misma que ParseSheetLink del dominio (requirement/domain/link.go).
// Las dos implementaciones pasan la tabla api/testdata/validation-cases.json (reglas §A.1.2).

export interface SheetRef {
  id: string
  gid: string
  range: string
}

const SHEET_ID = /^[A-Za-z0-9_-]{10,}$/
const GID = /^\d+$/
const RANGE = /^[A-Za-z0-9_!':$ ]{1,100}$/

/** Extrae el ID de la hoja, la pestaña (gid) y el rango. Solo valida la forma de la URL. */
export function parseSheetLink(raw: string): SheetRef | null {
  let url: URL
  try {
    url = new URL(raw.trim())
  } catch {
    return null
  }
  if (url.protocol !== 'https:' || url.host !== 'docs.google.com') return null
  const segments = url.pathname.split('/').filter(Boolean)
  if (segments.length < 3 || segments[0] !== 'spreadsheets') return null
  const marker = segments.indexOf('d', 1)
  const id = marker === -1 ? '' : (segments[marker + 1] ?? '')
  if (!SHEET_ID.test(id)) return null

  const ref: SheetRef = { id, gid: '', range: '' }
  // La pestaña y el rango pueden venir en la consulta o en el fragmento; manda el fragmento.
  for (const params of [url.searchParams, new URLSearchParams(url.hash.slice(1))]) {
    const gid = params.get('gid') ?? ''
    if (GID.test(gid)) ref.gid = gid
    const range = params.get('range') ?? ''
    if (RANGE.test(range)) ref.range = range
  }
  return ref
}

/** URL de la vista previa embebida de una hoja (solo lectura, la sirve Google). */
export function sheetPreviewUrl(sheet: { id: string; gid?: string; range?: string }): string {
  const params = new URLSearchParams()
  if (sheet.gid) params.set('gid', sheet.gid)
  if (sheet.range) params.set('range', sheet.range)
  const search = params.toString()
  return `https://docs.google.com/spreadsheets/d/${sheet.id}/preview${search ? `?${search}` : ''}`
}
