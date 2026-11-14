// Los iconos salen del boceto: si el boceto cambia un trazo, esta prueba avisa.
import { renderToStaticMarkup } from 'react-dom/server'
import { expect, it } from 'vitest'
import source from '../../../../tools/boceto/js/boceto.js?raw'
import { ICONS } from './Icon.tsx'

it('cada icono del boceto existe aquí con el mismo trazo', () => {
  const block = /var ICONS = \{([\s\S]*?)\n {2}\};/.exec(source)?.[1] ?? ''
  const sketch = [...block.matchAll(/^\s*'?([a-z-]+)'?: '(.*)',?$/gm)].map(([, name, paths]) => [name, paths] as const)
  expect(sketch.length).toBeGreaterThan(30)
  for (const [name, paths] of sketch) {
    const ours = ICONS[name as keyof typeof ICONS]
    // React cierra con </path>; el boceto, con />. El trazo es lo que se compara.
    const markup = renderToStaticMarkup(<>{ours}</>).replace(/><\/(path|circle|rect)>/g, '/>')
    expect(markup, name).toBe(paths)
  }
})
