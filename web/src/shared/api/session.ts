// Lo que varias pantallas necesitan de la API sin ser de un feature: sesión, sistemas y flujos.
import { useApiQuery } from './query.ts'
import type { Items, Me, System, Workflow } from './types.ts'

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

export function useWorkflow(entityType: 'requirement' | 'change' | 'defect') {
  return useApiQuery<Workflow>(`/api/workflows/${entityType}`)
}
