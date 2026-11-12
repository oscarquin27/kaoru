import { api, apiUrl, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Page, Report } from '../../shared/api/types.ts'

export const useReports = (filters?: QueryParams) => useApiQuery<Page<Report>>('/api/reports', filters)
export const useReport = (code: string) => useApiQuery<Report>(`/api/reports/${code}`)
export const useIssueReport = () => useApiMutation((packageCode: string) => api.post<Report>('/api/reports', { package: packageCode }))

/** Descarga del informe: el Markdown tal como se emitió o el PDF generado de él. */
export const reportExportUrl = (code: string, format: 'markdown' | 'pdf') => apiUrl(`/api/reports/${code}/export`, { format })
