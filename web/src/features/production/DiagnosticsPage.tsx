// Lista de diagnósticos posteriores al despliegue (DIAG). Se entregan desde la ficha del PKG.
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Async } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, Pager } from '../../shared/ui/Filters.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { useDiagnostics } from './api.ts'
import { DiagnosticsTable } from './VerificationTables.tsx'

export default function DiagnosticsPage() {
  const { params, pick } = useUrlFilters()
  const page = Number(params.get('page')) || 1
  const diagnostics = useDiagnostics({ ...pick(['package']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Producción', to: '/produccion' }, { label: 'Diagnósticos' }]} />
      <PageHead title="Diagnósticos" subtitle="Lo que dev u ops revisaron después de cada despliegue en producción, y si QA lo dio por conforme. En ruta alta, el PKG pasa a verificado con el último DIAG conforme." />
      <FilterBar>
        <FilterInput name="package" label="PKG" placeholder="PKG-2026-0001" />
      </FilterBar>
      <Async query={diagnostics}>
        {(data) => (
          <>
            <DiagnosticsTable items={data.items} showPackage empty="No hay diagnósticos con este filtro." />
            <Pager page={data.page} size={data.size} total={data.total} noun="diagnósticos" />
          </>
        )}
      </Async>
    </>
  )
}
