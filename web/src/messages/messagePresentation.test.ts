import { describe, expect, it } from 'vitest'
import type { Message } from '../api'
import {
  canDeleteMessageForUser,
  canEditMessageForUser,
  extractMessageCwd,
  extractMessageHostname,
  MESSAGE_PREVIEW_DELETED,
  MESSAGE_PREVIEW_EMPTY,
  MESSAGE_PREVIEW_JSON_RENDER,
  resolveMessageRenderContent,
  summarizeConversationLabel,
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
      json_render_spec: {
        root: 'stack-1',
        elements: {
          'stack-1': {
            type: 'Stack',
            children: ['heading-1', 'text-1'],
          },
          'heading-1': {
            type: 'Heading',
            props: { text: 'Deploy status' },
          },
          'text-1': {
            type: 'Text',
            props: { text: 'Build running on production' },
          },
        },
      },
    })
    expect(summarizeLastMessagePreview(message)).toBe('Deploy status Build running on production')
  })

  it('combines attachment labels with json-render preview fallback', () => {
    const imageMessage = createMessage({
      kind: 'json_render',
      attachment_type: 'image',
      attachment_url: 'https://example.test/image.png',
      json_render_spec: {
        root: 'alert-1',
        elements: {
          'alert-1': {
            type: 'Alert',
            props: {
              title: 'Heads up',
              message: 'Quarterly report is ready',
            },
          },
        },
      },
    })
    expect(summarizeLastMessagePreview(imageMessage)).toBe('[image] Heads up Quarterly report is ready')

    const fileMessage = createMessage({
      kind: 'json_render',
      attachment_type: 'file',
      attachment_url: 'https://example.test/file.pdf',
      json_render_spec: { root: 'r1', elements: {} },
    })
    expect(summarizeLastMessagePreview(fileMessage)).toBe('[file] [json-render]')
  })

  it('summarizes multiple image attachments compactly', () => {
    const message = createMessage({
      attachments: [
        { url: 'https://example.test/1.png', type: 'image' },
        { url: 'https://example.test/2.png', type: 'image' },
      ],
    })

    expect(summarizeLastMessagePreview(message)).toBe('[2 images]')
  })

  it('extracts a useful preview from table-based json render messages', () => {
    const message = createMessage({
      kind: 'json_render',
      json_render_spec: {
        root: 'table-1',
        elements: {
          'table-1': {
            type: 'Table',
            props: {
              columns: ['Step', 'Status'],
              rows: [['Build', 'running']],
            },
          },
        },
      },
    })

    expect(summarizeLastMessagePreview(message)).toBe('Step, Status - Build, running')
  })

  it('extracts a useful preview from git commit log json render messages', () => {
    const message = createMessage({
      kind: 'json_render',
      json_render_spec: {
        root: 'commit-log-1',
        elements: {
          'commit-log-1': {
            type: 'GitCommitLog',
            props: {
              title: 'Release history',
              repository: 'agent-message',
              branch: 'main',
              commits: [
                {
                  sha: '5e7f6b8f7c2b9a12f6b0c10b46c2cd884973a001',
                  subject: 'Ship commit log component',
                },
              ],
            },
          },
        },
      },
    })

    expect(summarizeLastMessagePreview(message)).toBe('Release history - agent-message - main - Ship commit log component')
  })

  it('extracts a useful preview from ask-question json render messages', () => {
    const message = createMessage({
      kind: 'json_render',
      json_render_spec: {
        root: 'question-1',
        elements: {
          'question-1': {
            type: 'AskQuestion',
            props: {
              question: 'Which environment should I use?',
              freeformPlaceholder: 'Type a custom environment',
            },
          },
        },
      },
    })

    expect(summarizeLastMessagePreview(message)).toBe('Which environment should I use?')
  })

  it('falls back to placeholder when json render has no extractable text', () => {
    const message = createMessage({
      kind: 'json_render',
      json_render_spec: { root: 'r1', elements: {} },
    })

    expect(summarizeLastMessagePreview(message)).toBe(MESSAGE_PREVIEW_JSON_RENDER)
  })

  it('extracts cwd from approval-card details in json render messages', () => {
    const message = createMessage({
      kind: 'json_render',
      json_render_spec: {
        root: 'approval-1',
        elements: {
          'approval-1': {
            type: 'ApprovalCard',
            props: {
              title: 'Command approval requested',
              details: ['Command: npm test', 'CWD: /Users/jay/git/agent-message'],
            },
          },
        },
      },
    })

    expect(extractMessageCwd(message)).toBe('/Users/jay/git/agent-message')
  })

  it('extracts hostname from approval-card details in json render messages', () => {
    const message = createMessage({
      kind: 'json_render',
      json_render_spec: {
        root: 'approval-1',
        elements: {
          'approval-1': {
            type: 'ApprovalCard',
            props: {
              title: 'Command approval requested',
              details: ['Command: npm test', 'Hostname: devbox.local'],
            },
          },
        },
      },
    })

    expect(extractMessageHostname(message)).toBe('devbox.local')
  })

  it('extracts cwd from plain text messages', () => {
    const message = createMessage({
      kind: 'text',
      content: 'Request received\nCWD: /tmp/worktree\nStatus: running',
    })

    expect(extractMessageCwd(message)).toBe('/tmp/worktree')
  })

  it('extracts hostname from plain text messages', () => {
    const message = createMessage({
      kind: 'text',
      content: 'Request received\nCWD: /tmp/worktree\nHostname: devbox.local\nStatus: running',
    })

    expect(extractMessageHostname(message)).toBe('devbox.local')
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

  it('summarizes conversation labels from session folder and hostname', () => {
    expect(
      summarizeConversationLabel({
        conversation: { id: 'c1', participant_a: 'u1', participant_b: 'u2', created_at: '2026-03-30T00:00:00.000Z' },
        other_user: { id: 'u2', account_id: 'agent-123', username: 'agent-123', created_at: '2026-03-30T00:00:00.000Z' },
        session_folder: 'agent-message',
        session_hostname: 'devbox.local',
      }),
    ).toBe('agent-message · devbox.local')
  })

  it('falls back to other username when session metadata is missing', () => {
    expect(
      summarizeConversationLabel({
        conversation: { id: 'c1', participant_a: 'u1', participant_b: 'u2', created_at: '2026-03-30T00:00:00.000Z' },
        other_user: { id: 'u2', account_id: 'agent-123', username: 'agent-123', created_at: '2026-03-30T00:00:00.000Z' },
      }),
    ).toBe('agent-123')
  })

  it('prefers conversation title over username and session metadata', () => {
    expect(
      summarizeConversationLabel({
        conversation: {
          id: 'c1',
          participant_a: 'u1',
          participant_b: 'u2',
          title: 'Janet Agent Message',
          created_at: '2026-03-30T00:00:00.000Z',
        },
        other_user: { id: 'u2', account_id: 'agent-123', username: 'agent-123', created_at: '2026-03-30T00:00:00.000Z' },
        session_folder: 'agent-message',
        session_hostname: 'devbox.local',
      }),
    ).toBe('Janet Agent Message')
  })
})
