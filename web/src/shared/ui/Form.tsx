// Formularios: un solo lugar pinta los `errors[]` del servidor bajo cada campo y lleva el foco
// al primero (reglas §A.16.2). Los campos no usan el aviso nativo del navegador.
import { createContext, useContext, useEffect, useId, useRef, useState, type ComponentProps, type FormEvent, type ReactNode } from 'react'
import { ApiError, toApiError, type FieldError } from '../api/errors.ts'
import { ErrorNotice } from './feedback.tsx'
import { Icon } from './Icon.tsx'

const FieldErrorsContext = createContext<FieldError[]>([])

interface FormProps {
  /** Guarda. Si lanza un ApiError, el formulario lo pinta. */
  onSubmit: (data: FormData) => Promise<unknown>
  /** Validación propia, además de los campos obligatorios. */
  validate?: (data: FormData) => FieldError[]
  onDone?: () => void
  onCancel?: () => void
  submitLabel: string
  label: string
  children: ReactNode
}

function missingRequired(form: HTMLFormElement): FieldError[] {
  const errors: FieldError[] = []
  for (const element of Array.from(form.elements)) {
    if (!(element instanceof HTMLInputElement || element instanceof HTMLSelectElement || element instanceof HTMLTextAreaElement)) continue
    const empty = element instanceof HTMLInputElement && element.type === 'file' ? !element.files?.length : element.value.trim() === ''
    if (element.required && empty) errors.push({ field: element.name, code: 'required', message: 'Este campo es obligatorio.' })
  }
  return errors
}

interface Failure {
  error: ApiError
  /** Errores de campo que no tienen dónde pintarse: van arriba, junto al mensaje general. */
  orphans: FieldError[]
}

export function Form({ onSubmit, validate, onDone, onCancel, submitLabel, label, children }: FormProps) {
  const ref = useRef<HTMLFormElement>(null)
  const [failure, setFailure] = useState<Failure | null>(null)
  const [pending, setPending] = useState(false)

  // Al guardar con errores, el foco va al primer campo marcado.
  useEffect(() => {
    if (failure) ref.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus()
  }, [failure])

  function fail(form: HTMLFormElement, error: ApiError) {
    setFailure({ error, orphans: error.fieldErrors.filter((item) => !form.elements.namedItem(item.field)) })
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    // Un formulario dentro de otro (un diálogo) no dispara el de afuera.
    event.stopPropagation()
    const form = event.currentTarget
    const data = new FormData(form)
    const local = [...missingRequired(form), ...(validate?.(data) ?? [])]
    if (local.length > 0) {
      fail(form, new ApiError(422, 'VALIDATION', 'Revisa los campos marcados.', local))
      return
    }
    setPending(true)
    try {
      await onSubmit(data)
      setFailure(null)
      onDone?.()
    } catch (caught) {
      fail(form, toApiError(caught))
    } finally {
      setPending(false)
    }
  }

  const error = failure?.error
  const fieldErrors = error?.fieldErrors ?? []
  const orphans = failure?.orphans ?? []
  const showNotice = error && (fieldErrors.length === 0 || orphans.length > 0)

  return (
    <form ref={ref} className="form" aria-label={label} noValidate onSubmit={(event) => void handleSubmit(event)}>
      {showNotice && <ErrorNotice error={new ApiError(error.status, error.code, error.message, orphans, error.traceId)} />}
      <FieldErrorsContext.Provider value={fieldErrors}>{children}</FieldErrorsContext.Provider>
      <div className="form__actions">
        {onCancel && (
          <button className="btn" type="button" onClick={onCancel}>
            Cancelar
          </button>
        )}
        <button className="btn btn--primary" type="submit" disabled={pending}>
          {pending ? 'Guardando…' : submitLabel}
        </button>
      </div>
    </form>
  )
}

interface FieldShellProps {
  name: string
  label: string
  required?: boolean
  hint?: ReactNode
}

/** Etiqueta, control y error de un campo. `control` recibe los atributos que lo enlazan. */
function FieldShell({ name, label, required, hint, control }: FieldShellProps & { control: (props: ControlProps) => ReactNode }) {
  const id = useId()
  const error = useContext(FieldErrorsContext).find((item) => item.field === name)
  const describedBy = [error && `${id}-error`, hint && `${id}-hint`].filter(Boolean).join(' ') || undefined
  return (
    <div className="field">
      <label htmlFor={id}>
        {label}
        {required && <span aria-hidden="true"> *</span>}
      </label>
      {control({ id, name, required, 'aria-invalid': error ? true : undefined, 'aria-describedby': describedBy })}
      {hint && (
        <span className="field__hint" id={`${id}-hint`}>
          {hint}
        </span>
      )}
      {error && (
        <span className="field__error" id={`${id}-error`}>
          <Icon name="triangle" />
          {error.message}
        </span>
      )}
    </div>
  )
}

interface ControlProps {
  id: string
  name: string
  required?: boolean
  'aria-invalid'?: true
  'aria-describedby'?: string
}

type Own<T> = FieldShellProps & Omit<T, keyof FieldShellProps | 'className' | 'id'>

export function TextField({ name, label, required, hint, ...rest }: Own<ComponentProps<'input'>>) {
  return <FieldShell name={name} label={label} required={required} hint={hint} control={(props) => <input className="input" {...rest} {...props} />} />
}

export function TextAreaField({ name, label, required, hint, ...rest }: Own<ComponentProps<'textarea'>>) {
  return <FieldShell name={name} label={label} required={required} hint={hint} control={(props) => <textarea className="textarea" {...rest} {...props} />} />
}

export function SelectField({ name, label, required, hint, children, ...rest }: Own<ComponentProps<'select'>>) {
  return (
    <FieldShell
      name={name}
      label={label}
      required={required}
      hint={hint}
      control={(props) => (
        <select className="select" {...rest} {...props}>
          {children}
        </select>
      )}
    />
  )
}

export function CheckField({ label, ...rest }: { label: ReactNode } & Omit<ComponentProps<'input'>, 'type' | 'className'>) {
  return (
    <label className="check">
      <input type="checkbox" {...rest} />
      {label}
    </label>
  )
}

/** Grupo de casillas con el mismo nombre; su error se pinta bajo el grupo. */
export function CheckGroupField({ name, label, choices, selected = [] }: { name: string; label: string; choices: { value: string; label: string }[]; selected?: string[] }) {
  const id = useId()
  const error = useContext(FieldErrorsContext).find((item) => item.field === name)
  return (
    <fieldset className="checks" aria-describedby={error ? `${id}-error` : undefined}>
      <legend>{label}</legend>
      {choices.map((choice, index) => (
        <CheckField
          key={choice.value}
          name={name}
          value={choice.value}
          label={choice.label}
          defaultChecked={selected.includes(choice.value)}
          aria-invalid={error && index === 0 ? true : undefined}
        />
      ))}
      {error && (
        <span className="field__error" id={`${id}-error`}>
          <Icon name="triangle" />
          {error.message}
        </span>
      )}
    </fieldset>
  )
}

/** Texto de un campo del formulario, sin espacios alrededor. */
export function text(data: FormData, name: string): string {
  const value = data.get(name)
  return typeof value === 'string' ? value.trim() : ''
}

/** Como `text`, pero un campo vacío no se envía. */
export function optional(data: FormData, name: string): string | undefined {
  return text(data, name) || undefined
}
