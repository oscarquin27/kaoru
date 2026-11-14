// Lo que comparten las pantallas de métricas: el encabezado, los filtros de sistema, severidad y
// período, y la frase que dice qué período se midió. Las métricas miden; no hay metas ni SLA.
import type { ReactNode } from 'react'
import { useSystems } from '../../shared/api/session.ts'
import type { MetricsPeriod } from '../../shared/api/types.ts'
import { formatDay } from '../../shared/lib/format.ts'
import { SEVERITY_LABELS, options } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { FilterBar, FilterInput, FilterSelect } from '../../shared/ui/Filters.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'

export function MetricsHead({ title, subtitle }: { title: string; subtitle: ReactNode }) {
  return (
    <>
      <Crumbs items={[{ label: 'Métricas' }, { label: title }]} />
      <PageHead title={title} subtitle={subtitle} />
    </>
  )
}

/** Filtros de una métrica. Sin `from`, la API mide los últimos 12 meses. */
export function MetricsFilters({ severity, period, children }: { severity?: boolean; period?: boolean; children?: ReactNode }) {
  const systems = useSystems().data?.items ?? []
  return (
    <FilterBar>
      <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
      {severity && <FilterSelect name="severity" label="Severidad" options={options(SEVERITY_LABELS)} />}
      {children}
      {period && (
        <>
          <FilterInput name="from" label="Desde" type="date" />
          <FilterInput name="to" label="Hasta" type="date" />
        </>
      )}
    </FilterBar>
  )
}

/** Los filtros de la URL que entiende la API, con la página si no es la primera. */
export function useMetricsQuery(names: string[]) {
  const { params, pick } = useUrlFilters()
  const page = Number(params.get('page')) || 1
  return { ...pick(names), page: page > 1 ? page : undefined }
}

/** «Del 1 dic 2025 al 30 nov 2026» o «Desde el 1 dic 2025», con lo que se mide. */
export function PeriodNote({ period, what }: { period: MetricsPeriod; what: string }) {
  const range = period.to ? `Del ${formatDay(period.from)} al ${formatDay(period.to)}` : `Desde el ${formatDay(period.from)}`
  return (
    <p className="muted">
      {range}: {what}.
    </p>
  )
}

/** Un número con su porcentaje: «3 de 12 (25 %)». */
export function share(part: number, total: number): string {
  return total === 0 ? '0' : `${part} de ${total} (${Math.round((part * 100) / total)} %)`
}
