import { api } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Risk, RiskInput } from '../../shared/api/types.ts'

/** RISK vigente. Solo se pide si el REQ ya tiene el riesgo evaluado. */
export const useRisk = (code: string, assessed: boolean) => useApiQuery<Risk>(`/api/requirements/${code}/risk`, undefined, assessed)
export const useAssessRisk = (code: string) => useApiMutation((input: RiskInput) => api.put<Risk>(`/api/requirements/${code}/risk`, input))
