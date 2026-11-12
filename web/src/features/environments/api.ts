import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Deployment, DeploymentInput, Environment, EnvironmentInput, EnvironmentPatch, Page } from '../../shared/api/types.ts'

export const useEnvironments = (filters?: QueryParams) => useApiQuery<Page<Environment>>('/api/environments', filters)
export const useEnvironment = (code: string) => useApiQuery<Environment>(`/api/environments/${code}`)
export const useCreateEnvironment = () => useApiMutation((input: EnvironmentInput) => api.post<Environment>('/api/environments', input))
export const useUpdateEnvironment = (code: string) => useApiMutation((patch: EnvironmentPatch) => api.patch<Environment>(`/api/environments/${code}`, patch))

export const useDeployments = (filters?: QueryParams) => useApiQuery<Page<Deployment>>('/api/deployments', filters)
export const useDeployment = (code: string) => useApiQuery<Deployment>(`/api/deployments/${code}`)
export const useRegisterDeployment = () => useApiMutation((input: DeploymentInput) => api.post<Deployment>('/api/deployments', input))
