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

/** Nombre legible del campo que registra la auditoría. Uno desconocido sale tal cual. */
function fieldName(field: string | undefined): string {
  if (!field) return 'un dato'
  if (field === 'comment') return 'un comentario'
  if (field === 'link') return 'un enlace'
  if (field.startsWith('entryContract.')) return `el ítem «${field.slice('entryContract.'.length)}» del contrato de entrada`
  return field
}

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
      if (entry.field) return `${who} agregó ${fieldName(entry.field)} a ${what}${entry.newValue ? `: ${entry.newValue}` : ''}.${reason}`
      return `${who} creó ${what}${entry.newValue ? `: ${entry.newValue}` : ''}.${reason}`
    case 'transition':
      return `${who} cambió el estado de ${what}: ${change(entry)}.${reason}`
    case 'update':
      return `${who} cambió ${fieldName(entry.field)} de ${what}${change(entry) ? `: ${change(entry)}` : ''}.${reason}`
    case 'decision':
      return `${who} registró una decisión sobre ${what}${change(entry) ? `: ${change(entry)}` : ''}.${reason}`
    case 'delete':
      return `${who} quitó ${fieldName(entry.field)} de ${what}${entry.oldValue ? `: ${entry.oldValue}` : ''}.${reason}`
  }
}
