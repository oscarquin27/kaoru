// Avisos de error, esqueletos de carga y el envoltorio que los aplica a una lectura.
import type { UseQueryResult } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import type { ApiError } from '../api/errors.ts'
import { Icon } from './Icon.tsx'

export function ErrorNotice({ error, onRetry }: { error: ApiError; onRetry?: () => void }) {
  return (
    <div className="notice notice--error" role="alert">
      <Icon name="triangle" />
      <div className="notice__main">
        {error.message}
        {error.fieldErrors.length > 0 && (
          <ul>
            {error.fieldErrors.map((item) => (
              <li key={item.field + item.message}>{item.message}</li>
            ))}
          </ul>
        )}
        {error.traceId && <div className="muted">Código de seguimiento: <span className="mono">{error.traceId}</span></div>}
      </div>
      {onRetry && (
        <button className="btn" type="button" onClick={onRetry}>
          Reintentar
        </button>
      )}
    </div>
  )
}

/** Esqueleto con la geometría de una pantalla: título y bloque de contenido. */
export function PageSkeleton() {
  return (
    <div className="stack" aria-busy="true" aria-label="Cargando">
      <span className="skeleton skeleton--title" />
      <span className="skeleton skeleton--row" />
      <span className="skeleton skeleton--block" />
    </div>
  )
}

export function BlockSkeleton() {
  return <span className="skeleton skeleton--block" aria-busy="true" aria-label="Cargando" />
}

interface AsyncProps<T> {
  query: UseQueryResult<T, ApiError>
  skeleton?: ReactNode
  children: (data: T) => ReactNode
}

/** Pinta una lectura: esqueleto mientras llega (solo la primera vez), aviso si falla, datos si hay. */
export function Async<T>({ query, skeleton, children }: AsyncProps<T>) {
  if (query.data !== undefined) return <>{children(query.data)}</>
  if (query.error) return <ErrorNotice error={query.error} onRetry={() => void query.refetch()} />
  return <>{skeleton ?? <BlockSkeleton />}</>
}

export function Empty({ children }: { children: ReactNode }) {
  return <p className="muted pad-s">{children}</p>
}
