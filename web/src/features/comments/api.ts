import { api } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { Comment, EntityType, Items } from '../../shared/api/types.ts'

export const useComments = (entityType: EntityType, entityCode: string, enabled = true) => useApiQuery<Items<Comment>>('/api/comments', { entityType, entityCode }, enabled)
export const useCreateComment = (entityType: EntityType, entityCode: string) =>
  useApiMutation((body: string) => api.post<Comment>('/api/comments', { entityType, entityCode, body }))
