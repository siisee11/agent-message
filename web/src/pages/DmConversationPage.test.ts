import { describe, expect, it } from 'vitest'
import { shouldStickToBottomOnMessageAppend } from './DmConversationPage'

describe('dm conversation scroll helpers', () => {
  it('sticks when the timeline is already near the bottom', () => {
    const timeline = {
      clientHeight: 600,
      scrollHeight: 1000,
      scrollTop: 352,
    } as Pick<HTMLDivElement, 'clientHeight' | 'scrollHeight' | 'scrollTop'>

    expect(shouldStickToBottomOnMessageAppend(timeline)).toBe(true)
  })

  it('does not stick when the user has scrolled well above the bottom', () => {
    const timeline = {
      clientHeight: 600,
      scrollHeight: 1000,
      scrollTop: 200,
    } as Pick<HTMLDivElement, 'clientHeight' | 'scrollHeight' | 'scrollTop'>

    expect(shouldStickToBottomOnMessageAppend(timeline)).toBe(false)
  })

  it('defaults to stick when the timeline ref is unavailable', () => {
    expect(shouldStickToBottomOnMessageAppend(null)).toBe(true)
  })
})
