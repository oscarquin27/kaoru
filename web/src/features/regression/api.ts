import { api } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Regression } from '../../shared/api/types.ts'

const base = (packageCode: string) => `/api/packages/${packageCode}/regression`

/** Selección de regresión del PKG. Un 404 dice que todavía no se propuso. */
export const useRegression = (packageCode: string) => useApiQuery<Regression>(base(packageCode))
export const useProposeRegression = (packageCode: string) => useApiMutation(() => api.post<Regression>(`${base(packageCode)}/proposal`, undefined))
export const useAdjustRegressionItem = (packageCode: string) =>
  useApiMutation(({ testCase, ...decision }: { testCase: string; included: boolean; reason?: string }) =>
    api.put<Regression>(`${base(packageCode)}/items/${testCase}`, decision),
  )
export const useConfirmRegression = (packageCode: string) => useApiMutation(() => api.post<Regression>(`${base(packageCode)}/confirmation`, undefined))
