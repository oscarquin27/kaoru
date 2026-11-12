// Piezas de página del boceto: migas de pan, encabezado y navegación secundaria.
import { createContext, useContext, useEffect, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { Link, NavLink } from 'react-router'

/** Lugar de la barra superior donde cada pantalla pinta sus migas. Lo provee el armazón. */
export const CrumbsSlotContext = createContext<HTMLElement | null>(null)

export interface Crumb {
  label: string
  to?: string
  /** Un ID legible va en monoespaciada. */
  code?: boolean
}

/** Migas de pan de la pantalla. La última es la página actual y da el título de la pestaña. */
export function Crumbs({ items }: { items: Crumb[] }) {
  const slot = useContext(CrumbsSlotContext)
  const current = items.at(-1)?.label
  useEffect(() => {
    if (current) document.title = `${current} · Kaoru`
  }, [current])
  if (!slot) return null
  return createPortal(
    items.map((item, index) => {
      const last = index === items.length - 1
      const className = item.code ? 'id' : undefined
      return (
        <li key={item.label}>
          {item.to && !last ? (
            <Link className={className} to={item.to}>
              {item.label}
            </Link>
          ) : (
            <span className={className} aria-current={last ? 'page' : undefined}>
              {item.label}
            </span>
          )}
        </li>
      )
    }),
    slot,
  )
}

export function PageHead({ title, subtitle, code, actions }: { title: string; subtitle?: ReactNode; code?: string; actions?: ReactNode }) {
  return (
    <div className="page-head">
      <div className="stack-s">
        {code && <span className="id">{code}</span>}
        <h1 className="title">{title}</h1>
        {subtitle && <p className="subtitle">{subtitle}</p>}
      </div>
      {actions && <div className="page-head__actions">{actions}</div>}
    </div>
  )
}

/** Pantallas hermanas dentro de un área (por ejemplo, Pruebas: matriz y casos de prueba). */
export function SubNav({ label, items }: { label: string; items: { to: string; label: string }[] }) {
  return (
    <nav className="tabs" aria-label={label}>
      {items.map((item) => (
        <NavLink key={item.to} className="tab" to={item.to} end>
          {item.label}
        </NavLink>
      ))}
    </nav>
  )
}

export function Meter({ percent, grow }: { percent: number; grow?: boolean }) {
  return (
    <span className="meter" role="img" aria-label={`${percent} %`} style={grow ? { flex: 1 } : undefined}>
      <span className="meter__fill" style={{ width: `${percent}%` }} />
    </span>
  )
}

/** Un dato con su etiqueta, dentro de un <dl class="meta">. */
export function MetaItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="meta__item">
      <dt>{label}</dt>
      <dd>{children}</dd>
    </div>
  )
}
