// Lista de checklists de producción (CHK). Se abren desde la ficha del PKG.
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Async } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, Pager } from '../../shared/ui/Filters.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { useChecklists } from './api.ts'
import { ChecklistsTable } from './ChecklistsTable.tsx'

export default function ChecklistsPage() {
  const { params, pick } = useUrlFilters()
  const page = Number(params.get('page')) || 1
  const checklists = useChecklists({ ...pick(['package']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Producción' }, { label: 'Checklists' }]} />
      <PageHead
        title="Checklists de producción"
        subtitle="Los pasos del pase a producción, firmados uno a uno. Un checklist se abre desde la ficha del PKG y sus firmas no se quitan."
      />
      <FilterBar>
        <FilterInput name="package" label="PKG" placeholder="PKG-2026-0001" />
      </FilterBar>
      <Async query={checklists}>
        {(data) => (
          <>
            <ChecklistsTable items={data.items} showPackage empty="No hay checklists con este filtro." />
            <Pager page={data.page} size={data.size} total={data.total} noun="checklists" />
          </>
        )}
      </Async>
    </>
  )
}
