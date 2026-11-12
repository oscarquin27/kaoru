// Misma tabla de casos que las pruebas de Go (reglas §A.1.2).
import { describe, expect, it } from 'vitest'
import cases from '../../../../api/testdata/validation-cases.json' with { type: 'json' }
import { parseSheetLink, sheetPreviewUrl } from './sheetLink.ts'

describe('sheetLink', () => {
  it.each(cases.sheetLink)('$name', ({ input, valid, normalized }) => {
    const parsed = parseSheetLink(input)
    expect(parsed !== null).toBe(valid)
    if (valid) expect(parsed).toEqual(normalized)
  })

  it('arma la URL de la vista previa con la pestaña y el rango', () => {
    expect(sheetPreviewUrl({ id: 'abc', gid: '7', range: 'A1:B2' })).toBe('https://docs.google.com/spreadsheets/d/abc/preview?gid=7&range=A1%3AB2')
    expect(sheetPreviewUrl({ id: 'abc' })).toBe('https://docs.google.com/spreadsheets/d/abc/preview')
  })
})
