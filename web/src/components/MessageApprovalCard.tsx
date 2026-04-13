import type { BaseComponentProps } from '@json-render/react'
import { useState } from 'react'
import styles from './MessageJsonRender.module.css'
import { useMessageJsonRenderRuntime } from './messageJsonRenderRuntime'

interface ApprovalAction {
  label: string
  value: string
  variant?: 'primary' | 'secondary' | 'destructive'
}

interface ApprovalCardProps {
  actions?: ApprovalAction[]
  badge?: string
  details?: string[]
  replyHint?: string
  title?: string
}

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message
  }
  return 'Failed to send approval response.'
}

function normalizeActions(actions: ApprovalAction[] | undefined): ApprovalAction[] {
  if (!Array.isArray(actions)) {
    return []
  }

  return actions.filter(
    (action): action is ApprovalAction =>
      typeof action?.label === 'string' &&
      action.label.trim() !== '' &&
      typeof action.value === 'string' &&
      action.value.trim() !== '',
  )
}

function resolveActionClassName(variant: ApprovalAction['variant']): string {
  switch (variant) {
    case 'destructive':
      return styles.approvalActionDestructive
    case 'secondary':
      return styles.approvalActionSecondary
    default:
      return styles.approvalActionPrimary
  }
}

export function MessageApprovalCard({ props }: BaseComponentProps<ApprovalCardProps>) {
  const { interactionDisabled, onReplyAction } = useMessageJsonRenderRuntime()
  const [pendingValue, setPendingValue] = useState<string | null>(null)
  const [submittedValue, setSubmittedValue] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)

  const actions = normalizeActions(props.actions)
  const details = Array.isArray(props.details)
    ? props.details.filter((detail): detail is string => typeof detail === 'string' && detail.trim() !== '')
    : []
  const replyHint = typeof props.replyHint === 'string' ? props.replyHint.trim() : ''
  const title = typeof props.title === 'string' && props.title.trim() !== '' ? props.title : 'Approval requested'
  const badge = typeof props.badge === 'string' && props.badge.trim() !== '' ? props.badge : null

  const isBusy = interactionDisabled || pendingValue !== null || submittedValue !== null

  async function handleAction(value: string): Promise<void> {
    if (!onReplyAction || isBusy) {
      return
    }

    setPendingValue(value)
    setSubmitError(null)
    try {
      await onReplyAction(value)
      setSubmittedValue(value)
    } catch (error) {
      setSubmitError(resolveErrorMessage(error))
    } finally {
      setPendingValue(null)
    }
  }

  return (
    <section className={styles.approvalCard}>
      {badge ? <p className={styles.approvalBadge}>{badge}</p> : null}
      <p className={styles.approvalTitle}>{title}</p>
      {details.length > 0 ? (
        <ul className={styles.approvalDetails}>
          {details.map((detail) => (
            <li key={detail}>{detail}</li>
          ))}
        </ul>
      ) : null}
      {replyHint !== '' ? <p className={styles.approvalHint}>Fallback reply: {replyHint}</p> : null}
      {actions.length > 0 ? (
        <div className={styles.approvalActions}>
          {actions.map((action) => (
            <button
              className={`${styles.approvalAction} ${resolveActionClassName(action.variant)}`}
              disabled={isBusy || !onReplyAction}
              key={`${action.value}:${action.label}`}
              onClick={() => {
                void handleAction(action.value)
              }}
              type="button"
            >
              {pendingValue === action.value ? 'Sending...' : action.label}
            </button>
          ))}
        </div>
      ) : null}
      {submittedValue ? <p className={styles.approvalStatus}>Sent response: {submittedValue}</p> : null}
      {submitError ? <p className={styles.approvalError}>{submitError}</p> : null}
      {!onReplyAction && actions.length > 0 ? (
        <p className={styles.approvalHint}>Buttons are unavailable in this view.</p>
      ) : null}
    </section>
  )
}
