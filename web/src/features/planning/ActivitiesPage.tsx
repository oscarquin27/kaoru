// Lista de actividades con filtros en la URL: por operador, clase, estado y CHG. Desde aquí se da de
// alta una actividad; reajustarla se hace en su ficha.
import { useState } from 'react'
import { useCan, useUserOptions } from '../../shared/api/session.ts'
import { ACTIVITY_KIND_LABELS, ACTIVITY_STATUS_LABELS, options } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { ActivityStatusChip, KindChip } from './ActivityChips.tsx'
import { ActivityForm } from './ActivityForm.tsx'
import { useActivities } from './api.ts'
import { lengthText, spanText } from './days.ts'

const FILTERS = ['operator', 'kind', 'status', 'change']

export default function ActivitiesPage() {
  const can = useCan()
  const { params, pick } = useUrlFilters()
  const operatorOptions = useUserOptions()
  const [creating, setCreating] = useState(false)
  const page = Number(params.get('page')) || 1
  const filters = pick(FILTERS)
  const activities = useActivities({ ...filters, change: filters.change?.toUpperCase(), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Planificación', to: '/planificacion' }, { label: 'Actividades' }]} />
      <PageHead
        title="Actividades"
        subtitle="Certificaciones, regresiones, pases a producción y lo demás que ocupa a los operadores, con sus fechas."
        actions={
          can('activity.manage') && (
            <button className="btn btn--primary" type="button" onClick={() => setCreating(true)}>
              <Icon name="plus" />
              Nueva actividad
            </button>
          )
        }
      />
      <FilterBar>
        <FilterSelect name="operator" label="Operador" options={operatorOptions} />
        <FilterSelect name="kind" label="Clase" options={options(ACTIVITY_KIND_LABELS)} />
        <FilterSelect name="status" label="Estado" options={options(ACTIVITY_STATUS_LABELS)} />
        <FilterInput name="change" label="CHG" placeholder="CHG-2026-0001" />
      </FilterBar>
      <Async query={activities}>
        {(data) => (
          <>
            {data.items.length === 0 ? (
              <Empty>No hay actividades con este filtro.</Empty>
            ) : (
              <div className="surface table-wrap">
                <table className="table" aria-label="Actividades">
                  <thead>
                    <tr>
                      <th className="col-fixed">ACT</th>
                      <th>Actividad</th>
                      <th>CHG</th>
                      <th>Fechas</th>
                      <th>Operadores</th>
                      <th>Estado</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((activity) => (
                      <tr key={activity.code}>
                        <td className="col-fixed">
                          <Code code={activity.code} />
                        </td>
                        <td>
                          <div className="cell-stack">
                            {activity.title}
                            <KindChip kind={activity.kind} />
                          </div>
                        </td>
                        <td>{activity.change ? <Code code={activity.change.code} /> : <span className="muted">—</span>}</td>
                        <td className="num">
                          {spanText(activity.startDate, activity.endDate)}
                          <br />
                          <span className="soft">{lengthText(activity.startDate, activity.endDate)}</span>
                        </td>
                        <td>
                          {activity.operators.length > 0 ? (
                            activity.operators.map((operator) => operator.displayName).join(', ')
                          ) : (
                            <span className="cell-line soft">
                              <Icon name="triangle" />
                              Sin operador
                            </span>
                          )}
                        </td>
                        <td>
                          <ActivityStatusChip status={activity.status} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <Pager page={data.page} size={data.size} total={data.total} noun="actividades" />
          </>
        )}
      </Async>
      {creating && (
        <Dialog title="Nueva actividad" onClose={() => setCreating(false)}>
          <ActivityForm onClose={() => setCreating(false)} />
        </Dialog>
      )}
    </>
  )
}
