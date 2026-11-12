import { apiUrl, type QueryParams } from '../../shared/api/client.ts'
import { useApiQuery } from '../../shared/api/query.ts'
import type { AuditEntry, AuditVerification, Page } from '../../shared/api/types.ts'

export const useAudit = (filters?: QueryParams) => useApiQuery<Page<AuditEntry>>('/api/audit', filters)
/** Recalcula la cadena de hash. Solo se pide cuando alguien lo solicita. */
export const useAuditVerification = (enabled: boolean) => useApiQuery<AuditVerification>('/api/audit/verification', undefined, enabled)

export function auditExportUrl(filters: QueryParams): string {
  return apiUrl('/api/audit/export', filters)
}

/** Un día del filtro (`2026-11-11`) como instante: el inicio del día, o el final con `end`. */
export function dayToInstant(day: string | undefined, end = false): string | undefined {
  if (!day) return undefined
  return new Date(`${day}T${end ? '23:59:59.999' : '00:00:00'}`).toISOString()
}
