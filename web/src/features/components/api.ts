import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Component, ComponentInput, ComponentPatch, Page } from '../../shared/api/types.ts'

export const useComponents = (filters?: QueryParams) => useApiQuery<Page<Component>>('/api/components', filters)
export const useComponent = (code: string) => useApiQuery<Component>(`/api/components/${code}`)
export const useCreateComponent = () => useApiMutation((input: ComponentInput) => api.post<Component>('/api/components', input))
export const useUpdateComponent = () =>
  useApiMutation(({ code, patch }: { code: string; patch: ComponentPatch }) => api.patch<Component>(`/api/components/${code}`, patch))
