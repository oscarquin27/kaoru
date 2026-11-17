// Selector de CHG para los formularios que registran algo de un cambio (un REQ, una actividad).
// Si el CHG todavía no existe, se elige «Crear un CHG nuevo», se escribe su título y se crea al
// guardar, antes de lo demás: nadie tiene que salir del formulario para darlo de alta primero.
import { useState } from 'react'
import { useChangeOptions } from '../api/catalog.ts'
import { api } from '../api/client.ts'
import { ApiError, toApiError, type FieldError } from '../api/errors.ts'
import { useSystems } from '../api/session.ts'
import type { Change } from '../api/types.ts'
import { SelectField, TextField, text } from './Form.tsx'

/** Valor del selector que pide crear un CHG nuevo. */
export const NEW_CHANGE = '__new__'
const TITLE = 'newChangeTitle'
const SYSTEM = 'newChangeSystem'

interface ChangeFieldProps {
  name?: string
  label?: string
  defaultValue?: string
  /** Texto de la opción vacía: «Sin cambio». */
  emptyLabel?: string
  hint?: string
  /** Pide el sistema del CHG nuevo. Sin él, el CHG nuevo toma el sistema del formulario. */
  askSystem?: boolean
}

export function ChangeField({ name = 'change', label = 'Cambio (CHG)', defaultValue = '', emptyLabel = 'Sin cambio', hint, askSystem = false }: ChangeFieldProps) {
  const changes = useChangeOptions().data?.items ?? []
  const systems = useSystems().data?.items ?? []
  const [value, setValue] = useState(defaultValue)
  // Un CHG que no está en la lista (de otra página o recién creado) se sigue mostrando.
  const listed = value === '' || value === NEW_CHANGE || changes.some((change) => change.code === value)

  return (
    <>
      <SelectField name={name} label={label} hint={hint} value={value} onChange={(event) => setValue(event.target.value)}>
        <option value="">{emptyLabel}</option>
        {!listed && <option value={value}>{value}</option>}
        {changes.map((change) => (
          <option key={change.code} value={change.code}>
            {change.code} · {change.title}
          </option>
        ))}
        <option value={NEW_CHANGE}>＋ Crear un CHG nuevo…</option>
      </SelectField>
      {value === NEW_CHANGE && (
        <div className="form__row">
          <TextField name={TITLE} label="Título del CHG nuevo" required maxLength={200} hint="Se crea al guardar, en ruta ligera; lo completas después en su ficha." />
          {askSystem && (
            <SelectField name={SYSTEM} label="Sistema del CHG" required defaultValue="">
              <option value="">Elige un sistema</option>
              {systems.map((system) => (
                <option key={system.key} value={system.key}>
                  {system.name}
                </option>
              ))}
            </SelectField>
          )}
        </div>
      )}
    </>
  )
}

/**
 * El código del CHG elegido; si se pidió uno nuevo, lo crea antes y devuelve su código.
 * `system` es el del formulario, para cuando el campo no pide el suyo. Sin CHG, devuelve ''.
 */
export async function resolveChange(data: FormData, { name = 'change', system }: { name?: string; system?: string } = {}): Promise<string> {
  const chosen = text(data, name)
  if (chosen !== NEW_CHANGE) return chosen.toUpperCase()
  try {
    const created = await api.post<Change>('/api/changes', { title: text(data, TITLE), system: system ?? text(data, SYSTEM), route: 'light' })
    return created.code
  } catch (caught) {
    // Los errores del CHG se pintan bajo sus campos de este formulario.
    const error = toApiError(caught)
    const fields: Record<string, string> = { title: TITLE, system: system ? 'system' : SYSTEM }
    const fieldErrors: FieldError[] = error.fieldErrors.map((item) => ({ ...item, field: fields[item.field] ?? item.field }))
    throw new ApiError(error.status, error.code, error.message, fieldErrors, error.traceId)
  }
}
