// Un registro de auditoría dicho en una frase.
import type { AuditEntry } from '../../shared/api/types.ts'
import { ENTITY_LABELS, labelOf } from '../../shared/lib/labels.ts'

export const ACTION_LABELS: Record<AuditEntry['action'], string> = {
  create: 'Creación',
  update: 'Edición',
  transition: 'Cambio de estado',
  decision: 'Decisión',
  delete: 'Eliminación',
}

// Campos de la planificación (fase 6); los de fases anteriores se muestran con su clave.
const FIELD_NAMES: Record<string, string> = {
  startDate: 'la fecha de inicio',
  endDate: 'la fecha final',
  operators: 'los operadores',
  cancelled: 'la cancelación',
  kind: 'la clase',
  title: 'el título',
  notes: 'las notas',
  change: 'el CHG',
}

/** Nombre legible del campo que registra la auditoría. Uno desconocido sale tal cual. */
function fieldName(field: string | undefined): string {
  if (!field) return 'un dato'
  if (field === 'comment') return 'un comentario'
  if (field === 'link') return 'un enlace'
  if (field.startsWith('entryContract.')) return `el ítem «${field.slice('entryContract.'.length)}» del contrato de entrada`
  if (field.startsWith('step.')) return `el paso ${field.slice('step.'.length)}`
  if (field === 'review') return 'la revisión'
  return FIELD_NAMES[field] ?? field
}

// Lo que se agrega a una entidad que ya existe. Con otro campo, el registro de creación solo dice
// con qué dato nació (el PKG del CHK, el DEP del DIAG, el resultado del SMOKE).
const PARTS = new Set(['comment', 'link'])

// Valores de las decisiones de producción, dichos como en pantalla.
const DECISION_VALUES: Record<string, string> = { signed: 'firmado', conforming: 'conforme', non_conforming: 'no conforme' }

function change(entry: AuditEntry): string {
  if (entry.oldValue && entry.newValue) return `${entry.oldValue} → ${entry.newValue}`
  return entry.newValue ?? entry.oldValue ?? ''
}

export function describeEntry(entry: AuditEntry): string {
  const who = entry.actor.displayName
  const what = `${labelOf(ENTITY_LABELS, entry.entityType).toLowerCase()} ${entry.entityCode ?? ''}`.trim()
  const reason = entry.reason ? ` Motivo: ${entry.reason}` : ''
  switch (entry.action) {
    case 'create':
      // Con campo, lo creado es una parte de la entidad (un enlace, un comentario).
      if (entry.field && PARTS.has(entry.field)) return `${who} agregó ${fieldName(entry.field)} a ${what}${entry.newValue ? `: ${entry.newValue}` : ''}.${reason}`
      return `${who} creó ${what}${entry.newValue ? `: ${entry.newValue}` : ''}.${reason}`
    case 'transition':
      return `${who} cambió el estado de ${what}: ${change(entry)}.${reason}`
    case 'update':
      return `${who} cambió ${fieldName(entry.field)} de ${what}${change(entry) ? `: ${change(entry)}` : ''}.${reason}`
    case 'decision':
      if (entry.field?.startsWith('step.') || entry.field === 'review') {
        // El motivo de una firma es su comentario; el de una revisión, sus notas.
        const note = entry.reason ? ` ${entry.field === 'review' ? 'Notas' : 'Comentario'}: ${entry.reason}` : ''
        return `${who} marcó ${fieldName(entry.field)} de ${what} como ${DECISION_VALUES[entry.newValue ?? ''] ?? entry.newValue}.${note}`
      }
      return `${who} registró una decisión sobre ${what}${change(entry) ? `: ${change(entry)}` : ''}.${reason}`
    case 'delete':
      return `${who} quitó ${fieldName(entry.field)} de ${what}${entry.oldValue ? `: ${entry.oldValue}` : ''}.${reason}`
  }
}
