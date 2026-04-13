import type { BaseComponentProps } from '@json-render/react'
import { useId, useState } from 'react'
import styles from './MessageJsonRender.module.css'
import { useMessageJsonRenderRuntime } from './messageJsonRenderRuntime'

interface AskQuestionOption {
  label: string
  value?: string
}

interface AskQuestionProps {
  confirmLabel?: string
  freeformPlaceholder?: string
  options?: AskQuestionOption[]
  question?: string
}

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message
  }
  return 'Failed to send response.'
}

function normalizeOptions(options: AskQuestionOption[] | undefined): AskQuestionOption[] {
  if (!Array.isArray(options)) {
    return []
  }

  return options.filter(
    (option): option is AskQuestionOption =>
      typeof option?.label === 'string' && option.label.trim() !== '',
  )
}

function resolveOptionValue(option: AskQuestionOption): string {
  return typeof option.value === 'string' && option.value.trim() !== '' ? option.value : option.label
}

export function MessageAskQuestion({ props }: BaseComponentProps<AskQuestionProps>) {
  const { interactionDisabled, onReplyAction } = useMessageJsonRenderRuntime()
  const [freeformValue, setFreeformValue] = useState('')
  const [pendingValue, setPendingValue] = useState<string | null>(null)
  const [submittedValue, setSubmittedValue] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const textareaId = useId()

  const question = typeof props.question === 'string' && props.question.trim() !== '' ? props.question.trim() : 'Choose a reply'
  const options = normalizeOptions(props.options)
  const freeformPlaceholder =
    typeof props.freeformPlaceholder === 'string' && props.freeformPlaceholder.trim() !== ''
      ? props.freeformPlaceholder
      : 'Write your answer'
  const confirmLabel =
    typeof props.confirmLabel === 'string' && props.confirmLabel.trim() !== '' ? props.confirmLabel.trim() : 'Confirm'
  const trimmedFreeformValue = freeformValue.trim()
  const isBusy = interactionDisabled || pendingValue !== null || submittedValue !== null

  async function sendReply(value: string): Promise<void> {
    if (!onReplyAction || isBusy) {
      return
    }

    setPendingValue(value)
    setSubmitError(null)
    try {
      await onReplyAction(value)
      setSubmittedValue(value)
      setFreeformValue('')
    } catch (error) {
      setSubmitError(resolveErrorMessage(error))
    } finally {
      setPendingValue(null)
    }
  }

  return (
    <section className={styles.askQuestionCard}>
      <p className={styles.askQuestionPrompt}>{question}</p>
      {options.length > 0 ? (
        <div className={styles.askQuestionOptions}>
          {options.map((option) => (
            <button
              className={styles.askQuestionOption}
              disabled={isBusy || !onReplyAction}
              key={`${resolveOptionValue(option)}:${option.label}`}
              onClick={() => {
                void sendReply(resolveOptionValue(option))
              }}
              type="button"
            >
              {pendingValue === resolveOptionValue(option) ? 'Sending...' : option.label}
            </button>
          ))}
        </div>
      ) : null}
      <div className={styles.askQuestionFreeform}>
        <label className={styles.askQuestionLabel} htmlFor={textareaId}>
          Your answer
        </label>
        <textarea
          className={styles.askQuestionTextarea}
          disabled={isBusy || !onReplyAction}
          id={textareaId}
          onChange={(event) => {
            setFreeformValue(event.target.value)
            if (submitError) {
              setSubmitError(null)
            }
          }}
          placeholder={freeformPlaceholder}
          rows={4}
          value={freeformValue}
        />
        <button
          className={styles.askQuestionConfirm}
          disabled={isBusy || !onReplyAction || trimmedFreeformValue === ''}
          onClick={() => {
            void sendReply(trimmedFreeformValue)
          }}
          type="button"
        >
          {pendingValue === trimmedFreeformValue && trimmedFreeformValue !== '' ? 'Sending...' : confirmLabel}
        </button>
      </div>
      {submittedValue ? <p className={styles.askQuestionStatus}>Sent response: {submittedValue}</p> : null}
      {submitError ? <p className={styles.askQuestionError}>{submitError}</p> : null}
      {!onReplyAction ? <p className={styles.askQuestionHint}>Reply controls are unavailable in this view.</p> : null}
    </section>
  )
}
