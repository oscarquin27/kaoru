import { Component, type ReactNode } from 'react'
import { ApiError } from '../shared/api/errors.ts'
import { ErrorNotice } from '../shared/ui/feedback.tsx'

const RENDER_ERROR = new ApiError(0, 'RENDER', 'Esta pantalla no se pudo mostrar. Recarga la página; si sigue igual, avisa a soporte.')

/** Una pantalla que falla al pintarse muestra el aviso dentro del armazón, no una página en blanco. */
export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  render() {
    return this.state.failed ? <ErrorNotice error={RENDER_ERROR} /> : this.props.children
  }
}
