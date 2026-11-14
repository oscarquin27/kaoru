import { api, type QueryParams } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Activity, ActivityInput, ActivityPatch, ActivitySummary, Calendar, Milestone, MilestoneInput, MilestonePatch, Page, Schedule } from '../../shared/api/types.ts'

export const useCalendar = () => useApiQuery<Calendar>('/api/calendar')
export const useMilestones = (filters?: QueryParams) => useApiQuery<Page<Milestone>>('/api/milestones', filters)
export const useCreateMilestone = () => useApiMutation((input: MilestoneInput) => api.post<Milestone>('/api/milestones', input))
export const useUpdateMilestone = (id: string) => useApiMutation((patch: MilestonePatch) => api.patch<Milestone>(`/api/milestones/${id}`, patch))

export const useSchedule = (filters?: QueryParams, enabled = true) => useApiQuery<Schedule>('/api/planning/schedule', filters, enabled)
export const useActivities = (filters?: QueryParams, enabled = true) => useApiQuery<Page<ActivitySummary>>('/api/activities', filters, enabled)
export const useActivity = (code: string) => useApiQuery<Activity>(`/api/activities/${code}`)
export const useCreateActivity = () => useApiMutation((input: ActivityInput) => api.post<Activity>('/api/activities', input))
export const useUpdateActivity = (code: string) => useApiMutation((patch: ActivityPatch) => api.patch<Activity>(`/api/activities/${code}`, patch))
