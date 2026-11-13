import { api, apiUrl, requestRaw } from '../../shared/api/client.ts'
import { useApiMutation, useApiQuery } from '../../shared/api/query.ts'
import type { ApiError } from '../../shared/api/errors.ts'
import type { Document, Items } from '../../shared/api/types.ts'
import { useQuery } from '@tanstack/react-query'

/** Entidades a las que se adjuntan documentos desde su ficha. */
export type DocumentOwner = 'requirement' | 'package' | 'checklist' | 'diagnostic' | 'smoke_test'
export const useEntityDocuments = (entityType: DocumentOwner, entityCode: string) => useApiQuery<Items<Document>>('/api/documents', { entityType, entityCode })
export const useRequirementDocuments = (requirementCode: string) => useEntityDocuments('requirement', requirementCode)
export const useDocument = (code: string) => useApiQuery<Document>(`/api/documents/${code}`)

export const useCreateDocument = () => useApiMutation((form: FormData) => api.upload<Document>('/api/documents', form))
export const useCreateDocumentVersion = (code: string) =>
  useApiMutation((form: FormData) => api.upload<Document>(`/api/documents/${code}/versions`, form))

export function contentUrl(code: string, version: number, download = false): string {
  return apiUrl(`/api/documents/${code}/versions/${version}/content`, { download: download || undefined })
}

/** PDF convertido de una versión de Office; solo existe si la versión dice `previewable`. */
export function previewUrl(code: string, version: number): string {
  return apiUrl(`/api/documents/${code}/versions/${version}/preview`)
}

async function sha256(bytes: ArrayBuffer): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join('')
}

/** Contenido de texto de una versión, con su SHA-256 recalculado en el navegador. */
export function useTextContent(code: string, version: number, enabled: boolean) {
  return useQuery<{ text: string; checksum: string }, ApiError>({
    queryKey: [contentUrl(code, version), 'text'],
    enabled,
    queryFn: async () => {
      const bytes = await (await requestRaw('GET', contentUrl(code, version))).arrayBuffer()
      return { text: new TextDecoder().decode(bytes), checksum: await sha256(bytes) }
    },
  })
}
