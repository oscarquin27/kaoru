// Alta y edición de un DEF. Abrir uno exige el REQ y el RUN que lo reprodujo.
import { useNavigate } from 'react-router'
import type { FieldError } from '../../shared/api/errors.ts'
import type { Defect, Severity } from '../../shared/api/types.ts'
import { SEVERITY_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, text } from '../../shared/ui/Form.tsx'
import { useCreateDefect, useUpdateDefect } from './api.ts'

interface DefectFormProps {
  /** Con `defect`, edita. */
  defect?: Defect
  /** REQ ya conocido (alta desde la ficha del REQ). */
  requirement?: string
  /** RUN del REQ entre los que elegir; sin lista, se escribe el código. */
  runs?: { code: string; label: string }[]
  onClose: () => void
}

function validateCodes(data: FormData): FieldError[] {
  const errors: FieldError[] = []
  for (const [field, prefix] of [['requirement', 'REQ'], ['run', 'RUN']] as const) {
    const value = text(data, field)
    if (data.has(field) && value && !new RegExp(`^${prefix}-\\d{4}-\\d{4,}$`).test(value)) {
      errors.push({ field, code: 'invalid', message: `Escribe el ID completo, por ejemplo ${prefix}-2026-0001.` })
    }
  }
  return errors
}

export function DefectForm({ defect, requirement, runs, onClose }: DefectFormProps) {
  const navigate = useNavigate()
  const create = useCreateDefect()
  const update = useUpdateDefect(defect?.code ?? '')

  async function submit(data: FormData) {
    const common = { title: text(data, 'title'), description: text(data, 'description'), severity: text(data, 'severity') as Severity }
    if (defect) {
      await update.mutateAsync(common)
      return
    }
    const created = await create.mutateAsync({ ...common, requirement: requirement ?? text(data, 'requirement'), run: text(data, 'run') })
    void navigate(`/defectos/${created.code}`)
  }

  return (
    <Form label={defect ? 'Editar defecto' : 'Abrir defecto'} submitLabel={defect ? 'Guardar' : 'Abrir DEF'} validate={validateCodes} onSubmit={submit} onDone={onClose} onCancel={onClose}>
      <TextField name="title" label="Título" required defaultValue={defect?.title} />
      <TextAreaField name="description" label="Descripción" required defaultValue={defect?.description} hint="Qué pasó, qué se esperaba y cómo reproducirlo." />
      <div className="form__row">
        <SelectField name="severity" label="Severidad" required defaultValue={defect?.severity ?? ''}>
          <option value="">Elige la severidad</option>
          {options(SEVERITY_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
        {!defect && !requirement && <TextField name="requirement" label="Requerimiento (REQ)" required placeholder="REQ-2026-0001" />}
        {!defect &&
          (runs ? (
            <SelectField name="run" label="RUN que lo reprodujo" required defaultValue="">
              <option value="">Elige un RUN</option>
              {runs.map((run) => (
                <option key={run.code} value={run.code}>
                  {run.label}
                </option>
              ))}
            </SelectField>
          ) : (
            <TextField name="run" label="RUN que lo reprodujo" required placeholder="RUN-2026-0001" />
          ))}
      </div>
      {!defect && <p className="muted">Un hallazgo de una exploración no se abre como defecto sin reproducirlo antes en un RUN.</p>}
    </Form>
  )
}
