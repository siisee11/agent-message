import { useEffect, useMemo, useRef, useState } from 'react'
import type { Message, Reaction } from '../api'

const DEFAULT_SSE_URL = resolveDefaultSSEURL()

export type EventStreamConnectionStatus = 'idle' | 'connecting' | 'open' | 'closed'

type KnownEventType =
  | 'message.new'
  | 'message.edited'
  | 'message.deleted'
  | 'reaction.added'
  | 'reaction.removed'

interface EventEnvelope<TType extends string, TData> {
  type: TType
  data: TData
}

export type MessageNewEvent = EventEnvelope<'message.new', Message>
export type MessageEditedEvent = EventEnvelope<'message.edited', Message>
export type MessageDeletedEvent = EventEnvelope<'message.deleted', { id: string }>
export type ReactionAddedEvent = EventEnvelope<'reaction.added', Reaction>
export type ReactionRemovedEvent = EventEnvelope<
  'reaction.removed',
  { message_id: string; emoji: string; user_id: string }
>
export type UnknownServerEvent = EventEnvelope<string, unknown>

export type KnownServerEvent =
  | MessageNewEvent
  | MessageEditedEvent
  | MessageDeletedEvent
  | ReactionAddedEvent
  | ReactionRemovedEvent

export type ServerEvent = KnownServerEvent | UnknownServerEvent

export interface UseEventStreamOptions {
  token: string | null
  enabled?: boolean
  eventURL?: string
  onOpen?: () => void
  onError?: () => void
  onEvent?: (event: ServerEvent) => void
  onMessageNew?: (event: MessageNewEvent) => void
  onMessageEdited?: (event: MessageEditedEvent) => void
  onMessageDeleted?: (event: MessageDeletedEvent) => void
  onReactionAdded?: (event: ReactionAddedEvent) => void
  onReactionRemoved?: (event: ReactionRemovedEvent) => void
}

export interface UseEventStreamResult {
  status: EventStreamConnectionStatus
  lastEvent: ServerEvent | null
}

function resolveDefaultSSEURL(): string {
  const configuredSSEURL = import.meta.env.VITE_SSE_URL?.trim()
  if (configuredSSEURL) {
    return configuredSSEURL
  }

  const apiBaseURL = import.meta.env.VITE_API_BASE_URL?.trim()
  if (!apiBaseURL) {
    return '/api/events'
  }

  return `${apiBaseURL.replace(/\/+$/, '')}/api/events`
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isKnownEventType(type: string): type is KnownEventType {
  return (
    type === 'message.new' ||
    type === 'message.edited' ||
    type === 'message.deleted' ||
    type === 'reaction.added' ||
    type === 'reaction.removed'
  )
}

function parseServerEvent(rawData: string): ServerEvent | null {
  let parsedJSON: unknown
  try {
    parsedJSON = JSON.parse(rawData)
  } catch {
    return null
  }

  if (!isObject(parsedJSON)) {
    return null
  }

  const type = parsedJSON.type
  if (typeof type !== 'string' || type.trim() === '') {
    return null
  }

  const data = parsedJSON.data
  if (!isKnownEventType(type)) {
    return { type, data }
  }

  switch (type) {
    case 'message.new':
      return { type, data: data as Message }
    case 'message.edited':
      return { type, data: data as Message }
    case 'message.deleted':
      return { type, data: data as { id: string } }
    case 'reaction.added':
      return { type, data: data as Reaction }
    case 'reaction.removed':
      return {
        type,
        data: data as { message_id: string; emoji: string; user_id: string },
      }
  }
}

export function useEventStream(options: UseEventStreamOptions): UseEventStreamResult {
  const {
    token,
    enabled = true,
    eventURL = DEFAULT_SSE_URL,
    onOpen,
    onError,
    onEvent,
    onMessageNew,
    onMessageEdited,
    onMessageDeleted,
    onReactionAdded,
    onReactionRemoved,
  } = options

  const [status, setStatus] = useState<EventStreamConnectionStatus>('idle')
  const [lastEvent, setLastEvent] = useState<ServerEvent | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  useEffect(() => {
    if (!enabled || !token) {
      eventSourceRef.current?.close()
      eventSourceRef.current = null
      setStatus('idle')
      setLastEvent(null)
      return
    }

    const normalizedToken = token.trim()
    if (normalizedToken === '') {
      setStatus('idle')
      setLastEvent(null)
      return
    }

    const normalizedURL = eventURL.includes('?')
      ? `${eventURL}&token=${encodeURIComponent(normalizedToken)}`
      : `${eventURL}?token=${encodeURIComponent(normalizedToken)}`

    setStatus('connecting')
    const source = new EventSource(normalizedURL)
    eventSourceRef.current = source

    const handleKnownEvent =
      (listener?: (event: any) => void) =>
      (event: MessageEvent<string>): void => {
        const serverEvent = parseServerEvent(event.data)
        if (!serverEvent) {
          return
        }
        setLastEvent(serverEvent)
        onEvent?.(serverEvent)
        listener?.(serverEvent)
      }

    source.onopen = () => {
      setStatus('open')
      onOpen?.()
    }

    source.onerror = () => {
      setStatus(source.readyState === EventSource.CLOSED ? 'closed' : 'connecting')
      onError?.()
    }

    source.addEventListener('message.new', handleKnownEvent(onMessageNew))
    source.addEventListener('message.edited', handleKnownEvent(onMessageEdited))
    source.addEventListener('message.deleted', handleKnownEvent(onMessageDeleted))
    source.addEventListener('reaction.added', handleKnownEvent(onReactionAdded))
    source.addEventListener('reaction.removed', handleKnownEvent(onReactionRemoved))

    return () => {
      source.close()
      if (eventSourceRef.current === source) {
        eventSourceRef.current = null
      }
    }
  }, [
    enabled,
    eventURL,
    onError,
    onEvent,
    onMessageDeleted,
    onMessageEdited,
    onMessageNew,
    onOpen,
    onReactionAdded,
    onReactionRemoved,
    token,
  ])

  return useMemo(
    () => ({
      status,
      lastEvent,
    }),
    [lastEvent, status],
  )
}
