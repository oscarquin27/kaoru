import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Build, BuildInput, Page, Requirement } from '../../shared/api/types.ts'

export const useBuilds = (filters?: QueryParams) => useApiQuery<Page<Build>>('/api/builds', filters)
export const useBuild = (code: string) => useApiQuery<Build>(`/api/builds/${code}`)
export const useRegisterBuild = () => useApiMutation((input: BuildInput) => api.post<Build>('/api/builds', input))
/** Los REQ de un sistema, para elegir los que incluye el BUILD. */
export const useSystemRequirements = (system: string) => useApiQuery<Page<Requirement>>('/api/requirements', { system, size: 200 }, system !== '')
