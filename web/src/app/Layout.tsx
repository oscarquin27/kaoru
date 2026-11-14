// Armazón común: barra lateral, migas de pan, herramientas, pie, búsqueda global y panel lateral.
import { Suspense, useState } from 'react'
import { Link, Outlet, matchPath, useLocation } from 'react-router'
import { useMe } from '../shared/api/session.ts'
import { ROLE_LABELS } from '../shared/lib/labels.ts'
import { PageSkeleton } from '../shared/ui/feedback.tsx'
import { Icon } from '../shared/ui/Icon.tsx'
import { Kana } from '../shared/ui/Kana.tsx'
import { CrumbsSlotContext, SubNav } from '../shared/ui/layout.tsx'
import { EntityPanel } from './EntityPanel.tsx'
import { ErrorBoundary } from './ErrorBoundary.tsx'
import { Palette } from './Palette.tsx'
import { usePreference, type Preference } from './preferences.ts'
import { ABOUT, ALL_SECTIONS, SECTIONS, type Section } from './sections.tsx'

function NavItem({ section, current }: { section: Section; current: boolean }) {
  const first = section.screens[0]
  return (
    <li>
      {first ? (
        <Link className="nav__item" to={first.path} aria-current={current ? 'page' : undefined}>
          <Icon name={section.icon} />
          <span>{section.label}</span>
        </Link>
      ) : (
        <a className="nav__item" aria-disabled="true" title="Sin pantalla en esta fase">
          <Icon name={section.icon} />
          <span>{section.label}</span>
          <span className="sr-only"> (sin pantalla en esta fase)</span>
        </a>
      )}
    </li>
  )
}

function Toggle({ preference, label, icon }: { preference: Preference; label: string; icon: 'rows' | 'layers' | 'sun' }) {
  const { on, toggle } = usePreference(preference)
  return (
    <button className="btn btn--icon" type="button" title={label} aria-label={label} aria-pressed={on} onClick={toggle}>
      <Icon name={icon} />
    </button>
  )
}

function UserMenu() {
  const me = useMe().data
  if (!me) return null
  return (
    <>
      <button className="btn" type="button" popoverTarget="user-menu" aria-haspopup="menu">
        <Icon name="user" />
        {me.displayName}
      </button>
      <div className="menu glass" id="user-menu" popover="auto">
        <div>
          <strong>{me.displayName}</strong>
          <div className="muted">{me.roles.map((role) => ROLE_LABELS[role]).join(' · ')}</div>
        </div>
        {me.accountUrl && (
          <a className="btn menu__item" href={me.accountUrl} target="_blank" rel="noreferrer">
            <Icon name="arrow-out" />
            Mi cuenta
          </a>
        )}
        <a className="btn menu__item" href={me.logoutUrl}>
          <Icon name="x" />
          Cerrar sesión
        </a>
      </div>
    </>
  )
}

export function Layout() {
  const { pathname } = useLocation()
  const [crumbsSlot, setCrumbsSlot] = useState<HTMLElement | null>(null)
  const [searching, setSearching] = useState(false)
  const current = ALL_SECTIONS.find((section) => section.screens.some((screen) => matchPath(screen.path, pathname)))
  const siblings = current?.screens.flatMap((screen) => (screen.label ? [{ to: screen.path, label: screen.label }] : [])) ?? []

  return (
    <div className="app">
      <aside className="sidebar glass">
        <Link className="brand" to="/" aria-label="Kaoru, inicio">
          <Icon name="kaoru" className="icon brand__mark" />
          <span className="brand__name">
            <Kana />
            <span className="wordmark">kaoru</span>
          </span>
        </Link>
        <nav aria-label="Áreas">
          <ul className="nav">
            {SECTIONS.map((section) => (
              <NavItem key={section.id} section={section} current={section === current} />
            ))}
          </ul>
        </nav>
        <ul className="sidebar__foot nav">
          <NavItem section={ABOUT} current={current === ABOUT} />
        </ul>
      </aside>

      <div className="main">
        <header className="topbar">
          <nav aria-label="Ubicación">
            <ol className="crumbs" ref={setCrumbsSlot} />
          </nav>
          <div className="topbar__tools">
            <button className="btn search-btn" type="button" onClick={() => setSearching(true)}>
              <Icon name="search" />
              <span>Buscar ID o texto</span>
              <span className="kbd">Ctrl K</span>
            </button>
            <Toggle preference="density" label="Densidad compacta" icon="rows" />
            <Toggle preference="glass" label="Reducir transparencia" icon="layers" />
            <Toggle preference="theme" label="Tema claro" icon="sun" />
            <UserMenu />
          </div>
        </header>

        <main className="content">
          <CrumbsSlotContext.Provider value={crumbsSlot}>
            {siblings.length > 1 && <SubNav label={current?.label ?? ''} items={siblings} />}
            <ErrorBoundary key={pathname}>
              <Suspense fallback={<PageSkeleton />}>
                <Outlet />
              </Suspense>
            </ErrorBoundary>
            <EntityPanel />
          </CrumbsSlotContext.Provider>
        </main>

        <footer className="foot">
          <span>Kaoru · Trazabilidad a prueba de errores</span>
          {import.meta.env.DEV && <span>Servidor de datos de ejemplo: no guarda cambios.</span>}
        </footer>
      </div>

      <Palette open={searching} onOpen={() => setSearching(true)} onClose={() => setSearching(false)} />
    </div>
  )
}
