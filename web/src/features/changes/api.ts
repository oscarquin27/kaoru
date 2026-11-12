import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Change, ChangeInput, ChangePatch, Page, Requirement, TransitionInput } from '../../shared/api/types.ts'

export const useChanges = (filters?: QueryParams) => useApiQuery<Page<Change>>('/api/changes', filters)
export const useChange = (code: string) => useApiQuery<Change>(`/api/changes/${code}`)
export const useChangeRequirements = (code: string) => useApiQuery<Page<Requirement>>('/api/requirements', { change: code })

export const useCreateChange = () => useApiMutation((input: ChangeInput) => api.post<Change>('/api/changes', input))
export const useUpdateChange = (code: string) => useApiMutation((patch: ChangePatch) => api.patch<Change>(`/api/changes/${code}`, patch))
export const useTransitionChange = (code: string) =>
  useApiMutation((input: TransitionInput) => api.post<Change>(`/api/changes/${code}/transitions`, input))
