// Catálogos que varias pantallas ofrecen en un selector: componentes, BUILD, ambientes, paquetes y CHG.
// Cada feature es dueño de sus pantallas; estas lecturas son lo común (reglas §A.16.2).
import { api } from './client.ts'
import { useApiMutation, useApiQuery } from './query.ts'
import type { Build, Change, Component, ComponentRef, Environment, Items, Package, Page } from './types.ts'

// Un selector trae el catálogo completo del sistema: el máximo que admite la API por página.
const ALL = 200

/** Componentes activos; con `system`, solo los de ese sistema. */
export const useComponentOptions = (system?: string, enabled = true) => useApiQuery<Page<Component>>('/api/components', { system, size: ALL }, enabled)
/** BUILD de un sistema, del más reciente al más antiguo. */
export const useBuildOptions = (system: string, enabled = true) => useApiQuery<Page<Build>>('/api/builds', { system, size: ALL }, enabled && system !== '')
/** Ambientes activos de un sistema. */
export const useEnvironmentOptions = (system: string, enabled = true) =>
  useApiQuery<Page<Environment>>('/api/environments', { system, size: ALL }, enabled && system !== '')
/** Paquetes; con `system`, solo los de ese sistema. */
export const usePackageOptions = (system?: string, enabled = true) => useApiQuery<Page<Package>>('/api/packages', { system, size: ALL }, enabled)

/** Los CHG, para filtrar por cambio. */
export const useChangeOptions = () => useApiQuery<Page<Change>>('/api/changes')

/** Componentes que declara un REQ o un TC. `path` es `/api/requirements/{code}/components` o el del TC. */
export const useImpact = (path: string) => useApiQuery<Items<ComponentRef>>(path)
export const useSetImpact = (path: string) => useApiMutation((components: string[]) => api.put<Items<ComponentRef>>(path, { components }))

export const requirementImpactPath = (code: string) => `/api/requirements/${code}/components`
export const testCaseImpactPath = (code: string) => `/api/test-cases/${code}/components`
