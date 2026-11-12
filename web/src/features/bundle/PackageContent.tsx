// Contenido de un paquete: sus BUILD y sus REQ. Solo cambia mientras el PKG está en armado.
import { useState } from 'react'
import { useBuildOptions } from '../../shared/api/catalog.ts'
import type { Package } from '../../shared/api/types.ts'
import { shortChecksum } from '../../shared/lib/format.ts'
import { StateChip } from '../../shared/ui/chips.tsx'
import { Code } from '../../shared/ui/Code.tsx'
import { Empty, ErrorNotice } from '../../shared/ui/feedback.tsx'
import { Form, SelectField, text } from '../../shared/ui/Form.tsx'
import { Icon } from '../../shared/ui/Icon.tsx'
import { Dialog } from '../../shared/ui/Modal.tsx'
import { useAddPackageBuild, useAddPackageRequirement, useRemovePackageBuild, useRemovePackageRequirement, useRequirementsInState } from './api.ts'

function AddBuildForm({ bundle, onClose }: { bundle: Package; onClose: () => void }) {
  const add = useAddPackageBuild(bundle.code)
  const builds = (useBuildOptions(bundle.system.key).data?.items ?? []).filter((build) => !bundle.builds.some((item) => item.code === build.code))
  return (
    <Form label="Agregar BUILD" submitLabel="Agregar BUILD" onSubmit={(data) => add.mutateAsync(text(data, 'build'))} onDone={onClose} onCancel={onClose}>
      <SelectField name="build" label="BUILD" required defaultValue="" hint={`Solo se ofrecen los BUILD de ${bundle.system.name} que no están en el paquete.`}>
        <option value="">Elige el BUILD</option>
        {builds.map((build) => (
          <option key={build.code} value={build.code}>
            {build.version} · {build.code} · {shortChecksum(build.checksum)}
          </option>
        ))}
      </SelectField>
    </Form>
  )
}

function AddRequirementForm({ bundle, onClose }: { bundle: Package; onClose: () => void }) {
  const add = useAddPackageRequirement(bundle.code)
  // Entran los REQ «Aprobado QA» y los que ya están «En paquete» (un diferido pasa al siguiente PKG).
  const approved = useRequirementsInState(bundle.system.key, 'approved_qa').data?.items ?? []
  const packaged = useRequirementsInState(bundle.system.key, 'in_package').data?.items ?? []
  const requirements = [...approved, ...packaged].filter((requirement) => !bundle.requirements.some((item) => item.code === requirement.code))
  return (
    <Form label="Agregar REQ" submitLabel="Agregar REQ" onSubmit={(data) => add.mutateAsync(text(data, 'requirement'))} onDone={onClose} onCancel={onClose}>
      <SelectField
        name="requirement"
        label="Requerimiento"
        required
        defaultValue=""
        hint="Solo los REQ del sistema en «Aprobado QA» o «En paquete». Uno «Aprobado QA» pasa a «En paquete» al agregarlo."
      >
        <option value="">Elige el REQ</option>
        {requirements.map((requirement) => (
          <option key={requirement.code} value={requirement.code}>
            {requirement.code} · {requirement.title} · {requirement.state.name}
          </option>
        ))}
      </SelectField>
    </Form>
  )
}

export function PackageContent({ bundle, editable }: { bundle: Package; editable: boolean }) {
  const [adding, setAdding] = useState<'build' | 'requirement' | null>(null)
  const removeBuild = useRemovePackageBuild(bundle.code)
  const removeRequirement = useRemovePackageRequirement(bundle.code)
  const busy = removeBuild.isPending || removeRequirement.isPending
  const error = removeBuild.error ?? removeRequirement.error
  const close = () => setAdding(null)

  return (
    <div className="stack">
      {!editable && <p className="soft">El contenido de un paquete solo cambia mientras está «En armado».</p>}
      {error && <ErrorNotice error={error} />}
      <section className="stack" aria-labelledby="h-pkg-builds">
        <div className="row row--between">
          <h3 className="h2" id="h-pkg-builds">
            BUILD del paquete
          </h3>
          {editable && (
            <button className="btn" type="button" onClick={() => setAdding('build')}>
              <Icon name="plus" />
              Agregar BUILD
            </button>
          )}
        </div>
        {bundle.builds.length === 0 ? (
          <Empty>Sin BUILD. El paquete necesita al menos uno para pasar a pruebas.</Empty>
        ) : (
          <div className="surface table-wrap">
            <table className="table" aria-label="BUILD del paquete">
              <thead>
                <tr>
                  <th>BUILD</th>
                  <th>Versión</th>
                  <th>SHA-256</th>
                  {editable && (
                    <th className="col-actions">
                      <span className="sr-only">Acciones</span>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {bundle.builds.map((build) => (
                  <tr key={build.code}>
                    <td>
                      <Code code={build.code} />
                    </td>
                    <td className="mono">{build.version}</td>
                    <td>
                      <span className="mono" title={build.checksum}>
                        {shortChecksum(build.checksum)}
                      </span>
                    </td>
                    {editable && (
                      <td className="col-actions">
                        <button className="btn" type="button" aria-label={`Quitar ${build.code}`} disabled={busy} onClick={() => removeBuild.mutate(build.code)}>
                          Quitar
                        </button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="stack" aria-labelledby="h-pkg-reqs">
        <div className="row row--between">
          <h3 className="h2" id="h-pkg-reqs">
            REQ del paquete
          </h3>
          {editable && (
            <button className="btn" type="button" onClick={() => setAdding('requirement')}>
              <Icon name="plus" />
              Agregar REQ
            </button>
          )}
        </div>
        {bundle.requirements.length === 0 ? (
          <Empty>Sin REQ. El paquete necesita al menos uno para pasar a pruebas.</Empty>
        ) : (
          <div className="surface table-wrap">
            <table className="table" aria-label="REQ del paquete">
              <thead>
                <tr>
                  <th>REQ</th>
                  <th>Requerimiento</th>
                  <th>Estado</th>
                  {editable && (
                    <th className="col-actions">
                      <span className="sr-only">Acciones</span>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {bundle.requirements.map((requirement) => (
                  <tr key={requirement.code}>
                    <td>
                      <Code code={requirement.code} />
                    </td>
                    <td>{requirement.title}</td>
                    <td>
                      <StateChip state={requirement.state} />
                    </td>
                    {editable && (
                      <td className="col-actions">
                        <button
                          className="btn"
                          type="button"
                          aria-label={`Quitar ${requirement.code}`}
                          title="El estado del REQ no cambia al quitarlo"
                          disabled={busy}
                          onClick={() => removeRequirement.mutate(requirement.code)}
                        >
                          Quitar
                        </button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {adding === 'build' && (
        <Dialog title={`Agregar BUILD a ${bundle.code}`} onClose={close}>
          <AddBuildForm bundle={bundle} onClose={close} />
        </Dialog>
      )}
      {adding === 'requirement' && (
        <Dialog title={`Agregar REQ a ${bundle.code}`} onClose={close}>
          <AddRequirementForm bundle={bundle} onClose={close} />
        </Dialog>
      )}
    </div>
  )
}
