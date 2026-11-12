// Lista de ambientes (ENV) con lo que tienen instalado: lo que dejó su último despliegue exitoso.
import { useState } from 'react'
import { useCan, useSystems } from '../../shared/api/session.ts'
import { formatDate } from '../../shared/lib/format.ts'
import { ENVIRONMENT_KIND_LABELS, OS_LABELS, options } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Chip, RetiredChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterCheck, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useEnvironments } from './api.ts'
import { Deployed } from './DeploymentsTable.tsx'
import { EnvironmentForm } from './EnvironmentForm.tsx'

export default function EnvironmentsPage() {
  const can = useCan()
  const { params, pick } = useUrlFilters()
  const [creating, setCreating] = useState(false)
  const systems = useSystems().data?.items ?? []
  const page = Number(params.get('page')) || 1
  const environments = useEnvironments({ ...pick(['system', 'kind', 'includeRetired']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Ambientes' }]} />
      <PageHead
        title="Ambientes"
        subtitle="Dónde se prueba y dónde se despliega. Lo instalado se lee del último despliegue exitoso."
        actions={
          can('env.manage') && (
            <button className="btn btn--primary" type="button" onClick={() => setCreating(true)}>
              <Icon name="plus" />
              Nuevo ENV
            </button>
          )
        }
      />
      <FilterBar>
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
        <FilterSelect name="kind" label="Tipo" options={options(ENVIRONMENT_KIND_LABELS)} />
        <FilterCheck name="includeRetired" label="Incluir retirados" />
      </FilterBar>
      <Async query={environments}>
        {(data) => (
          <>
            {data.items.length === 0 ? (
              <Empty>No hay ambientes con este filtro.</Empty>
            ) : (
              <div className="surface table-wrap">
                <table className="table" aria-label="Ambientes">
                  <thead>
                    <tr>
                      <th className="col-fixed">ENV</th>
                      <th>Ambiente</th>
                      <th>Sistema</th>
                      <th>Tipo</th>
                      <th>SO</th>
                      <th>Instalado</th>
                      <th>Vigencia</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((environment) => (
                      <tr key={environment.code}>
                        <td className="col-fixed">
                          <Code code={environment.code} />
                        </td>
                        <td>{environment.name}</td>
                        <td>{environment.system.name}</td>
                        <td>
                          {environment.kind === 'production' ? (
                            <Chip tone="aviso" icon="flag">
                              {ENVIRONMENT_KIND_LABELS[environment.kind]}
                            </Chip>
                          ) : (
                            ENVIRONMENT_KIND_LABELS[environment.kind]
                          )}
                        </td>
                        <td>{OS_LABELS[environment.os]}</td>
                        <td>
                          {environment.installed ? (
                            <div className="cell-stack">
                              <Deployed build={environment.installed.build} bundle={environment.installed.package} />
                              <span className="muted">desde el {formatDate(environment.installed.deployedAt)}</span>
                            </div>
                          ) : (
                            <span className="muted">Sin despliegues exitosos</span>
                          )}
                        </td>
                        <td>
                          <RetiredChip retired={environment.retired} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <Pager page={data.page} size={data.size} total={data.total} noun="ambientes" />
          </>
        )}
      </Async>
      {creating && (
        <Dialog title="Nuevo ambiente" onClose={() => setCreating(false)}>
          <EnvironmentForm onClose={() => setCreating(false)} />
        </Dialog>
      )}
    </>
  )
}
