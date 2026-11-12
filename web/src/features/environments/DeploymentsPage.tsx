// Lista general de despliegues (DEP). Se registran desde la ficha de cada ambiente.
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Async } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, Pager } from '../../shared/ui/Filters.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { useDeployments } from './api.ts'
import { DeploymentsTable } from './DeploymentsTable.tsx'

export default function DeploymentsPage() {
  const { params, pick } = useUrlFilters()
  const page = Number(params.get('page')) || 1
  const deployments = useDeployments({ ...pick(['environment', 'build', 'package']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Ambientes', to: '/ambientes' }, { label: 'Despliegues' }]} />
      <PageHead title="Despliegues" subtitle="Qué se desplegó, dónde y con qué resultado. Un despliegue se registra desde la ficha de su ambiente y no se edita." />
      <FilterBar>
        <FilterInput name="environment" label="Ambiente" placeholder="ENV-2026-0001" />
        <FilterInput name="build" label="BUILD" placeholder="BUILD-2026-0001" />
        <FilterInput name="package" label="PKG" placeholder="PKG-2026-0001" />
      </FilterBar>
      <Async query={deployments}>
        {(data) => (
          <>
            <DeploymentsTable items={data.items} showEnvironment empty="No hay despliegues con este filtro." />
            <Pager page={data.page} size={data.size} total={data.total} noun="despliegues" />
          </>
        )}
      </Async>
    </>
  )
}
