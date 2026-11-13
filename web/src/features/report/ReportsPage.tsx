// Lista de informes de conformidad (RPT). Se emiten desde la ficha del PKG.
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Async } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, Pager } from '../../shared/ui/Filters.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { useReports } from './api.ts'
import { ReportsTable } from './ReportsTable.tsx'

export default function ReportsPage() {
  const { params, pick } = useUrlFilters()
  const page = Number(params.get('page')) || 1
  const reports = useReports({ ...pick(['package']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Informes' }]} />
      <PageHead
        title="Informes de conformidad"
        subtitle="La evidencia de un PKG congelada en un documento: contenido, trazabilidad, ejecuciones, regresión, defectos, checklist y verificación. Un informe no cambia; si algo cambia, se emite otra versión."
      />
      <FilterBar>
        <FilterInput name="package" label="PKG" placeholder="PKG-2026-0001" />
      </FilterBar>
      <Async query={reports}>
        {(data) => (
          <>
            <ReportsTable items={data.items} showPackage empty="No hay informes con este filtro." />
            <Pager page={data.page} size={data.size} total={data.total} noun="informes" />
          </>
        )}
      </Async>
    </>
  )
}
