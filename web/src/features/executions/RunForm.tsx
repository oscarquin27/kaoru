// Registro de un RUN. No se edita después: quien ejecuta y la fecha de registro los pone el servidor.
import type { FieldError } from '../../shared/api/errors.ts'
import type { RunResult } from '../../shared/api/types.ts'
import { fromLocalInput, toLocalInput } from '../../shared/lib/format.ts'
import { RESULT_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateRun } from './api.ts'

function validateRun(data: FormData): FieldError[] {
  const errors: FieldError[] = []
  if (!/^[0-9a-f]{64}$/.test(text(data, 'buildChecksum').toLowerCase())) {
    errors.push({ field: 'buildChecksum', code: 'invalid', message: 'Pega el SHA-256 del BUILD: 64 caracteres hexadecimales.' })
  }
  if (new Date(text(data, 'executedAt')).getTime() > Date.now()) {
    errors.push({ field: 'executedAt', code: 'invalid', message: 'La fecha de ejecución no puede ser futura.' })
  }
  return errors
}

export function RunForm({ testCase, onClose }: { testCase: string; onClose: () => void }) {
  const create = useCreateRun()
  return (
    <Form
      label="Registrar ejecución"
      submitLabel="Registrar RUN"
      validate={validateRun}
      onSubmit={(data) =>
        create.mutateAsync({
          testCase,
          buildVersion: text(data, 'buildVersion'),
          buildChecksum: text(data, 'buildChecksum').toLowerCase(),
          environmentName: text(data, 'environmentName'),
          result: text(data, 'result') as RunResult,
          notes: text(data, 'notes'),
          executedAt: fromLocalInput(text(data, 'executedAt')) ?? '',
        })
      }
      onDone={onClose}
      onCancel={onClose}
    >
      <p className="soft">Un RUN queda como evidencia y no se puede editar después de registrarlo.</p>
      <div className="form__row">
        <TextField name="buildVersion" label="Versión del BUILD" required placeholder="2.14.0" />
        <TextField name="environmentName" label="Ambiente" required placeholder="QA-1 (línea base)" />
      </div>
      <TextField name="buildChecksum" label="SHA-256 del BUILD" required spellCheck={false} autoComplete="off" />
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
