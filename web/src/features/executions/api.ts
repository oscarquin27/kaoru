import { api } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Exploration, ExplorationInput, Page, Run, RunInput } from '../../shared/api/types.ts'

export const useRequirementRuns = (requirementCode: string) => useApiQuery<Page<Run>>('/api/runs', { requirement: requirementCode })
export const useCreateRun = () => useApiMutation((input: RunInput) => api.post<Run>('/api/runs', input))

export const useRequirementExplorations = (requirementCode: string) =>
  useApiQuery<Page<Exploration>>('/api/explorations', { requirement: requirementCode })
export const useCreateExploration = () => useApiMutation((input: ExplorationInput) => api.post<Exploration>('/api/explorations', input))
