import { parseMessageContent, type ConversationSummary, type JsonRenderSpec, type Message } from '../api'

export const MESSAGE_PREVIEW_EMPTY = 'Start a conversation'
export const MESSAGE_PREVIEW_DELETED = 'This message was deleted'
export const MESSAGE_PREVIEW_JSON_RENDER = '[json-render]'
const JSON_RENDER_PREVIEW_MAX_LENGTH = 96

interface BareUIElement {
  type: string
  props?: Record<string, unknown>
  children?: string[]
}

interface BareUISpec {
  root: string
  elements: Record<string, BareUIElement>
}

const CWD_PREFIX_PATTERN = /^CWD:\s*(.+)$/im
const HOSTNAME_PREFIX_PATTERN = /^Hostname:\s*(.+)$/im

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isBareUISpec(value: JsonRenderSpec | null): value is BareUISpec {
  if (!isObject(value) || typeof value.root !== 'string' || !isObject(value.elements)) {
    return false
  }

  return Object.values(value.elements).every(
    (element) => isObject(element) && typeof element.type === 'string',
  )
}

function normalizePreviewText(value: string): string {
  return value
    .replace(/!\[[^\]]*\]\([^)]+\)/g, ' ')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/[`#>*_~|-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function clipPreview(value: string): string {
  if (value.length <= JSON_RENDER_PREVIEW_MAX_LENGTH) {
    return value
  }
  return `${value.slice(0, JSON_RENDER_PREVIEW_MAX_LENGTH - 1).trimEnd()}…`
}

function readStringProp(props: Record<string, unknown> | undefined, key: string): string | null {
  const value = props?.[key]
  if (typeof value !== 'string') {
    return null
  }
  const normalized = normalizePreviewText(value)
  return normalized === '' ? null : normalized
}

function readVerbatimStringProp(props: Record<string, unknown> | undefined, key: string): string | null {
  const value = props?.[key]
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function joinCandidateParts(parts: Array<string | null | undefined>, separator = ' '): string | null {
  const normalized = parts.filter((part): part is string => typeof part === 'string' && part.trim() !== '')
  if (normalized.length === 0) {
    return null
  }
  return normalized.join(separator)
}

function summarizeTable(props: Record<string, unknown> | undefined): string | null {
  if (!props) {
    return null
  }

  const caption = readStringProp(props, 'caption')
  const columns = Array.isArray(props.columns)
    ? props.columns.filter((value): value is string => typeof value === 'string' && value.trim() !== '').slice(0, 3)
    : []
  const rows = Array.isArray(props.rows)
    ? props.rows
        .find((row): row is unknown[] => Array.isArray(row))
        ?.filter((value): value is string => typeof value === 'string' && value.trim() !== '')
        .slice(0, 3) ?? []
    : []

  return joinCandidateParts(
    [
      caption,
      columns.length > 0 ? columns.join(', ') : null,
      rows.length > 0 ? rows.join(', ') : null,
    ],
    ' - ',
  )
}

function summarizeGitCommitLog(props: Record<string, unknown> | undefined): { primary?: string; secondary?: string } {
  if (!props) {
    return {}
  }

  let firstCommitSubject: string | null = null
  if (Array.isArray(props.commits)) {
    for (const value of props.commits) {
      if (isObject(value) && typeof value.subject === 'string' && value.subject.trim() !== '') {
        firstCommitSubject = value.subject
        break
      }
    }
  }

  return {
    primary:
      joinCandidateParts(
        [
          readVerbatimStringProp(props, 'title'),
          readVerbatimStringProp(props, 'repository'),
          readVerbatimStringProp(props, 'branch'),
          normalizePreviewText(firstCommitSubject ?? ''),
        ],
        ' - ',
      ) ?? undefined,
    secondary: readVerbatimStringProp(props, 'description') ?? undefined,
  }
}

function summarizeElement(element: BareUIElement): { primary?: string; secondary?: string; fallback?: string } {
  const props = isObject(element.props) ? element.props : undefined

  switch (element.type) {
    case 'Text':
    case 'Heading':
      return { primary: readStringProp(props, 'text') ?? undefined }
    case 'Alert':
      return {
        primary: joinCandidateParts([readStringProp(props, 'title'), readStringProp(props, 'message')]) ?? undefined,
      }
    case 'ApprovalCard':
      return {
        primary:
          joinCandidateParts([
            readStringProp(props, 'title'),
            Array.isArray(props?.details)
              ? props.details.find((value): value is string => typeof value === 'string' && value.trim() !== '')
              : null,
          ]) ?? undefined,
        secondary: readStringProp(props, 'badge') ?? undefined,
      }
    case 'Card':
      return {
        primary:
          joinCandidateParts([readStringProp(props, 'title'), readStringProp(props, 'description')]) ?? undefined,
      }
    case 'GitCommitLog':
      return summarizeGitCommitLog(props)
    case 'Markdown':
      return { primary: readStringProp(props, 'content') ?? undefined }
    case 'Table':
      return { secondary: summarizeTable(props) ?? undefined }
    case 'Progress':
      return { secondary: readStringProp(props, 'label') ?? undefined }
    case 'BarGraph':
    case 'LineGraph':
      return {
        secondary:
          joinCandidateParts([readStringProp(props, 'title'), readStringProp(props, 'description')]) ?? undefined,
      }
    case 'Avatar':
      return { fallback: readStringProp(props, 'name') ?? undefined }
    case 'Image':
      return { fallback: readStringProp(props, 'alt') ?? undefined }
    case 'Badge':
      return { fallback: readStringProp(props, 'text') ?? undefined }
    default:
      return {
        fallback:
          joinCandidateParts([
            readStringProp(props, 'title'),
            readStringProp(props, 'text'),
            readStringProp(props, 'label'),
            readStringProp(props, 'message'),
          ]) ?? undefined,
      }
  }
}

function extractCwdFromText(value: string | null | undefined): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const match = value.match(CWD_PREFIX_PATTERN)
  const cwd = match?.[1]?.trim()
  return cwd ? cwd : null
}

function extractHostnameFromText(value: string | null | undefined): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const match = value.match(HOSTNAME_PREFIX_PATTERN)
  const hostname = match?.[1]?.trim()
  return hostname ? hostname : null
}

function extractJsonRenderCwd(spec: JsonRenderSpec | null): string | null {
  if (!isBareUISpec(spec) || !spec.elements[spec.root]) {
    return null
  }

  const queue = [spec.root]
  const visited = new Set<string>()

  while (queue.length > 0) {
    const elementId = queue.shift()
    if (!elementId || visited.has(elementId)) {
      continue
    }
    visited.add(elementId)

    const element = spec.elements[elementId]
    if (!element) {
      continue
    }

    const props = isObject(element.props) ? element.props : undefined
    const candidateTexts = [
      typeof props?.title === 'string' ? props.title : null,
      typeof props?.message === 'string' ? props.message : null,
      typeof props?.text === 'string' ? props.text : null,
      typeof props?.description === 'string' ? props.description : null,
      typeof props?.label === 'string' ? props.label : null,
      typeof props?.content === 'string' ? props.content : null,
      ...(Array.isArray(props?.details)
        ? props.details.filter((value): value is string => typeof value === 'string')
        : []),
    ]

    for (const candidateText of candidateTexts) {
      const cwd = extractCwdFromText(candidateText)
      if (cwd) {
        return cwd
      }
    }

    if (Array.isArray(element.children)) {
      for (const child of element.children) {
        if (typeof child === 'string' && child.trim() !== '') {
          queue.push(child)
        }
      }
    }
  }

  return null
}

function extractJsonRenderHostname(spec: JsonRenderSpec | null): string | null {
  if (!isBareUISpec(spec) || !spec.elements[spec.root]) {
    return null
  }

  const queue = [spec.root]
  const visited = new Set<string>()

  while (queue.length > 0) {
    const elementId = queue.shift()
    if (!elementId || visited.has(elementId)) {
      continue
    }
    visited.add(elementId)

    const element = spec.elements[elementId]
    if (!element) {
      continue
    }

    const props = isObject(element.props) ? element.props : undefined
    const candidateTexts = [
      typeof props?.title === 'string' ? props.title : null,
      typeof props?.message === 'string' ? props.message : null,
      typeof props?.text === 'string' ? props.text : null,
      typeof props?.description === 'string' ? props.description : null,
      typeof props?.label === 'string' ? props.label : null,
      typeof props?.content === 'string' ? props.content : null,
      ...(Array.isArray(props?.details)
        ? props.details.filter((value): value is string => typeof value === 'string')
        : []),
    ]

    for (const candidateText of candidateTexts) {
      const hostname = extractHostnameFromText(candidateText)
      if (hostname) {
        return hostname
      }
    }

    if (Array.isArray(element.children)) {
      for (const child of element.children) {
        if (typeof child === 'string' && child.trim() !== '') {
          queue.push(child)
        }
      }
    }
  }

  return null
}

function extractJsonRenderPreview(spec: JsonRenderSpec | null): string | null {
  if (!isBareUISpec(spec) || !spec.elements[spec.root]) {
    return null
  }

  const queue = [spec.root]
  const visited = new Set<string>()
  const primary: string[] = []
  const secondary: string[] = []
  const fallback: string[] = []

  while (queue.length > 0) {
    const elementId = queue.shift()
    if (!elementId || visited.has(elementId)) {
      continue
    }
    visited.add(elementId)

    const element = spec.elements[elementId]
    if (!element) {
      continue
    }

    const summary = summarizeElement(element)
    if (summary.primary) {
      primary.push(summary.primary)
    }
    if (summary.secondary) {
      secondary.push(summary.secondary)
    }
    if (summary.fallback) {
      fallback.push(summary.fallback)
    }

    if (Array.isArray(element.children)) {
      for (const child of element.children) {
        if (typeof child === 'string' && child.trim() !== '') {
          queue.push(child)
        }
      }
    }
  }

  const candidates = primary.length > 0 ? primary : secondary.length > 0 ? secondary : fallback
  const preview = joinCandidateParts(candidates.slice(0, 2))
  if (!preview) {
    return null
  }
  return clipPreview(preview)
}

function resolveAttachmentLabel(message: Message): string | undefined {
  const attachments = message.attachments ?? []
  if (attachments.length > 1) {
    const imageCount = attachments.filter((attachment) => attachment.type === 'image').length
    if (imageCount === attachments.length) {
      return `[${imageCount} images]`
    }
    return `[${attachments.length} attachments]`
  }

  if (message.attachment_type === 'image') {
    return '[image]'
  }
  if (message.attachment_type === 'file') {
    return '[file]'
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
    const preview = extractJsonRenderPreview(parsed.jsonRenderSpec) ?? MESSAGE_PREVIEW_JSON_RENDER
    if (attachmentLabel) {
      return `${attachmentLabel} ${preview}`
    }
    return preview
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

export function extractMessageCwd(message: Message): string | null {
  if (message.deleted) {
    return null
  }

  const parsed = parseMessageContent(message)
  if (parsed.kind === 'json_render') {
    return extractJsonRenderCwd(parsed.jsonRenderSpec)
  }

  return extractCwdFromText(parsed.textContent)
}

export function extractMessageHostname(message: Message): string | null {
  if (message.deleted) {
    return null
  }

  const parsed = parseMessageContent(message)
  if (parsed.kind === 'json_render') {
    return extractJsonRenderHostname(parsed.jsonRenderSpec)
  }

  return extractHostnameFromText(parsed.textContent)
}

export function summarizeConversationLabel(
  summary: Pick<ConversationSummary, 'conversation' | 'other_user' | 'session_folder' | 'session_hostname'>,
): string {
  const title = summary.conversation.title?.trim()
  if (title) {
    return title
  }
  const displayLabel = joinCandidateParts([summary.session_folder, summary.session_hostname], ' · ')
  return displayLabel ?? summary.other_user.username
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
