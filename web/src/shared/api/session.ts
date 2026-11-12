// Lo que varias pantallas necesitan de la API sin ser de un feature: sesión, sistemas y flujos.
import { useApiQuery } from './query.ts'
import type { Items, Me, System, User, Workflow, WorkflowEntityType } from './types.ts'

export function useMe() {
  return useApiQuery<Me>('/api/me')
}

/** Dice si el rol puede usar un botón. Solo oculta: la comprobación real está en el backend. */
export function useCan(): (permission: string) => boolean {
  const permissions = useMe().data?.permissions
  return (permission) => permissions?.includes(permission) ?? false
}

export function useSystems() {
  return useApiQuery<Items<System>>('/api/systems')
}

export function useWorkflow(entityType: WorkflowEntityType) {
  return useApiQuery<Workflow>(`/api/workflows/${entityType}`)
}

/** Quienes ya entraron a Kaoru, para elegir responsables. El directorio vive en Keycloak. */
export function useUsers() {
  return useApiQuery<Items<User>>('/api/users')
}

/** Los usuarios como opciones de un filtro o de un campo; el de la sesión va marcado. */
export function useUserOptions(): { value: string; label: string }[] {
  const me = useMe().data
  const users = useUsers().data?.items ?? []
  return users.map((user) => ({ value: user.id, label: user.id === me?.id ? `Yo (${user.displayName})` : user.displayName }))
}
