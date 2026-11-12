// Lista de BUILD: los artefactos sobre los que se ejecuta, con su checksum y su manifiesto.
import { useState } from 'react'
import { useCan, useSystems } from '../../shared/api/session.ts'
import { formatDate, shortChecksum } from '../../shared/lib/format.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useBuilds } from './api.ts'
import { BuildForm } from './BuildForm.tsx'

export default function BuildsPage() {
  const can = useCan()
  const { params, pick } = useUrlFilters()
  const [registering, setRegistering] = useState(false)
  const systems = useSystems().data?.items ?? []
  const page = Number(params.get('page')) || 1
  const builds = useBuilds({ ...pick(['system', 'requirement', 'q']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Paquetes', to: '/paquetes' }, { label: 'BUILD' }]} />
      <PageHead
        title="BUILD"
        subtitle="Cada BUILD es un artefacto con su SHA-256 y su manifiesto. No se edita ni se borra."
        actions={
          can('build.register') && (
            <button className="btn btn--primary" type="button" onClick={() => setRegistering(true)}>
              <Icon name="plus" />
              Registrar BUILD
            </button>
          )
        }
      />
      <FilterBar>
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
        <FilterInput name="requirement" label="REQ incluido" placeholder="REQ-2026-0001" />
        <FilterInput name="q" label="Texto en la versión" />
      </FilterBar>
      <Async query={builds}>
        {(data) => (
          <>
            {data.items.length === 0 ? (
              <Empty>No hay BUILD con este filtro.</Empty>
            ) : (
              <div className="surface table-wrap">
                <table className="table" aria-label="BUILD">
                  <thead>
                    <tr>
                      <th className="col-fixed">BUILD</th>
                      <th>Versión</th>
                      <th>Sistema</th>
                      <th>SHA-256</th>
                      <th>REQ incluidos</th>
                      <th>Registró</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((build) => (
                      <tr key={build.code}>
                        <td className="col-fixed">
                          <Code code={build.code} />
                        </td>
                        <td className="mono">{build.version}</td>
                        <td>{build.system.name}</td>
                        <td>
                          <span className="mono" title={build.checksum}>
                            {shortChecksum(build.checksum)}
                          </span>
                        </td>
                        <td>
                          <span className="cell-line">
                            {build.requirements.length === 0 ? <span className="muted">Ninguno</span> : build.requirements.map((requirement) => <Code key={requirement.code} code={requirement.code} />)}
                          </span>
                        </td>
                        <td>
                          {build.builtBy.displayName} · {formatDate(build.createdAt)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <Pager page={data.page} size={data.size} total={data.total} noun="BUILD" />
          </>
        )}
      </Async>
      {registering && (
        <Dialog title="Registrar BUILD" onClose={() => setRegistering(false)}>
          <BuildForm onClose={() => setRegistering(false)} />
        </Dialog>
      )}
    </>
  )
}
