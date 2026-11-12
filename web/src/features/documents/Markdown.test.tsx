import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import fixtures from '../../../mocks/fixtures.json' with { type: 'json' }
import { Markdown } from './Markdown.tsx'

it('renderiza el Markdown grabado: títulos, lista y negrita', () => {
  render(<Markdown source={fixtures['TEXT /api/documents/DOC-2026-0001/versions/2/content']} />)
  expect(screen.getByRole('heading', { level: 1, name: 'Límite diario en pagos móviles' })).toBeInTheDocument()
  expect(screen.getByRole('heading', { level: 2, name: 'Regla' })).toBeInTheDocument()
  expect(screen.getAllByRole('listitem')).toHaveLength(3)
  expect(screen.getByText('no').tagName).toBe('STRONG')
})

it('no deja pasar HTML ni enlaces peligrosos: salen como texto', () => {
  const { container } = render(
    <Markdown source={'<script>alert(1)</script> <img src=x onerror=alert(1)>\n\n[clic](javascript:alert(1)) y [bien](https://ejemplo.invalid/a)\n\n| a | b |\n|---|---|\n| `x` | *y* |'} />,
  )
  expect(container.querySelector('script, img')).toBeNull()
  expect(container.textContent).toContain('<script>alert(1)</script>')
  expect(screen.queryByRole('link', { name: 'clic' })).toBeNull()
  expect(screen.getByRole('link', { name: 'bien' })).toHaveAttribute('href', 'https://ejemplo.invalid/a')
  expect(screen.getByRole('link', { name: 'bien' })).toHaveAttribute('rel', 'noopener noreferrer')
  expect(screen.getByRole('table')).toBeInTheDocument()
  expect(screen.getByText('x').tagName).toBe('CODE')
})
