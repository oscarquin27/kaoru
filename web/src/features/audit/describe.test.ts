// Cómo se lee un registro de auditoría: lo que se agrega a una entidad y las decisiones de producción.
import { describe, expect, it } from 'vitest'
import type { AuditEntry } from '../../shared/api/types.ts'
import { ANA, CHECKSUM, JUAN } from '../../test-data.ts'
import { describeEntry } from './describe.ts'

const entry = (fields: Partial<AuditEntry>): AuditEntry => ({ seq: 1, occurredAt: '2026-11-12T10:00:00Z', actor: JUAN, entityType: 'checklist', action: 'create', hash: CHECKSUM, ...fields })

describe('describeEntry', () => {
  it('un enlace se agrega; el dato con el que nace una entidad no', () => {
    expect(describeEntry(entry({ entityType: 'requirement', entityCode: 'REQ-2026-0001', field: 'link', newValue: 'Spec' }))).toBe('Juan Díaz agregó un enlace a requerimiento REQ-2026-0001: Spec.')
    expect(describeEntry(entry({ entityCode: 'CHK-2026-0001', field: 'package', newValue: 'PKG-2026-0001' }))).toBe('Juan Díaz creó checklist de producción CHK-2026-0001: PKG-2026-0001.')
  })

  it('la firma de un paso y la revisión de un DIAG se dicen con el valor en español', () => {
    expect(describeEntry(entry({ entityCode: 'CHK-2026-0001', action: 'decision', field: 'step.2', newValue: 'signed', reason: 'Aplicada' }))).toBe(
      'Juan Díaz marcó el paso 2 de checklist de producción CHK-2026-0001 como firmado. Comentario: Aplicada',
    )
    expect(describeEntry(entry({ actor: ANA, entityType: 'diagnostic', entityCode: 'DIAG-2026-0001', action: 'decision', field: 'review', newValue: 'non_conforming' }))).toBe(
      'Ana Rojas marcó la revisión de diagnóstico DIAG-2026-0001 como no conforme.',
    )
  })
})
