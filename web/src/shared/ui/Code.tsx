// Un ID legible: enlace a su ficha, o botón que la abre en el panel lateral.
import { Fragment } from 'react'
import { Link, useSearchParams } from 'react-router'
import { CODE_PATTERN, prefixOf, routeOf } from '../lib/codes.ts'
import { PANEL_PARAM } from '../lib/urlFilters.ts'

// Tipos que el panel lateral sabe leer de la API.
const PANEL_PREFIXES = ['REQ', 'CHG', 'DEF', 'DOC', 'UC', 'TC', 'RUN', 'EXP']

export function Code({ code, panel }: { code: string; panel?: boolean }) {
  const [, setParams] = useSearchParams()
  const route = routeOf(code)
  // Lo que el panel no sabe leer (PKG, BUILD, ENV, DEP, COMP) va siempre a su ficha.
  if (route && (!panel || !PANEL_PREFIXES.includes(prefixOf(code)))) {
    return (
      <Link className="id" to={route}>
        {code}
      </Link>
    )
  }
  if (!PANEL_PREFIXES.includes(prefixOf(code))) return <span className="id">{code}</span>

  function openPanel() {
    setParams(
      (previous) => {
        const next = new URLSearchParams(previous)
        next.set(PANEL_PARAM, code)
        return next
      },
      { preventScrollReset: true },
    )
  }
  return (
    <button className="id" type="button" onClick={openPanel} title="Ver en el panel lateral">
      {code}
    </button>
  )
}

/** Texto de la API con los IDs que mencione convertidos en enlaces. */
export function TextWithCodes({ text }: { text: string }) {
  const codes = text.match(CODE_PATTERN) ?? []
  const parts = text.split(CODE_PATTERN)
  return (
    <>
      {parts.map((part, index) => (
        <Fragment key={index}>
          {part}
          {codes[index] && <Code code={codes[index]} />}
        </Fragment>
      ))}
    </>
  )
}
