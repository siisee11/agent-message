import { describe, expect, it } from 'vitest'
import type { Message } from '../api'
import {
  canDeleteMessageForUser,
  canEditMessageForUser,
  MESSAGE_PREVIEW_DELETED,
  MESSAGE_PREVIEW_EMPTY,
  MESSAGE_PREVIEW_JSON_RENDER,
  resolveMessageRenderContent,
  summarizeLastMessagePreview,
} from './messagePresentation'

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

describe('message presentation helpers', () => {
  it('uses empty preview when no last message exists', () => {
    expect(summarizeLastMessagePreview(undefined)).toBe(MESSAGE_PREVIEW_EMPTY)
  })

  it('keeps deleted preview precedence over every other field', () => {
    const message = createMessage({
      deleted: true,
      kind: 'json_render',
      content: 'should not show',
      json_render_spec: { root: 'r1', elements: {} },
      attachment_type: 'image',
      attachment_url: 'https://example.test/image.png',
    })
    expect(summarizeLastMessagePreview(message)).toBe(MESSAGE_PREVIEW_DELETED)
    expect(resolveMessageRenderContent(message).variant).toBe('deleted')
  })

  it('shows compact json-render preview fallback and not raw text/spec', () => {
    const message = createMessage({
      kind: 'json_render',
      content: '{"root":"raw-json-that-should-not-preview"}',
      json_render_spec: { root: 'r1', elements: {} },
    })
    expect(summarizeLastMessagePreview(message)).toBe(MESSAGE_PREVIEW_JSON_RENDER)
  })

  it('combines attachment labels with json-render preview fallback', () => {
    const imageMessage = createMessage({
      kind: 'json_render',
      attachment_type: 'image',
      attachment_url: 'https://example.test/image.png',
      json_render_spec: { root: 'r1', elements: {} },
    })
    expect(summarizeLastMessagePreview(imageMessage)).toBe('[image] [json-render]')

    const fileMessage = createMessage({
      kind: 'json_render',
      attachment_type: 'file',
      attachment_url: 'https://example.test/file.pdf',
      json_render_spec: { root: 'r1', elements: {} },
    })
    expect(summarizeLastMessagePreview(fileMessage)).toBe('[file] [json-render]')
  })

  it('allows delete but disallows edit for json_render messages', () => {
    const message = createMessage({
      sender_id: 'me',
      kind: 'json_render',
      json_render_spec: { root: 'r1', elements: {} },
    })
    expect(canDeleteMessageForUser(message, 'me')).toBe(true)
    expect(canEditMessageForUser(message, 'me')).toBe(false)
  })

  it('allows edit only for text messages owned by the current user', () => {
    const ownText = createMessage({
      sender_id: 'me',
      kind: 'text',
      content: 'hello',
    })
    expect(canEditMessageForUser(ownText, 'me')).toBe(true)
    expect(canDeleteMessageForUser(ownText, 'me')).toBe(true)

    const otherText = createMessage({
      sender_id: 'other',
      kind: 'text',
      content: 'hello',
    })
    expect(canEditMessageForUser(otherText, 'me')).toBe(false)
    expect(canDeleteMessageForUser(otherText, 'me')).toBe(false)
  })

  it('resolves text and json_render content branches explicitly', () => {
    const textMessage = createMessage({
      kind: 'text',
      content: '  hello world  ',
    })
    expect(resolveMessageRenderContent(textMessage)).toEqual({
      variant: 'text',
      textContent: 'hello world',
      jsonRenderSpec: null,
    })

    const jsonMessage = createMessage({
      kind: 'json_render',
      json_render_spec: { root: 'r1', elements: {} },
      content: 'ignored',
    })
    expect(resolveMessageRenderContent(jsonMessage)).toEqual({
      variant: 'json_render',
      textContent: null,
      jsonRenderSpec: { root: 'r1', elements: {} },
    })
  })
})
