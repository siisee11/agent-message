import type { ConversationSummary, JsonRenderSpec, Message, MessageDetails, MessageKind } from './types'

export const DEFAULT_MESSAGE_KIND: MessageKind = 'text'

export function resolveMessageKind(message: Pick<Message, 'kind'>): MessageKind {
  if (message.kind === 'json_render') {
    return 'json_render'
  }
  return DEFAULT_MESSAGE_KIND
}

export function isJsonRenderMessage(message: Pick<Message, 'kind'>): boolean {
  return resolveMessageKind(message) === 'json_render'
}

export function resolveJsonRenderSpec(message: Pick<Message, 'json_render_spec'>): JsonRenderSpec | null {
  return message.json_render_spec ?? null
}

export function normalizeMessageProtocol(message: Message): Message {
  return {
    ...message,
    kind: resolveMessageKind(message),
  }
}

export interface ParsedMessageContent {
  kind: MessageKind
  textContent: string | null
  jsonRenderSpec: JsonRenderSpec | null
}

export function parseMessageContent(message: Message): ParsedMessageContent {
  if (resolveMessageKind(message) === 'json_render') {
    return {
      kind: 'json_render',
      textContent: null,
      jsonRenderSpec: resolveJsonRenderSpec(message),
    }
  }

  return {
    kind: 'text',
    textContent: message.content ?? null,
    jsonRenderSpec: null,
  }
}

export function normalizeMessageDetailsProtocol(details: MessageDetails): MessageDetails {
  return {
    ...details,
    message: normalizeMessageProtocol(details.message),
    reactions: Array.isArray(details.reactions) ? details.reactions : [],
  }
}

export function normalizeConversationSummaryProtocol(summary: ConversationSummary): ConversationSummary {
  return {
    ...summary,
    last_message: summary.last_message ? normalizeMessageProtocol(summary.last_message) : undefined,
  }
}
