import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Calendar, Milestone, MilestoneInput, MilestonePatch, Page } from '../../shared/api/types.ts'

export const useCalendar = () => useApiQuery<Calendar>('/api/calendar')
export const useMilestones = (filters?: QueryParams) => useApiQuery<Page<Milestone>>('/api/milestones', filters)
export const useCreateMilestone = () => useApiMutation((input: MilestoneInput) => api.post<Milestone>('/api/milestones', input))
export const useUpdateMilestone = (id: string) => useApiMutation((patch: MilestonePatch) => api.patch<Milestone>(`/api/milestones/${id}`, patch))
