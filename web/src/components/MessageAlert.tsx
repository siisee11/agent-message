import type { BaseComponentProps } from '@json-render/react'
import styles from './MessageJsonRender.module.css'

type AlertType = 'error' | 'info' | 'success' | 'warning'

interface MessageAlertProps {
  message?: string | null
  title?: string | null
  type?: AlertType | null
}

function resolveText(value: string | null | undefined): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function resolveAlertClassName(type: AlertType | null | undefined): string {
  switch (type) {
    case 'success':
      return styles.alertSuccess
    case 'warning':
      return styles.alertWarning
    case 'error':
      return styles.alertError
    case 'info':
    default:
      return styles.alertInfo
  }
}

export function MessageAlert({ props }: BaseComponentProps<MessageAlertProps>) {
  const title = resolveText(props.title)
  const message = resolveText(props.message)

  if (!title && !message) {
    return null
  }

  return (
    <section className={`${styles.alertCard} ${resolveAlertClassName(props.type)}`} role="alert">
      {title ? <p className={styles.alertTitle}>{title}</p> : null}
      {message ? <p className={styles.alertMessage}>{message}</p> : null}
    </section>
  )
}
