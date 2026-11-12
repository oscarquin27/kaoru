// El estado del servidor vive en TanStack Query; no se copia a un store global.
import { QueryClient, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type QueryParams } from './client.ts'
import { ApiError, isRetryable } from './errors.ts'

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Hasta dos repeticiones, solo si repetir puede cambiar el resultado.
        retry: (failures, error) => failures < 2 && isRetryable(error),
        staleTime: 15_000,
        refetchOnWindowFocus: false,
      },
      mutations: { retry: false },
    },
  })
}

/** Lectura de la API. La ruta y sus filtros son la llave de caché. */
export function useApiQuery<T>(path: string, query?: QueryParams, enabled = true) {
  return useQuery<T, ApiError>({
    queryKey: [path, query ?? {}],
    queryFn: () => api.get<T>(path, query),
    enabled,
  })
}

/** Escritura en la API. Al guardar, se vuelve a leer lo que haya en pantalla. */
export function useApiMutation<TInput, TOutput>(write: (input: TInput) => Promise<TOutput>) {
  const queryClient = useQueryClient()
  return useMutation<TOutput, ApiError, TInput>({
    mutationFn: write,
    onSuccess: () => queryClient.invalidateQueries(),
  })
}
