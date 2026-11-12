// Los filtros de las listas viven en la URL: el enlace se puede compartir.
import { useSearchParams } from 'react-router'

export function useUrlFilters() {
  const [params, setParams] = useSearchParams()

  /** Cambia un filtro; vacío lo quita. Cualquier cambio vuelve a la primera página. */
  function setFilter(name: string, value: string) {
    setParams(
      (previous) => {
        const next = new URLSearchParams(previous)
        if (value) next.set(name, value)
        else next.delete(name)
        if (name !== 'page') next.delete('page')
        return next
      },
      { replace: true },
    )
  }

  /** Los filtros pedidos, sin los vacíos, listos para la consulta a la API. */
  function pick(names: string[]): Record<string, string | undefined> {
    return Object.fromEntries(names.map((name) => [name, params.get(name) || undefined]))
  }

  return { params, setFilter, pick }
}

/** Parte de una URL que abre la ficha de un ID en el panel lateral sin salir de la pantalla. */
export const PANEL_PARAM = 'ficha'
