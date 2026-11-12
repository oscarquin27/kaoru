// Registro de un BUILD con su manifiesto. Un BUILD no se edita ni se borra después.
import { useState } from 'react'
import { useNavigate } from 'react-router'
import type { FieldError } from '../../shared/api/errors.ts'
import { useSystems } from '../../shared/api/session.ts'
import { CheckGroupField, Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useRegisterBuild, useSystemRequirements } from './api.ts'

function validateBuild(data: FormData): FieldError[] {
  if (!/^[0-9a-f]{64}$/.test(text(data, 'checksum').toLowerCase())) {
    return [{ field: 'checksum', code: 'invalid', message: 'Pega el SHA-256 del artefacto: 64 caracteres hexadecimales.' }]
  }
  return []
}

export function BuildForm({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate()
  const systems = useSystems().data?.items ?? []
  const register = useRegisterBuild()
  // Los REQ que se ofrecen dependen del sistema elegido.
  const [system, setSystem] = useState('')
  const requirements = useSystemRequirements(system).data?.items ?? []

  async function submit(data: FormData) {
    const created = await register.mutateAsync({
      system: text(data, 'system'),
      version: text(data, 'version'),
      checksum: text(data, 'checksum').toLowerCase(),
      requirements: data.getAll('requirements') as string[],
      migrations: text(data, 'migrations'),
      dependencies: text(data, 'dependencies'),
      notes: text(data, 'notes'),
    })
    void navigate(`/paquetes/builds/${created.code}`)
  }

  return (
    <Form label="Registrar BUILD" submitLabel="Registrar BUILD" validate={validateBuild} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <p className="soft">Un BUILD no se edita ni se borra después de registrarlo: revisa la versión, el checksum y el manifiesto antes de guardar.</p>
      <div className="form__row">
        <SelectField name="system" label="Sistema" required value={system} onChange={(event) => setSystem(event.target.value)}>
          <option value="">Elige un sistema</option>
          {systems.map((item) => (
            <option key={item.key} value={item.key}>
              {item.name}
            </option>
          ))}
        </SelectField>
        <TextField name="version" label="Versión" required maxLength={80} placeholder="2.14.0" />
      </div>
      <TextField name="checksum" label="SHA-256 del artefacto" required spellCheck={false} autoComplete="off" hint="El mismo artefacto no se registra dos veces." />
      {system === '' ? (
        <p className="muted">Elige un sistema para ver sus requerimientos.</p>
      ) : requirements.length === 0 ? (
        <p className="muted">Este sistema no tiene requerimientos.</p>
      ) : (
        <CheckGroupField
          key={system}
          name="requirements"
          label="REQ incluidos en el BUILD"
          choices={requirements.map((requirement) => ({ value: requirement.code, label: `${requirement.code} · ${requirement.title}` }))}
        />
      )}
      <TextAreaField name="migrations" label="Migraciones" hint="Una por línea." />
      <TextAreaField name="dependencies" label="Dependencias" hint="Una por línea, con su versión." />
      <TextAreaField name="notes" label="Notas" />
    </Form>
  )
}
