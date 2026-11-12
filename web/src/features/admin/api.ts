import { api } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { BlockingRule, Role, RolePermissions, Workflow } from '../../shared/api/types.ts'

export const useRolePermissions = (enabled = true) => useApiQuery<RolePermissions>('/api/admin/role-permissions', undefined, enabled)
export const useSetRolePermissions = () =>
  useApiMutation(({ role, permissions }: { role: Role; permissions: string[] }) =>
    api.put<RolePermissions>(`/api/admin/role-permissions/${role}`, { permissions }),
  )
export const useUpdateBlockingRule = () =>
  useApiMutation(({ id, patch }: { id: string; patch: { enabled?: boolean; blocks?: boolean } }) =>
    api.patch<BlockingRule>(`/api/admin/blocking-rules/${id}`, patch),
  )

export type FlowType = 'requirement' | 'change' | 'defect'
const flow = (entityType: FlowType) => `/api/admin/workflows/${entityType}`

export const useAddState = (entityType: FlowType) =>
  useApiMutation((input: { key: string; name: string; position?: number; isFinal: boolean }) => api.post<Workflow>(`${flow(entityType)}/states`, input))
export const useUpdateState = (entityType: FlowType) =>
  useApiMutation(({ key, patch }: { key: string; patch: { name?: string; position?: number | null } }) =>
    api.patch<Workflow>(`${flow(entityType)}/states/${key}`, patch),
  )
export const useRemoveState = (entityType: FlowType) => useApiMutation((key: string) => api.delete(`${flow(entityType)}/states/${key}`))
export const useSetTransition = (entityType: FlowType) =>
  useApiMutation(({ from, to, ...body }: { from: string; to: string; permission: string; requiresReason: boolean }) =>
    api.put<Workflow>(`${flow(entityType)}/transitions/${from}/${to}`, body),
  )
export const useRemoveTransition = (entityType: FlowType) =>
  useApiMutation(({ from, to }: { from: string; to: string }) => api.delete(`${flow(entityType)}/transitions/${from}/${to}`))
