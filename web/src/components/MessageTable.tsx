import type { BaseComponentProps } from '@json-render/react'
import styles from './MessageJsonRender.module.css'

interface MessageTableProps {
  caption?: string | null
  columns?: unknown
  rows?: unknown
}

function resolveText(value: unknown): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function normalizeColumns(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value.flatMap((item) => {
    const text = resolveText(item)
    return text ? [text] : []
  })
}

function normalizeRows(value: unknown): string[][] {
  if (!Array.isArray(value)) {
    return []
  }

  return value.flatMap((row) => {
    if (!Array.isArray(row)) {
      return []
    }

    return [
      row.map((cell) => {
        if (cell == null) {
          return ''
        }
        return String(cell)
      }),
    ]
  })
}

export function MessageTable({ props }: BaseComponentProps<MessageTableProps>) {
  const columns = normalizeColumns(props.columns)
  const rows = normalizeRows(props.rows)
  const caption = resolveText(props.caption)

  return (
    <section className={styles.tableShell}>
      <div className={styles.tableFrame}>
        <div data-slot="table-container">
          <table data-slot="table">
            <thead data-slot="table-header">
              <tr data-slot="table-row">
                {columns.map((column) => (
                  <th data-slot="table-head" key={column} scope="col">
                    {column}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody data-slot="table-body">
              {rows.map((row, rowIndex) => (
                <tr data-slot="table-row" key={`row:${rowIndex}`}>
                  {row.map((cell, cellIndex) => (
                    <td data-slot="table-cell" key={`cell:${rowIndex}:${cellIndex}`}>
                      {cell}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      {caption ? (
        <p className={styles.tableCaption} data-slot="table-caption">
          {caption}
        </p>
      ) : null}
    </section>
  )
}
