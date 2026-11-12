// Pestañas como las del boceto: flechas para moverse y la pestaña activa en el fragmento de la URL.
import type { KeyboardEvent, ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router'

export interface TabItem {
  id: string
  label: string
  count?: ReactNode
  render: () => ReactNode
}

export function Tabs({ label, tabs }: { label: string; tabs: TabItem[] }) {
  const location = useLocation()
  const navigate = useNavigate()
  const active = tabs.find((tab) => `#${tab.id}` === location.hash) ?? tabs[0]
  if (!active) return null

  function select(tab: TabItem, focus: boolean) {
    void navigate({ search: location.search, hash: tab.id }, { replace: true, preventScrollReset: true })
    if (focus) document.getElementById(`tab-${tab.id}`)?.focus()
  }

  function onKeyDown(event: KeyboardEvent, index: number) {
    const step = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0
    const next = tabs[(index + step + tabs.length) % tabs.length]
    if (!step || !next) return
    event.preventDefault()
    select(next, true)
  }

  return (
    <section>
      <div className="tabs" role="tablist" aria-label={label}>
        {tabs.map((tab, index) => (
          <button
            key={tab.id}
            className="tab"
            role="tab"
            type="button"
            id={`tab-${tab.id}`}
            aria-controls={`p-${tab.id}`}
            aria-selected={tab === active}
            tabIndex={tab === active ? 0 : -1}
            onClick={() => select(tab, false)}
            onKeyDown={(event) => onKeyDown(event, index)}
          >
            {tab.label}
            {tab.count !== undefined && <span className="tab__count">{tab.count}</span>}
          </button>
        ))}
      </div>
      <div className="tabpanel" role="tabpanel" id={`p-${active.id}`} aria-labelledby={`tab-${active.id}`} tabIndex={0}>
        {active.render()}
      </div>
    </section>
  )
}
