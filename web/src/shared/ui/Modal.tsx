// Diálogos sobre <dialog>: atrapan el foco, cierran con Esc y lo devuelven a quien los abrió.
import { useEffect, useId, useRef, type MouseEvent, type ReactNode, type SyntheticEvent } from 'react'
import { Icon } from './Icon.tsx'

interface ModalProps {
  /** Título visible; el ID de una ficha va con `code`. */
  title: ReactNode
  code?: boolean
  variant: 'drawer' | 'dialog'
  onClose: () => void
  children: ReactNode
}

/** Abre un <dialog> como modal al montarse; al desmontarse lo cierra y devuelve el foco. */
export function useModal() {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    const opener = document.activeElement
    dialog.showModal()
    // En un formulario, el foco empieza en su primer campo, no en el botón de cerrar.
    dialog.querySelector<HTMLElement>('input, select, textarea')?.focus()
    return () => {
      dialog.close()
      if (opener instanceof HTMLElement) opener.focus()
    }
  }, [])
  return ref
}

/** Atributos de cierre de un <dialog> modal: Esc y clic en el fondo. */
export function closeHandlers(onClose: () => void) {
  return {
    onClose: (event: SyntheticEvent<HTMLDialogElement>) => {
      // Un cierre ya atendido (el diálogo volvió a abrirse) no cuenta.
      if (!event.currentTarget.open) onClose()
    },
    onClick: (event: MouseEvent<HTMLDialogElement>) => {
      if (event.target === event.currentTarget) onClose()
    },
  }
}

function Modal({ title, code, variant, onClose, children }: ModalProps) {
  const ref = useModal()
  const titleId = useId()

  return (
    <dialog
      ref={ref}
      className={`${variant} glass`}
      aria-labelledby={titleId}
      {...closeHandlers(onClose)}
    >
      <div className="drawer__head">
        <span className={code ? 'id' : 'h2'} id={titleId}>
          {title}
        </span>
        <button className="btn btn--icon" type="button" onClick={onClose} title="Cerrar" aria-label="Cerrar">
          <Icon name="x" />
        </button>
      </div>
      <div className={variant === 'drawer' ? 'drawer__body' : 'dialog__body'}>{children}</div>
    </dialog>
  )
}

/** Panel lateral: abre una ficha sin perder la posición. */
export function Drawer(props: Omit<ModalProps, 'variant' | 'code'>) {
  return <Modal {...props} variant="drawer" code />
}

/** Diálogo centrado para formularios. */
export function Dialog(props: Omit<ModalProps, 'variant' | 'code'>) {
  return <Modal {...props} variant="dialog" />
}
