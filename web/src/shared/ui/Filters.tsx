// Filtros de una lista. Su valor vive en la URL (compartible).
import type { ReactNode } from 'react'
import { useUrlFilters } from '../lib/urlFilters.ts'

export function FilterBar({ children, actions }: { children: ReactNode; actions?: ReactNode }) {
  return (
    <form className="surface pad-s filters" aria-label="Filtros" onSubmit={(event) => event.preventDefault()}>
      {children}
      {actions && <div className="filters__actions">{actions}</div>}
    </form>
  )
}

interface FilterProps {
  name: string
  label: string
}

export function FilterSelect({ name, label, options }: FilterProps & { options: { value: string; label: string }[] }) {
  const { params, setFilter } = useUrlFilters()
  return (
    <label className="field">
      {label}
      <select className="select" value={params.get(name) ?? ''} onChange={(event) => setFilter(name, event.target.value)}>
        <option value="">Todos</option>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  )
}

/** Campo de texto o fecha. El texto se aplica al salir del campo o con Enter. */
export function FilterInput({ name, label, type = 'text', placeholder }: FilterProps & { type?: 'text' | 'date'; placeholder?: string }) {
  const { params, setFilter } = useUrlFilters()
  const value = params.get(name) ?? ''
  return (
    <label className="field">
      {label}
      <input
        key={value}
        className="input"
        type={type}
        defaultValue={value}
        placeholder={placeholder}
        onChange={type === 'date' ? (event) => setFilter(name, event.target.value) : undefined}
        onBlur={(event) => event.target.value.trim() !== value && setFilter(name, event.target.value.trim())}
        onKeyDown={(event) => event.key === 'Enter' && setFilter(name, event.currentTarget.value.trim())}
      />
    </label>
  )
}

/** Pie de una lista paginada. */
export function Pager({ page, size, total, noun }: { page: number; size: number; total: number; noun: string }) {
  const { setFilter } = useUrlFilters()
  const first = total === 0 ? 0 : (page - 1) * size + 1
  const last = Math.min(page * size, total)
  return (
    <div className="pager">
      <span className="num">
        {first}–{last} de {total} {noun}
      </span>
      {total > size && (
        <span className="row">
          <button className="btn" type="button" disabled={page <= 1} onClick={() => setFilter('page', page > 2 ? String(page - 1) : '')}>
            Anterior
          </button>
          <button className="btn" type="button" disabled={last >= total} onClick={() => setFilter('page', String(page + 1))}>
            Siguiente
          </button>
        </span>
      )}
    </div>
  )
}
