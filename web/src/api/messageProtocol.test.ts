import { describe, expect, it } from 'vitest'
import type { Message } from './types'
import { parseMessageContent, resolveMessageKind, resolveJsonRenderSpec } from './messageProtocol'

function createMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: 'm1',
    conversation_id: 'c1',
    sender_id: 'u1',
    content: undefined,
    edited: false,
    deleted: false,
    created_at: '2026-03-30T00:00:00.000Z',
    updated_at: '2026-03-30T00:00:00.000Z',
    ...overrides,
  }
}

describe('message protocol parsing', () => {
  it('defaults unknown or missing kind to text', () => {
    const legacyMessage = createMessage({ content: 'hello' })
    expect(resolveMessageKind(legacyMessage)).toBe('text')

    const unknownKindMessage = createMessage({ kind: 'other' as Message['kind'] })
    expect(resolveMessageKind(unknownKindMessage)).toBe('text')
  })

  it('accepts both kind and message_kind for json_render', () => {
    const kindMessage = createMessage({ kind: 'json_render' })
    expect(resolveMessageKind(kindMessage)).toBe('json_render')

    const messageKindMessage = createMessage({ message_kind: 'json_render' })
    expect(resolveMessageKind(messageKindMessage)).toBe('json_render')
  })

  it('keeps text content as text even when it looks like json', () => {
    const jsonLookingText = '{"root":"card-1","elements":{}}'
    const message = createMessage({ content: jsonLookingText })
    expect(parseMessageContent(message)).toEqual({
      kind: 'text',
      textContent: jsonLookingText,
      jsonRenderSpec: null,
    })
  })

  it('returns json_render payload only when message kind is json_render', () => {
    const spec = { root: 'card-1', elements: { 'card-1': { type: 'Text', props: { text: 'Hello' } } } }
    const message = createMessage({
      kind: 'json_render',
      content: 'ignored for renderer',
      json_render_spec: spec,
    })
    expect(parseMessageContent(message)).toEqual({
      kind: 'json_render',
      textContent: null,
      jsonRenderSpec: spec,
    })
  })

  it('prefers json_render_spec over json_render when both exist', () => {
    const preferred = { root: 'preferred', elements: {} }
    const fallback = { root: 'fallback', elements: {} }
    const message = createMessage({
      kind: 'json_render',
      json_render_spec: preferred,
      json_render: fallback,
    })
    expect(resolveJsonRenderSpec(message)).toBe(preferred)
  })
})
