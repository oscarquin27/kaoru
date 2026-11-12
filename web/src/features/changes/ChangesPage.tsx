// Lista de cambios (CHG) con filtros en la URL y alta.
import { useState } from 'react'
import { useCan, useSystems, useWorkflow } from '../../shared/api/session.ts'
import { formatDate } from '../../shared/lib/format.ts'
import { ROUTE_LABELS } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useChanges } from './api.ts'
import { ChangeForm } from './ChangeForm.tsx'

export default function ChangesPage() {
  const can = useCan()
  const { params, pick } = useUrlFilters()
  const [creating, setCreating] = useState(false)
  const systems = useSystems().data?.items ?? []
  const states = useWorkflow('change').data?.states ?? []
  const page = Number(params.get('page')) || 1
  const changes = useChanges({ ...pick(['system', 'state', 'q']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Cambios' }]} />
      <PageHead
        title="Cambios"
        subtitle="Cada cambio (CHG) agrupa los requerimientos que salen juntos."
        actions={
          can('chg.create') && (
            <button className="btn btn--primary" type="button" onClick={() => setCreating(true)}>
              <Icon name="plus" />
              Nuevo CHG
            </button>
          )
        }
      />
      <FilterBar>
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
        <FilterSelect name="state" label="Estado" options={states.map((state) => ({ value: state.key, label: state.name }))} />
        <FilterInput name="q" label="Texto en el título" />
      </FilterBar>
      <Async query={changes}>
        {(data) => (
          <>
            {data.items.length === 0 ? (
              <Empty>No hay cambios con este filtro.</Empty>
            ) : (
              <div className="surface table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th className="col-fixed">CHG</th>
                      <th>Cambio</th>
                      <th>Sistema</th>
                      <th>Ruta</th>
                      <th>Estado</th>
                      <th>Ventana</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((change) => (
                      <tr key={change.code}>
                        <td className="col-fixed">
                          <Code code={change.code} />
                        </td>
                        <td>{change.title}</td>
                        <td>{change.system.name}</td>
                        <td>{ROUTE_LABELS[change.route]}</td>
                        <td>
                          <StateChip state={change.state} />
                        </td>
                        <td className="num">{formatDate(change.windowStart)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <Pager page={data.page} size={data.size} total={data.total} noun="cambios" />
          </>
        )}
      </Async>
      {creating && (
        <Dialog title="Nuevo cambio" onClose={() => setCreating(false)}>
          <ChangeForm onClose={() => setCreating(false)} />
        </Dialog>
      )}
    </>
  )
}
