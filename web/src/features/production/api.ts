import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type {
  Checklist,
  ChecklistInput,
  ChecklistTemplate,
  ChecklistTemplateInput,
  Diagnostic,
  DiagnosticInput,
  DiagnosticReviewInput,
  Items,
  Page,
  SmokeTest,
  SmokeTestInput,
} from '../../shared/api/types.ts'

export const useChecklistTemplates = (system?: string) => useApiQuery<Items<ChecklistTemplate>>('/api/checklist-templates', system ? { system } : undefined)
export const usePublishChecklistTemplate = () => useApiMutation((input: ChecklistTemplateInput) => api.post<ChecklistTemplate>('/api/checklist-templates', input))

export const useChecklists = (filters?: QueryParams) => useApiQuery<Page<Checklist>>('/api/checklists', filters)
export const useChecklist = (code: string) => useApiQuery<Checklist>(`/api/checklists/${code}`)
export const useStartChecklist = () => useApiMutation((input: ChecklistInput) => api.post<Checklist>('/api/checklists', input))
export const useSignChecklistStep = (code: string) =>
  useApiMutation(({ position, comment }: { position: number; comment: string }) => api.post<Checklist>(`/api/checklists/${code}/steps/${position}/signature`, { comment }))

export const useDiagnostics = (filters?: QueryParams) => useApiQuery<Page<Diagnostic>>('/api/diagnostics', filters)
export const useDiagnostic = (code: string) => useApiQuery<Diagnostic>(`/api/diagnostics/${code}`)
export const useSubmitDiagnostic = () => useApiMutation((input: DiagnosticInput) => api.post<Diagnostic>('/api/diagnostics', input))
export const useReviewDiagnostic = (code: string) => useApiMutation((input: DiagnosticReviewInput) => api.post<Diagnostic>(`/api/diagnostics/${code}/review`, input))

export const useSmokeTests = (filters?: QueryParams) => useApiQuery<Page<SmokeTest>>('/api/smoke-tests', filters)
export const useSmokeTest = (code: string) => useApiQuery<SmokeTest>(`/api/smoke-tests/${code}`)
export const useRegisterSmokeTest = () => useApiMutation((input: SmokeTestInput) => api.post<SmokeTest>('/api/smoke-tests', input))
