import type { InfiniteData } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'
import type { MessageDetails, Reaction } from '../api'
import { addReactionToPages, removeReactionFromPages } from './state'

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
