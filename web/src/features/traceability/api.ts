import { apiUrl, type QueryParams } from '../../shared/api/client.ts'
import { useApiQuery } from '../../shared/api/query.ts'
import type { Change, Matrix, Page, RequirementOverview } from '../../shared/api/types.ts'

/** Ficha consolidada del REQ: línea de vida, qué falta, cobertura y entidades relacionadas. */
export const useRequirementOverview = (code: string) => useApiQuery<RequirementOverview>(`/api/requirements/${code}/overview`)
export const useMatrix = (filters?: QueryParams) => useApiQuery<Matrix>('/api/traceability/matrix', filters)
/** Los CHG, para el filtro de la matriz. */
export const useChangeOptions = () => useApiQuery<Page<Change>>('/api/changes')

export function matrixExportUrl(format: 'csv' | 'xlsx', filters: QueryParams): string {
  return apiUrl('/api/traceability/matrix/export', { format, ...filters })
}
