import type { BaseComponentProps } from '@json-render/react'
import { useId, useState } from 'react'
import styles from './MessageJsonRender.module.css'
import { useMessageJsonRenderRuntime } from './messageJsonRenderRuntime'

interface AskQuestionOption {
  label: string
  value?: string
}

interface AskQuestionItem {
  freeformPlaceholder?: string
  id?: string
  options?: AskQuestionOption[]
  question?: string
}

interface AskQuestionProps {
  backLabel?: string
  confirmLabel?: string
  freeformPlaceholder?: string
  nextLabel?: string
  options?: AskQuestionOption[]
  question?: string
  questions?: AskQuestionItem[]
}

interface NormalizedAskQuestionItem {
  freeformPlaceholder: string
  id: string
  options: AskQuestionOption[]
  question: string
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

function resolveQuestionText(value: string | undefined): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function resolveFreeformPlaceholder(value: string | undefined, fallback: string): string {
  const text = resolveQuestionText(value)
  return text ?? fallback
}

export function normalizeAskQuestionItems(props: AskQuestionProps): NormalizedAskQuestionItem[] {
  const defaultPlaceholder = resolveFreeformPlaceholder(props.freeformPlaceholder, 'Write your answer')
  const normalizedQuestions = Array.isArray(props.questions)
    ? props.questions.flatMap((item, index) => {
        const question = resolveQuestionText(item?.question)
        if (!question) {
          return []
        }

        return [
          {
            freeformPlaceholder: resolveFreeformPlaceholder(item.freeformPlaceholder, defaultPlaceholder),
            id:
              typeof item.id === 'string' && item.id.trim() !== ''
                ? item.id.trim()
                : `question-${index + 1}`,
            options: normalizeOptions(item.options),
            question,
          },
        ]
      })
    : []

  if (normalizedQuestions.length > 0) {
    return normalizedQuestions
  }

  return [
    {
      freeformPlaceholder: defaultPlaceholder,
      id: 'question-1',
      options: normalizeOptions(props.options),
      question: resolveQuestionText(props.question) ?? 'Choose a reply',
    },
  ]
}

export function formatAskQuestionSubmission(
  items: Array<Pick<NormalizedAskQuestionItem, 'id' | 'question'>>,
  answers: Record<string, string>,
): string {
  const lines = ['Questionnaire responses:']

  items.forEach((item, index) => {
    const answer = answers[item.id]?.trim()
    if (!answer) {
      return
    }

    lines.push(`${index + 1}. ${item.question}`)
    lines.push(`Answer: ${answer}`)
  })

  return lines.join('\n')
}

function LegacyAskQuestion({ props }: { props: AskQuestionProps }) {
  const { interactionDisabled, onReplyAction } = useMessageJsonRenderRuntime()
  const [freeformValue, setFreeformValue] = useState('')
  const [pendingValue, setPendingValue] = useState<string | null>(null)
  const [submittedValue, setSubmittedValue] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const textareaId = useId()

  const question = resolveQuestionText(props.question) ?? 'Choose a reply'
  const options = normalizeOptions(props.options)
  const freeformPlaceholder = resolveFreeformPlaceholder(props.freeformPlaceholder, 'Write your answer')
  const confirmLabel = resolveQuestionText(props.confirmLabel) ?? 'Confirm'
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

function MultiStepAskQuestion({ props }: { props: AskQuestionProps }) {
  const { interactionDisabled, onReplyAction } = useMessageJsonRenderRuntime()
  const [answers, setAnswers] = useState<Record<string, string>>({})
  const [currentIndex, setCurrentIndex] = useState(0)
  const [pendingSubmission, setPendingSubmission] = useState<string | null>(null)
  const [submittedPayload, setSubmittedPayload] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const textareaId = useId()

  const items = normalizeAskQuestionItems(props)
  const currentItem = items[Math.min(currentIndex, items.length - 1)]
  const currentAnswer = currentItem ? answers[currentItem.id] ?? '' : ''
  const trimmedCurrentAnswer = currentAnswer.trim()
  const confirmLabel = resolveQuestionText(props.confirmLabel) ?? 'Submit'
  const nextLabel = resolveQuestionText(props.nextLabel) ?? 'Next'
  const backLabel = resolveQuestionText(props.backLabel) ?? 'Back'
  const isBusy = interactionDisabled || pendingSubmission !== null || submittedPayload !== null
  const isLastStep = currentIndex >= items.length - 1

  function updateAnswer(questionId: string, value: string): void {
    setAnswers((current) => ({
      ...current,
      [questionId]: value,
    }))
  }

  async function submitAllAnswers(): Promise<void> {
    if (!onReplyAction || isBusy || !isLastStep || trimmedCurrentAnswer === '') {
      return
    }

    const nextAnswers = {
      ...answers,
      [currentItem.id]: trimmedCurrentAnswer,
    }
    const payload = formatAskQuestionSubmission(items, nextAnswers)

    setPendingSubmission(payload)
    setSubmitError(null)
    try {
      await onReplyAction(payload)
      setSubmittedPayload(payload)
    } catch (error) {
      setSubmitError(resolveErrorMessage(error))
    } finally {
      setPendingSubmission(null)
    }
  }

  if (!currentItem) {
    return null
  }

  return (
    <section className={styles.askQuestionCard}>
      <p className={styles.askQuestionProgress}>
        Question {currentIndex + 1} of {items.length}
      </p>
      <p className={styles.askQuestionPrompt}>{currentItem.question}</p>
      {currentItem.options.length > 0 ? (
        <div className={styles.askQuestionOptions}>
          {currentItem.options.map((option) => {
            const optionValue = resolveOptionValue(option)
            const isSelected = trimmedCurrentAnswer !== '' && trimmedCurrentAnswer === optionValue

            return (
              <button
                className={`${styles.askQuestionOption}${isSelected ? ` ${styles.askQuestionOptionSelected}` : ''}`}
                disabled={isBusy || !onReplyAction}
                key={`${currentItem.id}:${optionValue}:${option.label}`}
                onClick={() => {
                  updateAnswer(currentItem.id, optionValue)
                  if (submitError) {
                    setSubmitError(null)
                  }
                }}
                type="button"
              >
                {option.label}
              </button>
            )
          })}
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
            updateAnswer(currentItem.id, event.target.value)
            if (submitError) {
              setSubmitError(null)
            }
          }}
          placeholder={currentItem.freeformPlaceholder}
          rows={4}
          value={currentAnswer}
        />
      </div>
      <div className={styles.askQuestionNav}>
        <button
          className={`${styles.askQuestionOption} ${styles.askQuestionNavSecondary}`}
          disabled={isBusy || currentIndex === 0}
          onClick={() => {
            setCurrentIndex((index) => Math.max(index - 1, 0))
          }}
          type="button"
        >
          {backLabel}
        </button>
        {!isLastStep ? (
          <button
            className={styles.askQuestionConfirm}
            disabled={isBusy || !onReplyAction || trimmedCurrentAnswer === ''}
            onClick={() => {
              setCurrentIndex((index) => Math.min(index + 1, items.length - 1))
            }}
            type="button"
          >
            {nextLabel}
          </button>
        ) : (
          <button
            className={styles.askQuestionConfirm}
            disabled={isBusy || !onReplyAction || trimmedCurrentAnswer === ''}
            onClick={() => {
              void submitAllAnswers()
            }}
            type="button"
          >
            {pendingSubmission ? 'Sending...' : confirmLabel}
          </button>
        )}
      </div>
      {submittedPayload ? (
        <p className={styles.askQuestionStatus}>Sent responses for {items.length} questions.</p>
      ) : null}
      {submitError ? <p className={styles.askQuestionError}>{submitError}</p> : null}
      {!onReplyAction ? <p className={styles.askQuestionHint}>Reply controls are unavailable in this view.</p> : null}
    </section>
  )
}

export function MessageAskQuestion({ props }: BaseComponentProps<AskQuestionProps>) {
  const usesMultiStepFlow =
    Array.isArray(props.questions) &&
    props.questions.some((item) => typeof item?.question === 'string' && item.question.trim() !== '')

  if (usesMultiStepFlow) {
    return <MultiStepAskQuestion props={props} />
  }

  return <LegacyAskQuestion props={props} />
}
