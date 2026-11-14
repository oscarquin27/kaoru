import { apiUrl, type QueryParams } from '../../shared/api/client.ts'
import { useApiQuery } from '../../shared/api/query.ts'
import type { Matrix, RequirementOverview } from '../../shared/api/types.ts'

/** Ficha consolidada del REQ: línea de vida, qué falta, cobertura y entidades relacionadas. */
export const useRequirementOverview = (code: string) => useApiQuery<RequirementOverview>(`/api/requirements/${code}/overview`)
export const useMatrix = (filters?: QueryParams) => useApiQuery<Matrix>('/api/traceability/matrix', filters)

export function matrixExportUrl(format: 'csv' | 'xlsx', filters: QueryParams): string {
  return apiUrl('/api/traceability/matrix/export', { format, ...filters })
}
