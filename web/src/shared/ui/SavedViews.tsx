// Vistas guardadas de una lista: sus filtros con nombre, personales. Compartir sigue siendo por URL.
import { useState } from 'react'
import { useSearchParams } from 'react-router'
import { api } from '../api/client.ts'
import { useApiMutation, useApiQuery } from '../api/query.ts'
import type { Items, SavedView, SavedViewScreen } from '../api/types.ts'
import { Form, TextField, text } from './Form.tsx'
import { ErrorNotice } from './feedback.tsx'
import { Dialog } from './Modal.tsx'

export interface ViewScope {
  screen: SavedViewScreen
  /** Los filtros de la lista: solo estos entran en una vista. */
  names: string[]
}

/** Los filtros activos como texto de URL, en orden estable para poder compararlos. */
function activeQuery(params: URLSearchParams, names: string[]): string {
  const picked = new URLSearchParams()
  for (const name of [...names].sort()) {
    const value = params.get(name)
    if (value) picked.set(name, value)
  }
  return picked.toString()
}

export function SavedViews({ screen, names }: ViewScope) {
  const [params, setParams] = useSearchParams()
  const [saving, setSaving] = useState(false)
  const current = activeQuery(params, names)
  const views = useApiQuery<Items<SavedView>>('/api/saved-views', { screen }).data?.items ?? []
  const save = useApiMutation((name: string) => api.post<SavedView>('/api/saved-views', { screen, name, query: current }))
  const remove = useApiMutation((id: string) => api.delete(`/api/saved-views/${id}`))
  const selected = views.find((view) => activeQuery(new URLSearchParams(view.query), names) === current)

  function apply(id: string) {
    const view = views.find((item) => item.id === id)
    setParams(
      (previous) => {
        const next = new URLSearchParams(previous)
        for (const name of [...names, 'page']) next.delete(name)
        new URLSearchParams(view?.query ?? '').forEach((value, name) => names.includes(name) && next.set(name, value))
        return next
      },
      { replace: true },
    )
  }

  return (
    <>
      <label className="field">
        Vista guardada
        <select className="select" value={selected?.id ?? ''} onChange={(event) => apply(event.target.value)}>
          <option value="">{current && !selected ? 'Filtros sin guardar' : 'Ninguna'}</option>
          {views.map((view) => (
            <option key={view.id} value={view.id}>
              {view.name}
            </option>
          ))}
        </select>
      </label>
      {selected ? (
        <button className="btn" type="button" disabled={remove.isPending} onClick={() => remove.mutate(selected.id)}>
          Quitar vista
        </button>
      ) : (
        <button className="btn" type="button" disabled={!current} title={current ? undefined : 'Elige algún filtro primero'} onClick={() => setSaving(true)}>
          Guardar vista
        </button>
      )}
      {remove.error && <ErrorNotice error={remove.error} />}
      {saving && (
        <Dialog title="Guardar vista" onClose={() => setSaving(false)}>
          <Form label="Guardar vista" submitLabel="Guardar" onSubmit={(data) => save.mutateAsync(text(data, 'name'))} onDone={() => setSaving(false)} onCancel={() => setSaving(false)}>
            <TextField name="name" label="Nombre de la vista" required maxLength={60} hint="Guarda los filtros que tienes puestos ahora. Solo la ves tú." />
          </Form>
        </Dialog>
      )}
    </>
  )
}
