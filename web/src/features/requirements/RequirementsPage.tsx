// Lista de requerimientos con filtros en la URL y alta.
import { useState } from 'react'
import { useCan, useUserOptions, useSystems, useWorkflow } from '../../shared/api/session.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Async } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { RequirementsTable } from '../../shared/ui/RequirementsTable.tsx'
import { useChangeOptions, useRequirements } from './api.ts'
import { RequirementForm } from './RequirementForm.tsx'

export default function RequirementsPage() {
  const can = useCan()
  const users = useUserOptions()
  const { params, pick } = useUrlFilters()
  const [creating, setCreating] = useState(false)
  const systems = useSystems().data?.items ?? []
  const states = useWorkflow('requirement').data?.states ?? []
  const changes = useChangeOptions().data?.items ?? []
  const page = Number(params.get('page')) || 1
  const requirements = useRequirements({ ...pick(['system', 'state', 'change', 'owner', 'q']), page: page > 1 ? page : undefined })

  return (
    <>
      <Crumbs items={[{ label: 'Requerimientos' }]} />
      <PageHead
        title="Requerimientos"
        subtitle="Cada REQ con su estado, su cambio y su responsable."
        actions={
          can('req.create') && (
            <button className="btn btn--primary" type="button" onClick={() => setCreating(true)}>
              <Icon name="plus" />
              Nuevo REQ
            </button>
          )
        }
      />
      <FilterBar views={{ screen: 'requirements', names: ['system', 'state', 'change', 'owner', 'q'] }}>
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
        <FilterSelect name="state" label="Estado" options={states.map((state) => ({ value: state.key, label: state.name }))} />
        <FilterSelect name="change" label="CHG" options={changes.map((change) => ({ value: change.code, label: change.code }))} />
        <FilterSelect name="owner" label="Responsable" options={users} />
        <FilterInput name="q" label="Texto en el título" />
      </FilterBar>
      <Async query={requirements}>
        {(data) => (
          <>
            <RequirementsTable items={data.items} />
            <Pager page={data.page} size={data.size} total={data.total} noun="requerimientos" />
          </>
        )}
      </Async>
      {creating && (
        <Dialog title="Nuevo requerimiento" onClose={() => setCreating(false)}>
          <RequirementForm onClose={() => setCreating(false)} />
        </Dialog>
      )}
    </>
  )
}
