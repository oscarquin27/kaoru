// Registro de un despliegue (DEP) en un ambiente: de un BUILD o de un PKG, uno de los dos.
// No se edita ni se borra después.
import { useBuildOptions, usePackageOptions } from '../../shared/api/catalog.ts'
import type { FieldError } from '../../shared/api/errors.ts'
import type { DeploymentResult, Environment } from '../../shared/api/types.ts'
import { fromLocalInput, toLocalInput } from '../../shared/lib/format.ts'
import { DEPLOYMENT_RESULT_LABELS, options } from '../../shared/lib/labels.ts'
import { Form, SelectField, TextAreaField, TextField, optional, text } from '../../shared/ui/Form.tsx'
import { useRegisterDeployment } from './api.ts'

function validateDeployment(data: FormData): FieldError[] {
  const errors: FieldError[] = []
  const build = text(data, 'build')
  const bundle = text(data, 'package')
  if (!build && !bundle) errors.push({ field: 'build', code: 'required', message: 'Elige un BUILD o un PKG.' })
  if (build && bundle) errors.push({ field: 'package', code: 'invalid', message: 'Un despliegue es de un BUILD o de un PKG, no de los dos.' })
  if (new Date(text(data, 'executedAt')).getTime() > Date.now()) {
    errors.push({ field: 'executedAt', code: 'invalid', message: 'La fecha del despliegue no puede ser futura.' })
  }
  return errors
}

export function DeploymentForm({ environment, onClose }: { environment: Environment; onClose: () => void }) {
  const register = useRegisterDeployment()
  const builds = useBuildOptions(environment.system.key).data?.items ?? []
  const packages = usePackageOptions(environment.system.key).data?.items ?? []
  return (
    <Form
      label="Registrar despliegue"
      submitLabel="Registrar DEP"
      validate={validateDeployment}
      onSubmit={(data) =>
        register.mutateAsync({
          environment: environment.code,
          build: optional(data, 'build'),
          package: optional(data, 'package'),
          result: text(data, 'result') as DeploymentResult,
          notes: text(data, 'notes'),
          executedAt: fromLocalInput(text(data, 'executedAt')) ?? '',
        })
      }
      onDone={onClose}
      onCancel={onClose}
    >
      <p className="soft">
        Lo instalado en {environment.name} se lee de su último despliegue exitoso. Un despliegue no se edita ni se borra después de registrarlo.
      </p>
      <div className="form__row">
        <SelectField name="build" label="BUILD" defaultValue="" hint="Elige un BUILD o un PKG, uno de los dos.">
          <option value="">Ninguno</option>
          {builds.map((build) => (
            <option key={build.code} value={build.code}>
              {build.version} · {build.code}
            </option>
          ))}
        </SelectField>
        <SelectField name="package" label="PKG" defaultValue="">
          <option value="">Ninguno</option>
          {packages.map((item) => (
            <option key={item.code} value={item.code}>
              {item.code} · {item.title}
            </option>
          ))}
        </SelectField>
      </div>
      <div className="form__row">
        <SelectField name="result" label="Resultado" required defaultValue="">
          <option value="">Elige el resultado</option>
          {options(DEPLOYMENT_RESULT_LABELS).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </SelectField>
        <TextField name="executedAt" label="Fecha del despliegue" type="datetime-local" required defaultValue={toLocalInput(new Date())} />
      </div>
      <TextAreaField name="notes" label="Notas" />
    </Form>
  )
}
