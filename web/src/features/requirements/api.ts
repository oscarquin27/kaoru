import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Change, EntryContractItem, Items, Link, LinkInput, LinkPatch, Page, Requirement, RequirementInput, RequirementPatch, TransitionInput } from '../../shared/api/types.ts'

export const useRequirements = (filters?: QueryParams) => useApiQuery<Page<Requirement>>('/api/requirements', filters)
/** Los CHG, para elegir a cuál pertenece un REQ. */
export const useChangeOptions = () => useApiQuery<Page<Change>>('/api/changes')

export const useCreateRequirement = () => useApiMutation((input: RequirementInput) => api.post<Requirement>('/api/requirements', input))
export const useUpdateRequirement = (code: string) =>
  useApiMutation((patch: RequirementPatch) => api.patch<Requirement>(`/api/requirements/${code}`, patch))
export const useTransitionRequirement = (code: string) =>
  useApiMutation((input: TransitionInput) => api.post<Requirement>(`/api/requirements/${code}/transitions`, input))

export const useLinks = (code: string) => useApiQuery<Items<Link>>(`/api/requirements/${code}/links`)
export const useCreateLink = (code: string) => useApiMutation((input: LinkInput) => api.post<Link>(`/api/requirements/${code}/links`, input))
export const useUpdateLink = (code: string) =>
  useApiMutation(({ id, patch }: { id: string; patch: LinkPatch }) => api.patch<Link>(`/api/requirements/${code}/links/${id}`, patch))
export const useDeleteLink = (code: string) => useApiMutation((id: string) => api.delete(`/api/requirements/${code}/links/${id}`))

export const useEntryContract = (code: string) => useApiQuery<Items<EntryContractItem>>(`/api/requirements/${code}/entry-contract`)
export const useToggleEntryContractItem = (code: string) =>
  useApiMutation(async ({ key, done }: { key: string; done: boolean }) => {
    const path = `/api/requirements/${code}/entry-contract/${key}`
    if (done) await api.put<EntryContractItem>(path, {})
    else await api.delete(path)
  })
