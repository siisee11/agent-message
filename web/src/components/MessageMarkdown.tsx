import type { BaseComponentProps } from '@json-render/react'
import ReactMarkdown from 'react-markdown'
import styles from './MessageJsonRender.module.css'

interface MarkdownProps {
  content?: string
  markdown?: string
  text?: string
}

function resolveMarkdown(props: MarkdownProps): string {
  const candidates = [props.content, props.markdown, props.text]
  for (const candidate of candidates) {
    if (typeof candidate === 'string' && candidate.trim() !== '') {
      return candidate
    }
  }

  return ''
}

export function MessageMarkdown({ props }: BaseComponentProps<MarkdownProps>) {
  const content = resolveMarkdown(props)
  if (content === '') {
    return null
  }

  return (
    <div className={styles.markdown}>
      <ReactMarkdown>{content}</ReactMarkdown>
    </div>
  )
}
