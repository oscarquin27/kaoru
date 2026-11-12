// El feature se llama `bundle`, como en el backend (`package` es palabra reservada de Go).
import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Package, PackageInput, PackagePatch, Page, Requirement, TransitionInput } from '../../shared/api/types.ts'

export const usePackages = (filters?: QueryParams) => useApiQuery<Page<Package>>('/api/packages', filters)
export const usePackage = (code: string) => useApiQuery<Package>(`/api/packages/${code}`)
export const useCreatePackage = () => useApiMutation((input: PackageInput) => api.post<Package>('/api/packages', input))
export const useUpdatePackage = (code: string) => useApiMutation((patch: PackagePatch) => api.patch<Package>(`/api/packages/${code}`, patch))
export const useTransitionPackage = (code: string) => useApiMutation((input: TransitionInput) => api.post<Package>(`/api/packages/${code}/transitions`, input))

export const useAddPackageBuild = (code: string) => useApiMutation((build: string) => api.put<Package>(`/api/packages/${code}/builds/${build}`, undefined))
export const useRemovePackageBuild = (code: string) => useApiMutation((build: string) => api.delete(`/api/packages/${code}/builds/${build}`))
export const useAddPackageRequirement = (code: string) =>
  useApiMutation((requirement: string) => api.put<Package>(`/api/packages/${code}/requirements/${requirement}`, undefined))
export const useRemovePackageRequirement = (code: string) => useApiMutation((requirement: string) => api.delete(`/api/packages/${code}/requirements/${requirement}`))

/** Los REQ de un sistema en un estado, para elegir los que entran al paquete. */
export const useRequirementsInState = (system: string, state: string) => useApiQuery<Page<Requirement>>('/api/requirements', { system, state, size: 200 })
