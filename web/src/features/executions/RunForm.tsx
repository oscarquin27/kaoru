// Registro de un RUN. No se edita después: quien ejecuta y la fecha de registro los pone el servidor.
import type { FieldError } from '../../shared/api/errors.ts'
import type { RunResult } from '../../shared/api/types.ts'
import { fromLocalInput, toLocalInput } from '../../shared/lib/format.ts'
import { RESULT_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateRun } from './api.ts'
import { TargetFields } from './Target.tsx'

function validateRun(data: FormData): FieldError[] {
  if (new Date(text(data, 'executedAt')).getTime() > Date.now()) {
    return [{ field: 'executedAt', code: 'invalid', message: 'La fecha de ejecución no puede ser futura.' }]
  }
  return []
}

/** `systemKey` es el sistema del TC: de ahí salen los BUILD y los ambientes que se ofrecen. */
export function RunForm({ testCase, systemKey, onClose }: { testCase: string; systemKey: string; onClose: () => void }) {
  const create = useCreateRun()
  return (
    <Form
      label="Registrar ejecución"
      submitLabel="Registrar RUN"
      validate={validateRun}
      onSubmit={(data) =>
        create.mutateAsync({
          testCase,
          build: text(data, 'build'),
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
