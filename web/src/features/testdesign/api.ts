import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { FunctionalCase, FunctionalCaseInput, Items, Page, TestCase, TestCaseInput } from '../../shared/api/types.ts'

export const useFunctionalCases = (requirementCode: string) =>
  useApiQuery<Items<FunctionalCase>>(`/api/requirements/${requirementCode}/functional-cases`)
export const useCreateFunctionalCase = (requirementCode: string) =>
  useApiMutation((input: FunctionalCaseInput) => api.post<FunctionalCase>(`/api/requirements/${requirementCode}/functional-cases`, input))

export const useTestCases = (filters?: QueryParams) => useApiQuery<Page<TestCase>>('/api/test-cases', filters)
export const useCreateTestCase = () => useApiMutation((input: TestCaseInput) => api.post<TestCase>('/api/test-cases', input))
