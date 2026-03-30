import { parseMessageContent, type JsonRenderSpec, type Message } from '../api'

export const MESSAGE_PREVIEW_EMPTY = '대화를 시작해 보세요'
export const MESSAGE_PREVIEW_DELETED = '삭제된 메시지입니다'
export const MESSAGE_PREVIEW_JSON_RENDER = '[json-render]'

function resolveAttachmentLabel(message: Message): string | undefined {
  if (message.attachment_type === 'image') {
    return '[이미지]'
  }
  if (message.attachment_type === 'file') {
    return '[파일]'
  }
  return undefined
}

export function summarizeLastMessagePreview(lastMessage?: Message): string {
  if (!lastMessage) {
    return MESSAGE_PREVIEW_EMPTY
  }

  if (lastMessage.deleted) {
    return MESSAGE_PREVIEW_DELETED
  }

  const parsed = parseMessageContent(lastMessage)
  const attachmentLabel = resolveAttachmentLabel(lastMessage)

  if (parsed.kind === 'json_render') {
    if (attachmentLabel) {
      return `${attachmentLabel} ${MESSAGE_PREVIEW_JSON_RENDER}`
    }
    return MESSAGE_PREVIEW_JSON_RENDER
  }

  const content = parsed.textContent?.trim()
  if (attachmentLabel && content) {
    return `${attachmentLabel} ${content}`
  }

  if (content) {
    return content
  }

  if (attachmentLabel) {
    return attachmentLabel
  }

  return MESSAGE_PREVIEW_EMPTY
}

export function canDeleteMessageForUser(message: Message, currentUserId: string | undefined): boolean {
  return message.sender_id === currentUserId && !message.deleted
}

export function canEditMessageForUser(message: Message, currentUserId: string | undefined): boolean {
  if (!canDeleteMessageForUser(message, currentUserId)) {
    return false
  }
  return parseMessageContent(message).kind === 'text'
}

export type MessageRenderVariant = 'deleted' | 'text' | 'json_render' | 'empty'

export interface MessageRenderContent {
  variant: MessageRenderVariant
  textContent: string | null
  jsonRenderSpec: JsonRenderSpec | null
}

export function resolveMessageRenderContent(message: Message): MessageRenderContent {
  if (message.deleted) {
    return {
      variant: 'deleted',
      textContent: null,
      jsonRenderSpec: null,
    }
  }

  const parsed = parseMessageContent(message)
  if (parsed.kind === 'json_render') {
    return {
      variant: 'json_render',
      textContent: null,
      jsonRenderSpec: parsed.jsonRenderSpec,
    }
  }

  const trimmedText = parsed.textContent?.trim()
  if (trimmedText) {
    return {
      variant: 'text',
      textContent: trimmedText,
      jsonRenderSpec: null,
    }
  }

  return {
    variant: 'empty',
    textContent: null,
    jsonRenderSpec: null,
  }
}
