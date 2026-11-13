// Lista de pruebas de humo posteriores al despliegue (SMOKE). Se registran desde la ficha del PKG.
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Async } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, Pager } from '../../shared/ui/Filters.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { useSmokeTests } from './api.ts'
import { SmokeTestsTable } from './VerificationTables.tsx'

export default function SmokeTestsPage() {
  const { params, pick } = useUrlFilters()
  const page = Number(params.get('page')) || 1
  const smokeTests = useSmokeTests({ ...pick(['package']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Producción', to: '/produccion' }, { label: 'Pruebas de humo' }]} />
      <PageHead title="Pruebas de humo" subtitle="El resultado de la prueba de humo después de cada despliegue en producción. En ruta ligera, el PKG pasa a verificado con la última aprobada." />
      <FilterBar>
        <FilterInput name="package" label="PKG" placeholder="PKG-2026-0001" />
      </FilterBar>
      <Async query={smokeTests}>
        {(data) => (
          <>
            <SmokeTestsTable items={data.items} showPackage empty="No hay pruebas de humo con este filtro." />
            <Pager page={data.page} size={data.size} total={data.total} noun="pruebas de humo" />
          </>
        )}
      </Async>
    </>
  )
}
