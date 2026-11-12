// Sobre qué se ejecutó un RUN o una exploración: versión del BUILD y nombre del ambiente, tal
// como quedaron en la evidencia. Si vienen los códigos del catálogo, enlazan a su ficha.
import { Link } from 'react-router'
import { routeOf } from '../lib/codes.ts'

interface Target {
  buildVersion: string
  environmentName: string
  /** Ausentes en lo registrado antes de los catálogos de BUILD y de ambientes. */
  build?: string
  environment?: string
}

function Linked({ code, children, mono }: { code: string | undefined; children: string; mono?: boolean }) {
  const route = code ? routeOf(code) : null
  if (!route) return <span className={mono ? 'mono' : undefined}>{children}</span>
  return (
    <Link className={mono ? 'mono' : undefined} to={route} title={code}>
      {children}
    </Link>
  )
}

export function BuildLink({ target }: { target: Target }) {
  return (
    <Linked code={target.build} mono>
      {target.buildVersion}
    </Linked>
  )
}

export function EnvironmentLink({ target }: { target: Target }) {
  return <Linked code={target.environment}>{target.environmentName}</Linked>
}

/** «versión · ambiente» en una línea. */
export function RunTarget({ target }: { target: Target }) {
  return (
    <span>
      <BuildLink target={target} /> · <EnvironmentLink target={target} />
    </span>
  )
}
