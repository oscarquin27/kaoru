// Impacto: los componentes (COMP) que toca un REQ o que ejercita un TC. De su cruce sale la
// regresión que el sistema propone para un paquete.
import { useState } from 'react'
import { Link } from 'react-router'
import { useComponentOptions, useImpact, useSetImpact } from '../api/catalog.ts'
import type { ComponentRef } from '../api/types.ts'
import { COMPONENT_KIND_LABELS } from '../lib/labels.ts'
import { ComponentChips } from './chips.tsx'
import { Async, BlockSkeleton } from './feedback.tsx'
import { CheckGroupField, Form } from './Form.tsx'
import { Icon } from './Icon.tsx'
import { Dialog } from './Modal.tsx'

interface ImpactProps {
  /** Ruta de la API con los componentes de la entidad. */
  path: string
  systemKey: string
}

/** Casillas con los componentes activos del sistema. Guarda la lista completa con PUT. */
export function ImpactForm({ path, systemKey, selected, onClose }: ImpactProps & { selected: ComponentRef[]; onClose: () => void }) {
  const catalog = useComponentOptions(systemKey)
  const save = useSetImpact(path)
  return (
    <Async query={catalog} skeleton={<BlockSkeleton />}>
      {(data) => {
        // Un componente ya declarado sigue en la lista aunque lo hayan retirado, para poder quitarlo.
        const listed = data.items.map((component) => ({ code: component.code, name: component.name, kind: component.kind }))
        const choices = [...listed, ...selected.filter((component) => !listed.some((item) => item.code === component.code))]
        return (
          <Form label="Declarar componentes" submitLabel="Guardar" onSubmit={(form) => save.mutateAsync(form.getAll('components') as string[])} onDone={onClose} onCancel={onClose}>
            {choices.length === 0 ? (
              <p className="soft">
                Este sistema no tiene componentes en el catálogo. Regístralos en <Link to="/pruebas/componentes">Pruebas · Componentes</Link>.
              </p>
            ) : (
              <CheckGroupField
                name="components"
                label="Componentes del sistema"
                choices={choices.map((component) => ({ value: component.code, label: `${component.name} · ${COMPONENT_KIND_LABELS[component.kind]}` }))}
                selected={selected.map((component) => component.code)}
              />
            )}
          </Form>
        )
      }}
    </Async>
  )
}

/** Tarjeta con los componentes declarados y, con permiso, el botón para cambiarlos. */
export function ImpactCard({ path, systemKey, title, hint, canEdit }: ImpactProps & { title: string; hint: string; canEdit: boolean }) {
  const impact = useImpact(path)
  const [editing, setEditing] = useState(false)
  return (
    <section className="surface pad stack" aria-label={title}>
      <div className="row row--between">
        <div>
          <h3 className="h3">{title}</h3>
          <p className="subtitle">{hint}</p>
        </div>
        {canEdit && (
          <button className="btn" type="button" onClick={() => setEditing(true)}>
            <Icon name="pencil" />
            Declarar componentes
          </button>
        )}
      </div>
      <Async query={impact}>{(data) => <ComponentChips components={data.items} empty="Sin componentes declarados." />}</Async>
      {editing && (
        <Dialog title={title} onClose={() => setEditing(false)}>
          <ImpactForm path={path} systemKey={systemKey} selected={impact.data?.items ?? []} onClose={() => setEditing(false)} />
        </Dialog>
      )}
    </section>
  )
}

/** El mismo control dentro de un diálogo, para un TC que no tiene pantalla propia. */
export function ImpactDialog({ path, systemKey, title, onClose }: ImpactProps & { title: string; onClose: () => void }) {
  const impact = useImpact(path)
  return (
    <Dialog title={title} onClose={onClose}>
      <Async query={impact}>{(data) => <ImpactForm path={path} systemKey={systemKey} selected={data.items} onClose={onClose} />}</Async>
    </Dialog>
  )
}
