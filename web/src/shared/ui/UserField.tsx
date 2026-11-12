// Campo para elegir a una persona como responsable o asignada.
import { useUserOptions } from '../api/session.ts'
import type { UserRef } from '../api/types.ts'
import { SelectField } from './Form.tsx'

export function UserField({ name, label, current }: { name: string; label: string; current?: UserRef }) {
  const listed = useUserOptions()
  // Quien ya está asignado aparece aunque la lista (acotada) no lo traiga.
  const options = current && !listed.some((option) => option.value === current.id) ? [{ value: current.id, label: current.displayName }, ...listed] : listed
  return (
    <SelectField name={name} label={label} defaultValue={current?.id ?? ''} key={options.length} hint="Aparece quien ya entró a Kaoru alguna vez.">
      <option value="">Sin asignar</option>
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </SelectField>
  )
}
