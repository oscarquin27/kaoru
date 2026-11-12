import { useEffect, useState } from 'react'

/** El valor, pero solo después de que deja de cambiar durante `delay` milisegundos. */
export function useDebounced<T>(value: T, delay: number): T {
  const [settled, setSettled] = useState(value)
  useEffect(() => {
    const timer = setTimeout(() => setSettled(value), delay)
    return () => clearTimeout(timer)
  }, [value, delay])
  return settled
}
