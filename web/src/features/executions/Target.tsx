// Sobre qué se ejecuta: un BUILD y un ambiente del catálogo, del sistema del TC o del REQ.
// Si el BUILD todavía no existe, se elige «Registrar un BUILD nuevo», se escriben su versión y su
// checksum y se registra al guardar, antes del RUN: nadie tiene que ir a Paquetes primero.
import { useState } from 'react'
import { Link } from 'react-router'
import { useBuildOptions, useEnvironmentOptions } from '../../shared/api/catalog.ts'
import { api } from '../../shared/api/client.ts'
import { ApiError, toApiError, type FieldError } from '../../shared/api/errors.ts'
import { useCan } from '../../shared/api/session.ts'
import type { Build } from '../../shared/api/types.ts'
import { shortChecksum } from '../../shared/lib/format.ts'
import { ENVIRONMENT_KIND_LABELS } from '../../shared/lib/labels.ts'
import { SelectField, TextField, text } from '../../shared/ui/Form.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'

/** Valor del selector que pide registrar un BUILD nuevo. */
export const NEW_BUILD = '__new__'
const VERSION = 'newBuildVersion'
const CHECKSUM = 'newBuildChecksum'

/** Selectores de BUILD y ambiente. Con `official` (un RUN) no se ofrece producción: QA no ejecuta ahí. */
export function TargetFields({ systemKey, official }: { systemKey: string; official?: boolean }) {
  const can = useCan()
  const canRegister = can('build.register')
  const builds = useBuildOptions(systemKey)
  const environments = useEnvironmentOptions(systemKey)
  const [build, setBuild] = useState('')
  const buildItems = builds.data?.items ?? []
  const environmentItems = (environments.data?.items ?? []).filter((environment) => !official || environment.kind !== 'production')
  const missingBuilds = builds.data !== undefined && buildItems.length === 0
  const missingEnvironments = environments.data !== undefined && environmentItems.length === 0
  const failed = builds.error ?? environments.error

  return (
    <>
      {failed && (
        <div className="notice notice--error" role="alert">
          <Icon name="triangle" />
          <div className="notice__main">No se pudo leer el catálogo de BUILD y ambientes. {failed.message}</div>
        </div>
      )}
      {((missingBuilds && build !== NEW_BUILD) || missingEnvironments) && (
        <div className="notice" role="status">
          <Icon name="triangle" />
          <div className="notice__main">
            {missingBuilds &&
              build !== NEW_BUILD &&
              (canRegister ? (
                <p>Este sistema no tiene ningún BUILD registrado. Elige «Registrar un BUILD nuevo» y se registra al guardar.</p>
              ) : (
                <p>
                  Este sistema no tiene ningún BUILD registrado. Regístralo en <Link to="/paquetes/builds">Paquetes · BUILD</Link>.
                </p>
              ))}
            {missingEnvironments && (
              <p>
                Este sistema no tiene ambientes {official ? 'de prueba ' : ''}registrados. Regístralos en <Link to="/ambientes">Ambientes</Link>.
              </p>
            )}
          </div>
        </div>
      )}
      <div className="form__row">
        <SelectField name="build" label="BUILD" required value={build} onChange={(event) => setBuild(event.target.value)}>
          <option value="">Elige el BUILD</option>
          {buildItems.map((item) => (
            <option key={item.code} value={item.code}>
              {item.version} · {item.code} · {shortChecksum(item.checksum)}
            </option>
          ))}
          {canRegister && <option value={NEW_BUILD}>＋ Registrar un BUILD nuevo…</option>}
        </SelectField>
        <SelectField name="environment" label="Ambiente" required defaultValue="" hint={official ? 'No se ofrecen ambientes de producción.' : undefined}>
          <option value="">Elige el ambiente</option>
          {environmentItems.map((environment) => (
            <option key={environment.code} value={environment.code}>
              {environment.name} · {ENVIRONMENT_KIND_LABELS[environment.kind]}
            </option>
          ))}
        </SelectField>
      </div>
      {build === NEW_BUILD && (
        <div className="form__row">
          <TextField name={VERSION} label="Versión del BUILD nuevo" required maxLength={80} placeholder="2.14.0" />
          <TextField
            name={CHECKSUM}
            label="SHA-256 del artefacto"
            required
            spellCheck={false}
            autoComplete="off"
            hint="Se registra al guardar; su manifiesto lo completas después en Paquetes · BUILD."
          />
        </div>
      )}
    </>
  )
}

/** Valida el checksum del BUILD nuevo, si se pidió uno. */
export function validateNewBuild(data: FormData): FieldError[] {
  if (text(data, 'build') !== NEW_BUILD || /^[0-9a-f]{64}$/.test(text(data, CHECKSUM).toLowerCase())) return []
  return [{ field: CHECKSUM, code: 'invalid', message: 'Pega el SHA-256 del artefacto: 64 caracteres hexadecimales.' }]
}

/**
 * Lo ya registrado en este formulario, para que un reintento después de un error no registre dos
 * veces lo mismo. Una por formulario (useRef).
 */
export type Registered = Map<string, string>

/**
 * El código del BUILD elegido; si se pidió uno nuevo, lo registra antes y devuelve su código.
 * `requirement` es el REQ que el BUILD incluye, si se conoce.
 */
export async function resolveBuild(data: FormData, { system, requirement, registered }: { system: string; requirement?: string; registered: Registered }): Promise<string> {
  const chosen = text(data, 'build')
  if (chosen !== NEW_BUILD) return chosen
  const version = text(data, VERSION)
  const checksum = text(data, CHECKSUM).toLowerCase()
  const key = `build|${version}|${checksum}`
  const known = registered.get(key)
  if (known) return known
  try {
    const created = await api.post<Build>('/api/builds', {
      system,
      version,
      checksum,
      requirements: requirement ? [requirement] : [],
      migrations: '',
      dependencies: '',
      notes: '',
    })
    registered.set(key, created.code)
    return created.code
  } catch (caught) {
    // Los errores del BUILD se pintan bajo sus campos de este formulario.
    const error = toApiError(caught)
    const fields: Record<string, string> = { version: VERSION, checksum: CHECKSUM }
    const fieldErrors: FieldError[] = error.fieldErrors.map((item) => ({ ...item, field: fields[item.field] ?? 'build' }))
    throw new ApiError(error.status, error.code, error.message, fieldErrors, error.traceId)
  }
}
