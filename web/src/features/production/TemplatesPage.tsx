// Plantillas del checklist de producción: una por sistema, versionada. No se editan; se publica otra versión.
import { useState } from 'react'
import { useCan, useSystems } from '../../shared/api/session.ts'
import type { ChecklistTemplate } from '../../shared/api/types.ts'
import { formatDateTime } from '../../shared/lib/format.ts'
import { useUrlFilters } from '../../shared/lib/urlFilters.ts'
import { Async, Empty } from '../../shared/ui/feedback.tsx'
import { FilterBar, FilterSelect } from '../../shared/ui/Filters.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useChecklistTemplates } from './api.ts'
import { TemplateForm } from './forms.tsx'

/** La API trae las versiones con la más reciente de cada sistema primero. */
function bySystem(templates: ChecklistTemplate[]): ChecklistTemplate[][] {
  const groups = new Map<string, ChecklistTemplate[]>()
  for (const template of templates) groups.set(template.system.key, [...(groups.get(template.system.key) ?? []), template])
  return [...groups.values()]
}

function TemplateCard({ versions, canPublish, onPublish }: { versions: ChecklistTemplate[]; canPublish: boolean; onPublish: (current: ChecklistTemplate) => void }) {
  const [current, ...older] = versions
  if (!current) return null
  return (
    <section className="surface pad stack" aria-labelledby={`tpl-${current.system.key}`}>
      <div className="row row--between">
        <div className="stack-s">
          <h2 className="h2" id={`tpl-${current.system.key}`}>
            {current.system.name} · v{current.version}
          </h2>
          <span className="muted">
            Publicó {current.createdBy.displayName} · {formatDateTime(current.createdAt)}
          </span>
        </div>
        {canPublish && (
          <button className="btn" type="button" onClick={() => onPublish(current)}>
            <Icon name="plus" />
            Nueva versión
          </button>
        )}
      </div>
      <ol className="stack-s">
        {current.steps.map((step) => (
          <li key={step.position}>
            <strong>{step.title}</strong>
            {step.description && <span className="soft"> · {step.description}</span>}
          </li>
        ))}
      </ol>
      {older.length > 0 && (
        <p className="muted">
          Versiones anteriores: {older.map((template) => `v${template.version} (${template.steps.length === 1 ? '1 paso' : `${template.steps.length} pasos`})`).join(', ')}. Los checklists abiertos con ellas las conservan.
        </p>
      )}
    </section>
  )
}

export default function TemplatesPage() {
  const can = useCan()
  const { params } = useUrlFilters()
  const systems = useSystems().data?.items ?? []
  const templates = useChecklistTemplates(params.get('system') ?? undefined)
  // undefined: cerrado; null: una plantilla nueva; una plantilla: su siguiente versión.
  const [publishing, setPublishing] = useState<ChecklistTemplate | null | undefined>(undefined)
  const canPublish = can('chk.manage_template')

  return (
    <>
      <Crumbs items={[{ label: 'Producción', to: '/produccion' }, { label: 'Plantillas' }]} />
      <PageHead
        title="Plantillas del checklist"
        subtitle="Los pasos que se firman en cada pase a producción, por sistema. Una plantilla no se edita: se publica otra versión."
        actions={
          canPublish && (
            <button className="btn btn--primary" type="button" onClick={() => setPublishing(null)}>
              <Icon name="plus" />
              Publicar plantilla
            </button>
          )
        }
      />
      <FilterBar>
        <FilterSelect name="system" label="Sistema" options={systems.map((system) => ({ value: system.key, label: system.name }))} />
      </FilterBar>
      <Async query={templates}>
        {(data) =>
          data.items.length === 0 ? (
            <Empty>Ningún sistema tiene plantilla publicada. Sin plantilla no se puede abrir el checklist de un PKG.</Empty>
          ) : (
            <div className="stack">
              {bySystem(data.items).map((versions) => (
                <TemplateCard key={versions[0]?.system.key} versions={versions} canPublish={canPublish} onPublish={setPublishing} />
              ))}
            </div>
          )
        }
      </Async>
      {publishing !== undefined && (
        <Dialog title={publishing ? `Nueva versión de ${publishing.system.name}` : 'Publicar plantilla'} onClose={() => setPublishing(undefined)}>
          <TemplateForm current={publishing ?? undefined} onClose={() => setPublishing(undefined)} />
        </Dialog>
      )}
    </>
  )
}
