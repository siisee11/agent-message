import type { InfiniteData } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'
import type { MessageDetails, Reaction } from '../api'
import {
  addUnreadConversation,
  addReactionToPages,
  announceRealtimeMessageWillAppend,
  REALTIME_MESSAGE_WILL_APPEND_EVENT,
  removeUnreadConversation,
  removeReactionFromPages,
  shouldMarkConversationUnread,
} from './state'

function buildMessageDetails(messageId: string): MessageDetails {
  return {
    message: {
      id: messageId,
      conversation_id: 'conversation-1',
      sender_id: 'user-1',
      content: 'hello',
      edited: false,
      deleted: false,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
    sender: {
      id: 'user-1',
      account_id: 'alice',
      username: 'alice',
      created_at: '2026-01-01T00:00:00Z',
    },
    reactions: [],
  }
}

function buildReaction(overrides: Partial<Reaction> = {}): Reaction {
  return {
    id: 'reaction-1',
    message_id: 'message-1',
    user_id: 'user-2',
    emoji: '👍',
    created_at: '2026-01-01T00:00:01Z',
    ...overrides,
  }
}

function buildPages(): InfiniteData<MessageDetails[]> {
  return {
    pageParams: [undefined],
    pages: [[buildMessageDetails('message-1'), buildMessageDetails('message-2')]],
  }
}

describe('realtime reaction page updates', () => {
  it('adds reactions into cached message pages', () => {
    const updated = addReactionToPages(buildPages(), buildReaction())

    expect(updated?.pages[0][0].reactions).toEqual([buildReaction()])
    expect(updated?.pages[0][1].reactions).toEqual([])
  })

  it('removes reactions from cached message pages', () => {
    const current = addReactionToPages(buildPages(), buildReaction())
    const updated = removeReactionFromPages(current, {
      message_id: 'message-1',
      emoji: '👍',
      user_id: 'user-2',
    })

    expect(updated?.pages[0][0].reactions).toEqual([])
  })
})

describe('realtime message append event', () => {
  it('dispatches the active conversation id before appending a realtime message', () => {
    let receivedConversationId: string | null = null
    const originalWindow = Reflect.get(globalThis, 'window')
    const windowStub = new EventTarget()
    const handleEvent = (event: Event) => {
      receivedConversationId = (event as CustomEvent<{ conversationId: string }>).detail.conversationId
    }

    Reflect.set(globalThis, 'window', windowStub)
    windowStub.addEventListener(REALTIME_MESSAGE_WILL_APPEND_EVENT, handleEvent)
    try {
      announceRealtimeMessageWillAppend('conversation-42')
    } finally {
      windowStub.removeEventListener(REALTIME_MESSAGE_WILL_APPEND_EVENT, handleEvent)
      if (originalWindow === undefined) {
        Reflect.deleteProperty(globalThis, 'window')
      } else {
        Reflect.set(globalThis, 'window', originalWindow)
      }
    }

    expect(receivedConversationId).toBe('conversation-42')
  })
})

describe('realtime unread conversation helpers', () => {
  it('marks only incoming messages outside the active conversation as unread', () => {
    const incomingMessage = { ...buildMessageDetails('message-3').message, sender_id: 'user-2' }
    const ownMessage = { ...incomingMessage, sender_id: 'user-1' }

    expect(shouldMarkConversationUnread(incomingMessage, 'user-1', null)).toBe(true)
    expect(shouldMarkConversationUnread(incomingMessage, 'user-1', 'conversation-1')).toBe(false)
    expect(shouldMarkConversationUnread(ownMessage, 'user-1', null)).toBe(false)
  })

  it('adds and removes unread conversation ids without mutating the original set', () => {
    const current = new Set<string>(['conversation-1'])
    const added = addUnreadConversation(current, 'conversation-2')
    const removed = removeUnreadConversation(added, 'conversation-1')

    expect(Array.from(current)).toEqual(['conversation-1'])
    expect(Array.from(added)).toEqual(['conversation-1', 'conversation-2'])
    expect(Array.from(removed)).toEqual(['conversation-2'])
  })
})
