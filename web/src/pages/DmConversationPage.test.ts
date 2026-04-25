import { describe, expect, it } from 'vitest'
import {
  resolveConversationShortcutIndex,
  resolvePendingTurnMessageId,
  resolvePendingTurnStatus,
  shouldStickToBottomOnMessageAppend,
} from './DmConversationPage'

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

describe('dm conversation pending turn helpers', () => {
  it('marks the latest own message as pending when no agent reply exists yet', () => {
    const messages = [
      { message: { id: 'agent-1', sender_id: 'agent', deleted: false } },
      { message: { id: 'user-1', sender_id: 'user', deleted: false } },
    ] as const

    expect(resolvePendingTurnMessageId(messages, 'user')).toBe('user-1')
  })

  it('does not mark a pending turn when the latest message is from the agent', () => {
    const messages = [
      { message: { id: 'user-1', sender_id: 'user', deleted: false } },
      { message: { id: 'agent-1', sender_id: 'agent', deleted: false } },
    ] as const

    expect(resolvePendingTurnMessageId(messages, 'user')).toBeNull()
  })

  it('does not mark deleted user messages as pending', () => {
    const messages = [{ message: { id: 'user-1', sender_id: 'user', deleted: true } }] as const

    expect(resolvePendingTurnMessageId(messages, 'user')).toBeNull()
  })

  it('resolves a working label when updates are live and the watcher is online', () => {
    expect(resolvePendingTurnStatus('open', true)).toEqual({
      label: 'agent working',
      tone: 'working',
    })
  })

  it('resolves a reconnecting label while the event stream is reconnecting', () => {
    expect(resolvePendingTurnStatus('connecting', true)).toEqual({
      label: 'reconnecting to updates',
      tone: 'connecting',
    })
  })

  it('resolves an interrupted label when the event stream closes', () => {
    expect(resolvePendingTurnStatus('closed', true)).toEqual({
      label: 'updates interrupted',
      tone: 'offline',
    })
  })
})

describe('dm conversation keyboard shortcuts', () => {
  function shortcutEvent(overrides: Partial<KeyboardEvent>): KeyboardEvent {
    return {
      altKey: false,
      ctrlKey: false,
      defaultPrevented: false,
      isComposing: false,
      key: '1',
      metaKey: true,
      shiftKey: false,
      ...overrides,
    } as KeyboardEvent
  }

  it('maps command number shortcuts to zero-based conversation indexes', () => {
    expect(resolveConversationShortcutIndex(shortcutEvent({ key: '1' }))).toBe(0)
    expect(resolveConversationShortcutIndex(shortcutEvent({ key: '9' }))).toBe(8)
  })

  it('ignores non-command and modified shortcuts', () => {
    expect(resolveConversationShortcutIndex(shortcutEvent({ metaKey: false }))).toBeNull()
    expect(resolveConversationShortcutIndex(shortcutEvent({ altKey: true }))).toBeNull()
    expect(resolveConversationShortcutIndex(shortcutEvent({ shiftKey: true }))).toBeNull()
    expect(resolveConversationShortcutIndex(shortcutEvent({ key: '0' }))).toBeNull()
  })

  it('ignores consumed and composing keyboard events', () => {
    expect(resolveConversationShortcutIndex(shortcutEvent({ defaultPrevented: true }))).toBeNull()
    expect(resolveConversationShortcutIndex(shortcutEvent({ isComposing: true }))).toBeNull()
  })
})
