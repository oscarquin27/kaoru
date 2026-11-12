import { api } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { BlockingRule, Role, RolePermissions } from '../../shared/api/types.ts'

export const useRolePermissions = () => useApiQuery<RolePermissions>('/api/admin/role-permissions')
export const useSetRolePermissions = () =>
  useApiMutation(({ role, permissions }: { role: Role; permissions: string[] }) =>
    api.put<RolePermissions>(`/api/admin/role-permissions/${role}`, { permissions }),
  )
export const useUpdateBlockingRule = () =>
  useApiMutation(({ id, patch }: { id: string; patch: { enabled?: boolean; blocks?: boolean } }) =>
    api.patch<BlockingRule>(`/api/admin/blocking-rules/${id}`, patch),
  )
