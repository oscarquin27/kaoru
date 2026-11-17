// Catálogo de componentes (COMP): lo que un REQ toca y un TC ejercita. No se borran: se retiran.
import { useState } from 'react'
import { useCan, useSystems } from '../../shared/api/session.ts'
import type { Component } from '../../shared/api/types.ts'
import { useComponentOptions } from '../../shared/api/catalog.ts'
import { componentKindLabel, componentKindOptions } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { RetiredChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Async, Empty, ErrorNotice } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterCheck, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useComponents, useUpdateComponent } from './api.ts'
import { ComponentForm } from './ComponentForm.tsx'

export default function ComponentsPage() {
  const canManage = useCan()('comp.manage')
  const { params, pick } = useUrlFilters()
  const systems = useSystems().data?.items ?? []
  const page = Number(params.get('page')) || 1
  const components = useComponents({ ...pick(['system', 'kind', 'q', 'includeRetired']), page: page > 1 ? page : undefined })
  const update = useUpdateComponent()
  // Las clases propias que ya usa el catálogo se ofrecen en el filtro junto a las de entrada.
  const kindsInUse = (useComponentOptions().data?.items ?? []).map((component) => component.kind)
  const [form, setForm] = useState<{ component?: Component } | null>(null)

  return (
    <>
      <Crumbs items={[{ label: 'Pruebas', to: '/pruebas' }, { label: 'Componentes' }]} />
      <PageHead
        title="Componentes"
        subtitle="Cada REQ declara los componentes que toca y cada TC los que ejercita. De ese cruce sale la regresión de un paquete."
        actions={
          canManage && (
            <button className="btn btn--primary" type="button" onClick={() => setForm({})}>
              <Icon name="plus" />
              Nuevo COMP
            </button>
          )
        }
      />
      <FilterBar>
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
        <FilterSelect name="kind" label="Clase" options={componentKindOptions(kindsInUse)} />
        <FilterInput name="q" label="Texto en el nombre" />
        <FilterCheck name="includeRetired" label="Incluir retirados" />
      </FilterBar>
      {update.error && <ErrorNotice error={update.error} />}
      <Async query={components}>
        {(data) => (
          <>
            {data.items.length === 0 ? (
              <Empty>No hay componentes con este filtro.</Empty>
            ) : (
              <div className="surface table-wrap">
                <table className="table" aria-label="Componentes">
                  <thead>
                    <tr>
                      <th className="col-fixed">COMP</th>
                      <th>Componente</th>
                      <th>Clase</th>
                      <th>Sistema</th>
                      <th>Vigencia</th>
                      {canManage && (
                        <th className="col-actions">
                          <span className="sr-only">Acciones</span>
                        </th>
                      )}
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((component) => (
                      <tr key={component.code}>
                        <td className="col-fixed">
                          <Code code={component.code} />
                        </td>
                        <td>
                          <div className="cell-stack">
                            <span>{component.name}</span>
                            {component.description && <span className="muted">{component.description}</span>}
                          </div>
                        </td>
                        <td>{componentKindLabel(component.kind)}</td>
                        <td>{component.system.name}</td>
                        <td>
                          <RetiredChip retired={component.retired} />
                        </td>
                        {canManage && (
                          <td className="col-actions">
                            <span className="cell-line">
                              <button className="btn" type="button" aria-label={`Editar ${component.name}`} onClick={() => setForm({ component })}>
                                Editar
                              </button>
                              <button
                                className="btn"
                                type="button"
                                aria-label={`${component.retired ? 'Reactivar' : 'Retirar'} ${component.name}`}
                                title={component.retired ? undefined : 'Retirado, ya no se ofrece para declarar impacto'}
                                disabled={update.isPending}
                                onClick={() => update.mutate({ code: component.code, patch: { retired: !component.retired } })}
                              >
                                {component.retired ? 'Reactivar' : 'Retirar'}
                              </button>
                            </span>
                          </td>
                        )}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <Pager page={data.page} size={data.size} total={data.total} noun="componentes" />
          </>
        )}
      </Async>
      {form && (
        <Dialog title={form.component ? `Editar ${form.component.code}` : 'Nuevo componente'} onClose={() => setForm(null)}>
          <ComponentForm component={form.component} onClose={() => setForm(null)} />
        </Dialog>
      )}
    </>
  )
}
