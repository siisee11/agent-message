import type { ConversationSummary, JsonRenderSpec, Message, MessageDetails, MessageKind } from './types'

export const DEFAULT_MESSAGE_KIND: MessageKind = 'text'

function resolveRawMessageKind(message: Pick<Message, 'kind' | 'message_kind'>): unknown {
  if (message.kind !== undefined && message.kind !== null) {
    return message.kind
  }
  return message.message_kind
}

export function resolveMessageKind(message: Pick<Message, 'kind' | 'message_kind'>): MessageKind {
  const rawKind = resolveRawMessageKind(message)
  if (rawKind === 'json_render') {
    return 'json_render'
  }
  return DEFAULT_MESSAGE_KIND
}

export function isJsonRenderMessage(message: Pick<Message, 'kind' | 'message_kind'>): boolean {
  return resolveMessageKind(message) === 'json_render'
}

export function resolveJsonRenderSpec(message: Pick<Message, 'json_render' | 'json_render_spec'>): JsonRenderSpec | null {
  if (message.json_render_spec !== undefined && message.json_render_spec !== null) {
    return message.json_render_spec
  }
  if (message.json_render !== undefined && message.json_render !== null) {
    return message.json_render
  }
  return null
}

export function normalizeMessageProtocol(message: Message): Message {
  const kind = resolveMessageKind(message)
  const jsonRenderSpec = resolveJsonRenderSpec(message)
  return {
    ...message,
    kind,
    json_render_spec: jsonRenderSpec ?? undefined,
  }
}

export interface ParsedMessageContent {
  kind: MessageKind
  textContent: string | null
  jsonRenderSpec: JsonRenderSpec | null
}

export function parseMessageContent(message: Message): ParsedMessageContent {
  const normalizedMessage = normalizeMessageProtocol(message)
  if (normalizedMessage.kind === 'json_render') {
    return {
      kind: 'json_render',
      textContent: null,
      jsonRenderSpec: resolveJsonRenderSpec(normalizedMessage),
    }
  }

  return {
    kind: 'text',
    textContent: normalizedMessage.content ?? null,
    jsonRenderSpec: null,
  }
}

export function normalizeMessageDetailsProtocol(details: MessageDetails): MessageDetails {
  return {
    ...details,
    message: normalizeMessageProtocol(details.message),
  }
}

export function normalizeConversationSummaryProtocol(summary: ConversationSummary): ConversationSummary {
  return {
    ...summary,
    last_message: summary.last_message ? normalizeMessageProtocol(summary.last_message) : undefined,
  }
}
