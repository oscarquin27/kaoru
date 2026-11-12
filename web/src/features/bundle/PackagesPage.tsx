// Lista de paquetes (PKG): lo que sale junto a producción, con sus BUILD y sus REQ.
import { useState } from 'react'
import { useCan, useSystems, useWorkflow } from '../../shared/api/session.ts'
import { formatDate } from '../../shared/lib/format.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { usePackages } from './api.ts'
import { PackageForm } from './PackageForm.tsx'

export default function PackagesPage() {
  const can = useCan()
  const { params, pick } = useUrlFilters()
  const [creating, setCreating] = useState(false)
  const systems = useSystems().data?.items ?? []
  const states = useWorkflow('package').data?.states ?? []
  const page = Number(params.get('page')) || 1
  const packages = usePackages({ ...pick(['system', 'state', 'q']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Paquetes' }]} />
      <PageHead
        title="Paquetes"
        subtitle="Cada paquete (PKG) reúne los BUILD y los REQ que salen juntos, con su regresión."
        actions={
          can('pkg.build') && (
            <button className="btn btn--primary" type="button" onClick={() => setCreating(true)}>
              <Icon name="plus" />
              Nuevo PKG
            </button>
          )
        }
      />
      <FilterBar>
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
        <FilterSelect name="state" label="Estado" options={states.map((state) => ({ value: state.key, label: state.name }))} />
        <FilterInput name="q" label="Texto en el título" />
      </FilterBar>
      <Async query={packages}>
        {(data) => (
          <>
            {data.items.length === 0 ? (
              <Empty>No hay paquetes con este filtro.</Empty>
            ) : (
              <div className="surface table-wrap">
                <table className="table" aria-label="Paquetes">
                  <thead>
                    <tr>
                      <th className="col-fixed">PKG</th>
                      <th>Paquete</th>
                      <th>Sistema</th>
                      <th>Estado</th>
                      <th className="num">BUILD</th>
                      <th className="num">REQ</th>
                      <th>Creado</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((item) => (
                      <tr key={item.code}>
                        <td className="col-fixed">
                          <Code code={item.code} />
                        </td>
                        <td>{item.title}</td>
                        <td>{item.system.name}</td>
                        <td>
                          <StateChip state={item.state} />
                        </td>
                        <td className="num">{item.builds.length}</td>
                        <td className="num">{item.requirements.length}</td>
                        <td>
                          {item.createdBy.displayName} · {formatDate(item.createdAt)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <Pager page={data.page} size={data.size} total={data.total} noun="paquetes" />
          </>
        )}
      </Async>
      {creating && (
        <Dialog title="Nuevo paquete" onClose={() => setCreating(false)}>
          <PackageForm onClose={() => setCreating(false)} />
        </Dialog>
      )}
    </>
  )
}
