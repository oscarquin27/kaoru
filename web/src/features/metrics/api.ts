import type { QueryParams } from '../../shared/api/client.ts'
import { useApiQuery } from '../../shared/api/query.ts'
import type { CoverageMetrics, DefectStateMetrics, EscapedDefectMetrics, RegressionMetrics } from '../../shared/api/types.ts'

export const useCoverageMetrics = (filters: QueryParams) => useApiQuery<CoverageMetrics>('/api/metrics/coverage', filters)
export const useDefectStateMetrics = (filters: QueryParams) => useApiQuery<DefectStateMetrics>('/api/metrics/defect-states', filters)
export const useRegressionMetrics = (filters: QueryParams) => useApiQuery<RegressionMetrics>('/api/metrics/regression', filters)
export const useEscapedDefectMetrics = (filters: QueryParams) => useApiQuery<EscapedDefectMetrics>('/api/metrics/escaped-defects', filters)
