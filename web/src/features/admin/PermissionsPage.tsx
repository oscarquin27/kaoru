// Permisos por rol: una matriz de casillas. Se guarda rol por rol.
import { useState } from 'react'
import { useCan } from '../../shared/api/session.ts'
import type { Role, RolePermissions } from '../../shared/api/types.ts'
import { ROLE_LABELS } from '../../shared/lib/labels.ts'
import { Async, ErrorNotice } from '../../shared/ui/feedback.tsx'
import { Crumbs, PageHead } from '../../shared/ui/layout.tsx'
import { useRolePermissions, useSetRolePermissions } from './api.ts'

function PermissionMatrix({ data, editable }: { data: RolePermissions; editable: boolean }) {
  const save = useSetRolePermissions()
  // Solo los roles con cambios sin guardar.
  const [draft, setDraft] = useState<Partial<Record<Role, string[]>>>({})
  const granted = (role: Role) => draft[role] ?? data.roles.find((item) => item.role === role)?.permissions ?? []
  const changed = Object.keys(draft) as Role[]

  function toggle(role: Role, permission: string, on: boolean) {
    const current = granted(role)
    setDraft({ ...draft, [role]: on ? [...current, permission] : current.filter((item) => item !== permission) })
  }

  async function saveAll() {
    for (const role of changed) {
      await save.mutateAsync({ role, permissions: draft[role] ?? [] })
    }
    setDraft({})
  }

  return (
    <>
      {save.error && <ErrorNotice error={save.error} />}
      <div className="surface table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th className="col-fixed">Permiso</th>
              {data.roles.map(({ role }) => (
                <th key={role}>{ROLE_LABELS[role]}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.permissions.map((permission) => (
              <tr key={permission.key}>
                <td className="col-fixed">
                  <div className="cell-stack">
                    <span>{permission.description}</span>
                    <span className="mono muted">{permission.key}</span>
                  </div>
                </td>
                {data.roles.map(({ role }) => (
                  <td key={role}>
                    <label className="check">
                      <input
                        type="checkbox"
                        aria-label={`${ROLE_LABELS[role]}: ${permission.description}`}
                        checked={granted(role).includes(permission.key)}
                        disabled={!editable || save.isPending}
                        onChange={(event) => toggle(role, permission.key, event.target.checked)}
                      />
                    </label>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {editable && (
        <div className="row">
          <button className="btn btn--primary" type="button" disabled={changed.length === 0 || save.isPending} onClick={() => void saveAll().catch(() => undefined)}>
            {save.isPending ? 'Guardando…' : 'Guardar cambios'}
          </button>
          <button className="btn" type="button" disabled={changed.length === 0 || save.isPending} onClick={() => setDraft({})}>
            Descartar
          </button>
          <span className="muted" aria-live="polite">
            {changed.length > 0 ? `Cambios sin guardar en: ${changed.map((role) => ROLE_LABELS[role]).join(', ')}.` : 'Sin cambios pendientes.'}
          </span>
        </div>
      )}
    </>
  )
}

export default function PermissionsPage() {
  const can = useCan()
  const permissions = useRolePermissions()
  return (
    <>
      <Crumbs items={[{ label: 'Administración' }, { label: 'Permisos por rol' }]} />
      <PageHead title="Permisos por rol" subtitle="Qué puede hacer cada rol. Los roles se asignan a las personas en Keycloak." />
      <Async query={permissions}>{(data) => <PermissionMatrix data={data} editable={can('admin.configure')} />}</Async>
    </>
  )
}
