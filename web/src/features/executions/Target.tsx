// Sobre qué se ejecuta: un BUILD y un ambiente del catálogo, del sistema del TC o del REQ.
import { Link } from 'react-router'
import { useBuildOptions, useEnvironmentOptions } from '../../shared/api/catalog.ts'
import { shortChecksum } from '../../shared/lib/format.ts'
import { ENVIRONMENT_KIND_LABELS } from '../../shared/lib/labels.ts'
import { SelectField } from '../../shared/ui/Form.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'

/** Selectores de BUILD y ambiente. Con `official` (un RUN) no se ofrece producción: QA no ejecuta ahí. */
export function TargetFields({ systemKey, official }: { systemKey: string; official?: boolean }) {
  const builds = useBuildOptions(systemKey)
  const environments = useEnvironmentOptions(systemKey)
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
      {(missingBuilds || missingEnvironments) && (
        <div className="notice" role="status">
          <Icon name="triangle" />
          <div className="notice__main">
            {missingBuilds && (
              <p>
                Este sistema no tiene ningún BUILD registrado. Regístralo en <Link to="/paquetes/builds">Paquetes · BUILD</Link>.
              </p>
            )}
            {missingEnvironments && (
              <p>
                Este sistema no tiene ambientes {official ? 'de prueba ' : ''}registrados. Regístralos en <Link to="/ambientes">Ambientes</Link>.
              </p>
            )}
          </div>
        </div>
      )}
      <div className="form__row">
        <SelectField name="build" label="BUILD" required defaultValue="">
          <option value="">Elige el BUILD</option>
          {buildItems.map((build) => (
            <option key={build.code} value={build.code}>
              {build.version} · {build.code} · {shortChecksum(build.checksum)}
            </option>
          ))}
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
    </>
  )
}
