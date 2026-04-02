import type { BaseComponentProps } from '@json-render/react'
import styles from './MessageJsonRender.module.css'

interface GraphDatum {
  color?: string
  label: string
  value: number
}

interface MessageGraphProps {
  currency?: string
  data?: GraphDatum[]
  description?: string
  format?: 'currency' | 'number' | 'percent'
  maxValue?: number
  title?: string
}

const GRAPH_COLORS = ['#2563eb', '#0f766e', '#d97706', '#9333ea', '#dc2626', '#0891b2']
const GRAPH_HEIGHT = 140
const GRAPH_WIDTH = 320
const GRAPH_PADDING_X = 22
const GRAPH_PADDING_Y = 16
const LINE_GRID_LINES = 4

function isGraphDatum(value: unknown): value is GraphDatum {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  const candidate = value as Record<string, unknown>
  return typeof candidate.label === 'string' && typeof candidate.value === 'number'
}

function normalizeData(data: MessageGraphProps['data']): GraphDatum[] {
  if (!Array.isArray(data)) {
    return []
  }

  return data.flatMap((item) => {
    if (!isGraphDatum(item)) {
      return []
    }

    const label = item.label.trim()
    if (label === '' || !Number.isFinite(item.value)) {
      return []
    }

    return [
      {
        color: typeof item.color === 'string' && item.color.trim() !== '' ? item.color : undefined,
        label,
        value: Math.max(0, item.value),
      },
    ]
  })
}

function resolveGraphColor(item: GraphDatum, index: number): string {
  return item.color ?? GRAPH_COLORS[index % GRAPH_COLORS.length]
}

function resolveMaxValue(data: GraphDatum[], maxValue: number | undefined): number {
  const normalizedMax = typeof maxValue === 'number' && Number.isFinite(maxValue) && maxValue > 0 ? maxValue : 0
  const inferredMax = data.reduce((currentMax, item) => Math.max(currentMax, item.value), 0)
  return Math.max(normalizedMax, inferredMax, 1)
}

function resolveText(value: string | undefined): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function formatValue(value: number, props: MessageGraphProps): string {
  switch (props.format) {
    case 'currency':
      return new Intl.NumberFormat('en-US', {
        currency: props.currency ?? 'USD',
        maximumFractionDigits: value >= 100 ? 0 : 2,
        style: 'currency',
      }).format(value)
    case 'percent':
      return `${Number.isInteger(value) ? value : value.toFixed(1)}%`
    default:
      return new Intl.NumberFormat('en-US', {
        maximumFractionDigits: value >= 100 ? 0 : 2,
      }).format(value)
  }
}

function buildAriaLabel(kind: 'bar' | 'line', title: string | null, data: GraphDatum[], props: MessageGraphProps): string {
  const name = title ?? (kind === 'bar' ? 'Bar graph' : 'Line graph')
  const points = data.map((item) => `${item.label}: ${formatValue(item.value, props)}`).join(', ')
  return `${name}. ${points}`
}

function renderHeader(title: string | null, description: string | null) {
  if (!title && !description) {
    return null
  }

  return (
    <div className={styles.graphHeader}>
      {title ? <p className={styles.graphTitle}>{title}</p> : null}
      {description ? <p className={styles.graphDescription}>{description}</p> : null}
    </div>
  )
}

function renderEmptyState() {
  return <p className={styles.graphEmpty}>No graph data provided.</p>
}

export function MessageBarGraph({ props }: BaseComponentProps<MessageGraphProps>) {
  const title = resolveText(props.title)
  const description = resolveText(props.description)
  const data = normalizeData(props.data)

  if (data.length === 0) {
    return <section className={styles.graphCard}>{renderHeader(title, description)}{renderEmptyState()}</section>
  }

  const maxValue = resolveMaxValue(data, props.maxValue)

  return (
    <section className={styles.graphCard}>
      {renderHeader(title, description)}
      <div aria-label={buildAriaLabel('bar', title, data, props)} className={styles.barGraph} role="img">
        <div className={styles.barGraphPlot}>
          {data.map((item, index) => {
            const height = `${Math.max((item.value / maxValue) * 100, 6)}%`
            return (
              <div className={styles.barGraphColumn} key={`${item.label}:${index}`}>
                <span className={styles.graphValue}>{formatValue(item.value, props)}</span>
                <div className={styles.barGraphTrack}>
                  <div
                    className={styles.barGraphBar}
                    style={{ background: resolveGraphColor(item, index), height }}
                  />
                </div>
                <span className={styles.graphLabel}>{item.label}</span>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}

export function MessageLineGraph({ props }: BaseComponentProps<MessageGraphProps>) {
  const title = resolveText(props.title)
  const description = resolveText(props.description)
  const data = normalizeData(props.data)

  if (data.length === 0) {
    return <section className={styles.graphCard}>{renderHeader(title, description)}{renderEmptyState()}</section>
  }

  const maxValue = resolveMaxValue(data, props.maxValue)
  const usableWidth = GRAPH_WIDTH - GRAPH_PADDING_X * 2
  const usableHeight = GRAPH_HEIGHT - GRAPH_PADDING_Y * 2
  const stepX = data.length > 1 ? usableWidth / (data.length - 1) : 0

  const points = data
    .map((item, index) => {
      const x = GRAPH_PADDING_X + stepX * index
      const y = GRAPH_HEIGHT - GRAPH_PADDING_Y - (item.value / maxValue) * usableHeight
      return `${x},${y}`
    })
    .join(' ')

  return (
    <section className={styles.graphCard}>
      {renderHeader(title, description)}
      <div aria-label={buildAriaLabel('line', title, data, props)} className={styles.lineGraph} role="img">
        <svg
          aria-hidden="true"
          className={styles.lineGraphSvg}
          viewBox={`0 0 ${GRAPH_WIDTH} ${GRAPH_HEIGHT}`}
          xmlns="http://www.w3.org/2000/svg"
        >
          {Array.from({ length: LINE_GRID_LINES + 1 }, (_, index) => {
            const y = GRAPH_PADDING_Y + (usableHeight / LINE_GRID_LINES) * index
            return <line className={styles.lineGraphGrid} key={index} x1={GRAPH_PADDING_X} x2={GRAPH_WIDTH - GRAPH_PADDING_X} y1={y} y2={y} />
          })}
          <polyline className={styles.lineGraphPath} points={points} />
          {data.map((item, index) => {
            const x = GRAPH_PADDING_X + stepX * index
            const y = GRAPH_HEIGHT - GRAPH_PADDING_Y - (item.value / maxValue) * usableHeight
            return (
              <g key={`${item.label}:${index}`}>
                <circle
                  className={styles.lineGraphPoint}
                  cx={x}
                  cy={y}
                  r="4"
                  style={{ fill: resolveGraphColor(item, index) }}
                />
              </g>
            )
          })}
        </svg>
        <div className={styles.lineGraphLegend}>
          {data.map((item, index) => (
            <div className={styles.lineGraphLegendItem} key={`${item.label}:${index}`}>
              <span
                aria-hidden="true"
                className={styles.lineGraphLegendSwatch}
                style={{ background: resolveGraphColor(item, index) }}
              />
              <span className={styles.graphLabel}>{item.label}</span>
              <span className={styles.graphValue}>{formatValue(item.value, props)}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
