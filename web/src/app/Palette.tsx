// Búsqueda global (Ctrl+K): un ID o un texto lleva a su ficha. Incluye acciones rápidas.
import { useEffect, useState, type KeyboardEvent } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { useApiQuery } from '../shared/api/query.ts'
import { useCan } from '../shared/api/session.ts'
import type { Items, SearchHit } from '../shared/api/types.ts'
import { routeOf } from '../shared/lib/codes.ts'
import { ENTITY_LABELS } from '../shared/lib/labels.ts'
import { PANEL_PARAM } from '../shared/lib/urlFilters.ts'
import { useDebounced } from '../shared/lib/useDebounced.ts'
import { Icon, type IconName } from '../shared/ui/Icon.tsx'
import { closeHandlers, useModal } from '../shared/ui/Modal.tsx'

interface Entry {
  group: 'Ir a' | 'Acciones'
  label: string
  note: string
  icon: IconName
  to: string
  code?: boolean
}

const HIT_ICONS: Record<SearchHit['type'], IconName> = {
  requirement: 'file',
  change: 'swap',
  risk_assessment: 'triangle',
  functional_case: 'flask',
  test_case: 'flask',
  test_run: 'flask',
  exploratory_session: 'flask',
  defect: 'bug',
  document: 'file',
  component: 'layers',
  build: 'box',
  package: 'box',
  environment: 'server',
  deployment: 'server',
  checklist: 'flag',
  diagnostic: 'flag',
  smoke_test: 'flag',
  report: 'chart',
}

const ACTIONS: (Entry & { permission?: string })[] = [
  { group: 'Acciones', label: 'Nuevo DEF', note: 'Abrir un defecto', icon: 'plus', to: '/defectos?nuevo=1', permission: 'def.create' },
  { group: 'Acciones', label: 'Ir a matriz', note: 'Matriz de trazabilidad', icon: 'grid', to: '/pruebas' },
  { group: 'Acciones', label: 'Ir a paquetes', note: 'Paquetes y regresión', icon: 'box', to: '/paquetes' },
  { group: 'Acciones', label: 'Ir a ambientes', note: 'Ambientes y despliegues', icon: 'server', to: '/ambientes' },
  { group: 'Acciones', label: 'Ir a producción', note: 'Checklists, diagnósticos y pruebas de humo', icon: 'flag', to: '/produccion' },
  { group: 'Acciones', label: 'Ir a informes', note: 'Informes de conformidad', icon: 'chart', to: '/informes' },
  { group: 'Acciones', label: 'Ir a calendario', note: 'Hitos, ventanas y atrasos', icon: 'calendar', to: '/calendario' },
  { group: 'Acciones', label: 'Ir a inicio', note: 'Tablero', icon: 'house', to: '/' },
]

function PaletteDialog({ onClose }: { onClose: () => void }) {
  const ref = useModal()
  const navigate = useNavigate()
  const location = useLocation()
  const can = useCan()
  const [text, setText] = useState('')
  const [active, setActive] = useState(0)
  const query = useDebounced(text.trim(), 200)
  const search = useApiQuery<Items<SearchHit>>('/api/search', { q: query }, query.length >= 2)

  /** Un ID con pantalla propia va a su ruta; el resto se abre en el panel lateral, aquí mismo. */
  function hitTarget(code: string): string {
    const params = new URLSearchParams(location.search)
    params.set(PANEL_PARAM, code)
    return routeOf(code) ?? `${location.pathname}?${params.toString()}`
  }

  const hits: Entry[] = (query.length >= 2 ? (search.data?.items ?? []) : []).map((hit) => ({
    group: 'Ir a',
    label: hit.code,
    note: `${ENTITY_LABELS[hit.type]} · ${hit.title}`,
    icon: HIT_ICONS[hit.type],
    to: hitTarget(hit.code),
    code: true,
  }))
  const actions = ACTIONS.filter((action) => !action.permission || can(action.permission)).filter(
    (action) => !text.trim() || `${action.label} ${action.note}`.toLowerCase().includes(text.trim().toLowerCase()),
  )
  const shown = [...hits, ...actions]
  const selected = Math.min(active, Math.max(shown.length - 1, 0))

  function go(entry: Entry) {
    onClose()
    void navigate(entry.to)
  }

  function onKeyDown(event: KeyboardEvent) {
    if ((event.key === 'ArrowDown' || event.key === 'ArrowUp') && shown.length > 0) {
      event.preventDefault()
      const next = (selected + (event.key === 'ArrowDown' ? 1 : -1) + shown.length) % shown.length
      setActive(next)
      document.getElementById(`palette-opt-${next}`)?.scrollIntoView?.({ block: 'nearest' })
    } else if (event.key === 'Enter' && shown[selected]) {
      event.preventDefault()
      go(shown[selected])
    }
  }

  const waiting = query.length >= 2 && (search.isLoading || query !== text.trim())
  return (
    <dialog ref={ref} className="palette glass" aria-label="Búsqueda global" {...closeHandlers(onClose)}>
      <input
        className="palette__input"
        type="text"
        role="combobox"
        aria-expanded="true"
        aria-controls="palette-list"
        aria-activedescendant={shown.length > 0 ? `palette-opt-${selected}` : undefined}
        aria-label="Buscar ID o texto"
        placeholder="Escribe un ID o un texto"
        autoComplete="off"
        spellCheck={false}
        value={text}
        onChange={(event) => {
          setText(event.target.value)
          setActive(0)
        }}
        onKeyDown={onKeyDown}
      />
      <div className="palette__list" id="palette-list" role="listbox" aria-label="Resultados">
        {search.error && query.length >= 2 && (
          <p className="palette__empty" role="alert">
            {search.error.message}
          </p>
        )}
        {waiting && !search.error && <p className="palette__empty">Buscando…</p>}
        {!waiting && !search.error && shown.length === 0 && (
          <p className="palette__empty">Sin resultados. Prueba con un ID completo, por ejemplo REQ-2026-0042.</p>
        )}
        {shown.map((entry, index) => (
          <div key={entry.group + entry.label}>
            {entry.group !== shown[index - 1]?.group && <p className="palette__group h3">{entry.group}</p>}
            <a
              className="palette__item"
              role="option"
              id={`palette-opt-${index}`}
              aria-selected={index === selected}
              href={entry.to}
              onClick={(event) => {
                event.preventDefault()
                go(entry)
              }}
            >
              <Icon name={entry.icon} />
              <span className={entry.code ? 'id' : undefined}>{entry.label}</span>
              <span className="muted">{entry.note}</span>
            </a>
          </div>
        ))}
      </div>
      <div className="palette__foot">
        <span>
          <span className="kbd">↑↓</span> mover
        </span>
        <span>
          <span className="kbd">Enter</span> abrir
        </span>
        <span>
          <span className="kbd">Esc</span> cerrar
        </span>
      </div>
    </dialog>
  )
}

export function Palette({ open, onOpen, onClose }: { open: boolean; onOpen: () => void; onClose: () => void }) {
  useEffect(() => {
    function onKeyDown(event: globalThis.KeyboardEvent) {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        onOpen()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [onOpen])
  return open ? <PaletteDialog onClose={onClose} /> : null
}
