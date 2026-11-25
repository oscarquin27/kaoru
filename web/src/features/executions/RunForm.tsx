// Registro de un RUN. No se edita después: quien ejecuta y la fecha de registro los pone el servidor.
import { useRef } from 'react'
import type { FieldError } from '../../shared/api/errors.ts'
import type { RunResult } from '../../shared/api/types.ts'
import { fromLocalInput, toLocalInput } from '../../shared/lib/format.ts'
import { RESULT_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateRun } from './api.ts'
import { TargetFields, resolveBuild, validateNewBuild, type Registered } from './Target.tsx'

function validateRun(data: FormData): FieldError[] {
  const errors = validateNewBuild(data)
  if (errors.length > 0) return errors
  if (new Date(text(data, 'executedAt')).getTime() > Date.now()) {
    return [{ field: 'executedAt', code: 'invalid', message: 'La fecha de ejecución no puede ser futura.' }]
  }
  return []
}

/**
 * `systemKey` es el sistema del TC: de ahí salen los BUILD y los ambientes que se ofrecen.
 * `requirement` es el REQ desde el que se registra: un BUILD nuevo lo incluye.
 */
export function RunForm({ testCase, systemKey, requirement, onClose }: { testCase: string; systemKey: string; requirement?: string; onClose: () => void }) {
  const create = useCreateRun()
  const registered = useRef<Registered>(new Map())
  return (
    <Form
      label="Registrar ejecución"
      submitLabel="Registrar RUN"
      validate={validateRun}
      onSubmit={async (data) =>
        create.mutateAsync({
          testCase,
          build: await resolveBuild(data, { system: systemKey, requirement, registered: registered.current }),
          environment: text(data, 'environment'),
          result: text(data, 'result') as RunResult,
          notes: text(data, 'notes'),
          executedAt: fromLocalInput(text(data, 'executedAt')) ?? '',
        })
      }
      onDone={onClose}
      onCancel={onClose}
    >
      <p className="soft">Un RUN queda como evidencia y no se puede editar después de registrarlo.</p>
      <TargetFields systemKey={systemKey} official />
      <div className="form__row">
        <SelectField name="result" label="Resultado" required defaultValue="">
          <option value="">Elige el resultado</option>
          {options(RESULT_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
        <TextField name="executedAt" label="Fecha de ejecución" type="datetime-local" required defaultValue={toLocalInput(new Date())} />
      </div>
      <TextAreaField name="notes" label="Notas" />
    </Form>
  )
}
