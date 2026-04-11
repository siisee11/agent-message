import type {
  AttachmentType,
  ConversationSummary,
  JsonRenderSpec,
  Message,
  MessageAttachment,
  MessageDetails,
  MessageKind,
} from './types'

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

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function normalizeMessageAttachments(message: Message): MessageAttachment[] | undefined {
  if (Array.isArray(message.attachments) && message.attachments.length > 0) {
    const normalized = message.attachments.filter(
      (attachment): attachment is MessageAttachment =>
        isObject(attachment) &&
        typeof attachment.url === 'string' &&
        (attachment.type === 'image' || attachment.type === 'file'),
    )
    return normalized.length > 0 ? normalized : undefined
  }

  if (typeof message.attachment_url === 'string' && (message.attachment_type === 'image' || message.attachment_type === 'file')) {
    return [
      {
        url: message.attachment_url,
        type: message.attachment_type as AttachmentType,
      },
    ]
  }

  return undefined
}

export function normalizeMessageProtocol(message: Message): Message {
  const attachments = normalizeMessageAttachments(message)
  const firstAttachment = attachments?.[0]

  return {
    ...message,
    kind: resolveMessageKind(message),
    attachments,
    attachment_url: firstAttachment?.url,
    attachment_type: firstAttachment?.type,
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
