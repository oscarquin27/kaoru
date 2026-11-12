import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Defect, DefectInput, DefectPatch, Page, TransitionInput } from '../../shared/api/types.ts'

export const useDefects = (filters?: QueryParams) => useApiQuery<Page<Defect>>('/api/defects', filters)
export const useDefect = (code: string) => useApiQuery<Defect>(`/api/defects/${code}`)

export const useCreateDefect = () => useApiMutation((input: DefectInput) => api.post<Defect>('/api/defects', input))
export const useUpdateDefect = (code: string) => useApiMutation((patch: DefectPatch) => api.patch<Defect>(`/api/defects/${code}`, patch))
export const useTransitionDefect = (code: string) =>
  useApiMutation((input: TransitionInput) => api.post<Defect>(`/api/defects/${code}/transitions`, input))
