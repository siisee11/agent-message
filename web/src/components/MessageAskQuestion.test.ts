import { describe, expect, it } from 'vitest'
import { formatAskQuestionSubmission, normalizeAskQuestionItems } from './MessageAskQuestion'

describe('MessageAskQuestion helpers', () => {
  it('normalizes multi-question props into a paginated questionnaire', () => {
    const items = normalizeAskQuestionItems({
      freeformPlaceholder: 'Write your answer',
      questions: [
        {
          id: 'environment',
          options: [{ label: 'Staging', value: 'staging' }],
          question: 'Which environment should I use?',
        },
        {
          question: 'Anything else I should know?',
        },
      ],
    })

    expect(items).toHaveLength(2)
    expect(items[0]).toMatchObject({
      id: 'environment',
      question: 'Which environment should I use?',
    })
    expect(items[1]).toMatchObject({
      id: 'question-2',
      freeformPlaceholder: 'Write your answer',
      question: 'Anything else I should know?',
    })
  })

  it('formats questionnaire answers into a single AI reply payload', () => {
    const payload = formatAskQuestionSubmission(
      [
        { id: 'environment', question: 'Which environment should I use?' },
        { id: 'notes', question: 'Anything else I should know?' },
      ],
      {
        environment: 'staging',
        notes: 'Need EU region support',
      },
    )

    expect(payload).toBe(
      'Questionnaire responses:\n1. Which environment should I use?\nAnswer: staging\n2. Anything else I should know?\nAnswer: Need EU region support',
    )
  })
})
