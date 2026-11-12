// Lista de defectos con filtros en la URL y alta. `?nuevo=1` abre el formulario (acción de Ctrl+K).
import { useCan, useUserOptions, useWorkflow } from '../../shared/api/session.ts'
import { SEVERITY_LABELS, options } from '../../shared/lib/labels.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Async } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterInput, FilterSelect, Pager } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useDefects } from './api.ts'
import { DefectForm } from './DefectForm.tsx'
import { DefectsTable } from './DefectsTable.tsx'

export default function DefectsPage() {
  const can = useCan()
  const users = useUserOptions()
  const { params, pick, setFilter } = useUrlFilters()
  const states = useWorkflow('defect').data?.states ?? []
  const page = Number(params.get('page')) || 1
  const defects = useDefects({ ...pick(['state', 'severity', 'requirement', 'assignee', 'q']), page: page > 1 ? page : undefined })
  const creating = params.has('nuevo') && can('def.create')

  return (
    <>
      <Crumbs items={[{ label: 'Defectos' }]} />
      <PageHead
        title="Defectos"
        subtitle="Tiempo en estado, sin SLA."
        actions={
          can('def.create') && (
            <button className="btn btn--primary" type="button" onClick={() => setFilter('nuevo', '1')}>
              <Icon name="plus" />
              Nuevo DEF
            </button>
          )
        }
      />
      <FilterBar views={{ screen: 'defects', names: ['state', 'severity', 'requirement', 'assignee', 'q'] }}>
        <FilterSelect name="state" label="Estado" options={states.map((state) => ({ value: state.key, label: state.name }))} />
        <FilterSelect name="severity" label="Severidad" options={options(SEVERITY_LABELS)} />
        <FilterSelect name="assignee" label="Asignado a" options={users} />
        <FilterInput name="requirement" label="REQ" placeholder="REQ-2026-0001" />
        <FilterInput name="q" label="Texto en el título" />
      </FilterBar>
      <Async query={defects}>
        {(data) => (
          <>
            <DefectsTable items={data.items} showRequirement />
            <Pager page={data.page} size={data.size} total={data.total} noun="defectos" />
          </>
        )}
      </Async>
      {creating && (
        <Dialog title="Abrir defecto" onClose={() => setFilter('nuevo', '')}>
          <DefectForm onClose={() => setFilter('nuevo', '')} />
        </Dialog>
      )}
    </>
  )
}
