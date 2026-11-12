// Renderizador mínimo de Markdown. Genera nodos de React, nunca HTML crudo: lo que el documento
// traiga como etiquetas se muestra como texto, así que no hay nada que sanear después.
// Cubre lo que usan las especificaciones: títulos, párrafos, listas, tablas, código, negrita,
// cursiva y enlaces http(s). Lo demás queda como texto.
import { Fragment, type ReactNode } from 'react'

const INLINE = /(`[^`]+`|\*\*[^*]+\*\*|\*[^*\s][^*]*\*|\[[^\]]+\]\([^)\s]+\))/g
const SAFE_LINK = /^(https?:\/\/|mailto:)/i

function inline(text: string): ReactNode {
  return text.split(INLINE).map((part, index) => {
    if (index % 2 === 0) return <Fragment key={index}>{part}</Fragment>
    if (part.startsWith('`')) return <code key={index}>{part.slice(1, -1)}</code>
    if (part.startsWith('**')) return <strong key={index}>{part.slice(2, -2)}</strong>
    if (part.startsWith('*')) return <em key={index}>{part.slice(1, -1)}</em>
    const [, label = '', href = ''] = /^\[([^\]]+)\]\(([^)\s]+)\)$/.exec(part) ?? []
    if (!SAFE_LINK.test(href)) return <Fragment key={index}>{label}</Fragment>
    return (
      <a key={index} href={href} target="_blank" rel="noopener noreferrer">
        {label}
      </a>
    )
  })
}

function cells(row: string): string[] {
  return row.trim().replace(/^\||\|$/g, '').split('|').map((cell) => cell.trim())
}

const LIST_ITEM = /^\s*([-*]|\d+\.)\s+/
const TABLE_RULE = /^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)*\|?\s*$/

export function Markdown({ source }: { source: string }) {
  const lines = source.replace(/\r\n?/g, '\n').split('\n')
  const blocks: ReactNode[] = []
  let index = 0
  const at = (position: number) => lines[position] ?? ''

  while (index < lines.length) {
    const line = at(index)
    const key = blocks.length
    if (line.trim() === '') {
      index++
    } else if (line.startsWith('```')) {
      const code: string[] = []
      index++
      while (index < lines.length && !at(index).startsWith('```')) code.push(at(index++))
      index++
      blocks.push(
        <pre key={key}>
          <code>{code.join('\n')}</code>
        </pre>,
      )
    } else if (/^#{1,6}\s/.test(line)) {
      const level = Math.min(line.indexOf(' '), 3)
      const Heading = `h${level}` as 'h1' | 'h2' | 'h3'
      blocks.push(<Heading key={key}>{inline(line.replace(/^#+\s+/, ''))}</Heading>)
      index++
    } else if (LIST_ITEM.test(line)) {
      const ordered = /^\s*\d+\./.test(line)
      const items: string[] = []
      while (index < lines.length && LIST_ITEM.test(at(index))) items.push(at(index++).replace(LIST_ITEM, ''))
      const List = ordered ? 'ol' : 'ul'
      blocks.push(
        <List key={key}>
          {items.map((item, position) => (
            <li key={position}>{inline(item)}</li>
          ))}
        </List>,
      )
    } else if (line.includes('|') && TABLE_RULE.test(at(index + 1))) {
      const head = cells(line)
      const rows: string[][] = []
      index += 2
      while (index < lines.length && at(index).includes('|')) rows.push(cells(at(index++)))
      blocks.push(
        <table key={key}>
          <thead>
            <tr>
              {head.map((cell, position) => (
                <th key={position}>{inline(cell)}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, position) => (
              <tr key={position}>
                {row.map((cell, column) => (
                  <td key={column}>{inline(cell)}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>,
      )
    } else {
      const paragraph: string[] = []
      while (index < lines.length && at(index).trim() !== '' && !/^(#{1,6}\s|```)/.test(at(index)) && !LIST_ITEM.test(at(index))) {
        paragraph.push(at(index++).trim())
      }
      blocks.push(<p key={key}>{inline(paragraph.join(' '))}</p>)
    }
  }
  return <article className="prose">{blocks}</article>
}
