import { JSONUIProvider, Renderer, type ComponentRegistry } from '@json-render/react'
import type { Spec } from '@json-render/core'
import type { JsonRenderSpec } from '../api'
import styles from './MessageJsonRender.module.css'

interface MessageJsonRenderProps {
  spec: JsonRenderSpec | null
}

interface BareUIElement {
  type: string
  props?: Record<string, unknown>
  children?: string[]
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function toSpec(value: JsonRenderSpec | null): Spec | null {
  if (!isObject(value)) {
    return null
  }

  const root = value.root
  const elements = value.elements
  if (typeof root !== 'string' || root.trim() === '' || !isObject(elements)) {
    return null
  }

  const normalizedElements: Spec['elements'] = {}
  for (const [key, rawElement] of Object.entries(elements)) {
    if (!isObject(rawElement) || typeof rawElement.type !== 'string' || rawElement.type.trim() === '') {
      return null
    }

    const normalizedElement: Spec['elements'][string] = {
      ...rawElement,
      type: rawElement.type,
      props: isObject(rawElement.props) ? rawElement.props : {},
    }

    if (Array.isArray(rawElement.children)) {
      normalizedElement.children = rawElement.children.filter(
        (child): child is string => typeof child === 'string' && child.trim() !== '',
      )
    }

    normalizedElements[key] = normalizedElement
  }

  return {
    ...value,
    root,
    elements: normalizedElements,
    state: isObject(value.state) ? value.state : undefined,
  } as Spec
}

const messageJsonRenderRegistry: ComponentRegistry = {
  Stack: ({ children }) => <div className={styles.stack}>{children}</div>,
  Text: ({ element }) => {
    const text = typeof element.props?.text === 'string' ? element.props.text : ''
    if (text.trim() === '') {
      return null
    }
    return <p className={styles.text}>{text}</p>
  },
  Badge: ({ element }) => {
    const label = typeof element.props?.label === 'string' ? element.props.label : ''
    if (label.trim() === '') {
      return null
    }
    return <span className={styles.badge}>{label}</span>
  },
}

const messageJsonRenderFallback: ComponentRegistry[string] = ({ element }) => {
  const typedElement = element as BareUIElement
  return <p className={styles.fallback}>Unsupported component: {typedElement.type}</p>
}

export function MessageJsonRender({ spec }: MessageJsonRenderProps) {
  const parsedSpec = toSpec(spec)
  if (!parsedSpec) {
    return <p className={styles.fallback}>[json-render message]</p>
  }

  return (
    <div className={styles.root}>
      <JSONUIProvider initialState={parsedSpec.state} registry={messageJsonRenderRegistry}>
        <Renderer fallback={messageJsonRenderFallback} registry={messageJsonRenderRegistry} spec={parsedSpec} />
      </JSONUIProvider>
    </div>
  )
}
