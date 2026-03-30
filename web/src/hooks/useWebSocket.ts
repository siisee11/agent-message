import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Message, Reaction } from '../api'

const DEFAULT_WS_PATH = import.meta.env.VITE_WS_URL?.trim() || '/ws'
const DEFAULT_INITIAL_RETRY_DELAY_MS = 500
const DEFAULT_MAX_RETRY_DELAY_MS = 10_000

export type WebSocketConnectionStatus = 'idle' | 'connecting' | 'open' | 'reconnecting' | 'closed'

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

export type ServerWebSocketEvent = KnownServerEvent | UnknownServerEvent

export interface UseWebSocketOptions {
  token: string | null
  enabled?: boolean
  wsPath?: string
  initialRetryDelayMs?: number
  maxRetryDelayMs?: number
  maxReconnectAttempts?: number
  onOpen?: () => void
  onClose?: (event: CloseEvent) => void
  onError?: (event: Event) => void
  onEvent?: (event: ServerWebSocketEvent) => void
  onMessageNew?: (event: MessageNewEvent) => void
  onMessageEdited?: (event: MessageEditedEvent) => void
  onMessageDeleted?: (event: MessageDeletedEvent) => void
  onReactionAdded?: (event: ReactionAddedEvent) => void
  onReactionRemoved?: (event: ReactionRemovedEvent) => void
}

interface ReadEventPayload {
  type: 'read'
  data: {
    conversation_id: string
  }
}

export interface UseWebSocketResult {
  status: WebSocketConnectionStatus
  reconnectAttempts: number
  lastEvent: ServerWebSocketEvent | null
  sendEvent: (event: unknown) => boolean
  sendReadEvent: (conversationId: string) => boolean
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

function parseServerEvent(rawData: unknown): ServerWebSocketEvent | null {
  if (!isObject(rawData)) {
    return null
  }

  const type = rawData.type
  if (typeof type !== 'string' || type.trim() === '') {
    return null
  }

  const data = rawData.data
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

function isMessageNewEvent(event: ServerWebSocketEvent): event is MessageNewEvent {
  return event.type === 'message.new'
}

function isMessageEditedEvent(event: ServerWebSocketEvent): event is MessageEditedEvent {
  return event.type === 'message.edited'
}

function isMessageDeletedEvent(event: ServerWebSocketEvent): event is MessageDeletedEvent {
  return event.type === 'message.deleted'
}

function isReactionAddedEvent(event: ServerWebSocketEvent): event is ReactionAddedEvent {
  return event.type === 'reaction.added'
}

function isReactionRemovedEvent(event: ServerWebSocketEvent): event is ReactionRemovedEvent {
  return event.type === 'reaction.removed'
}

function resolveWebSocketURL(token: string, wsPath: string): string | null {
  if (typeof window === 'undefined') {
    return null
  }

  let url: URL
  if (wsPath.startsWith('ws://') || wsPath.startsWith('wss://')) {
    url = new URL(wsPath)
  } else if (wsPath.startsWith('http://') || wsPath.startsWith('https://')) {
    url = new URL(wsPath)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  } else {
    const baseProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const normalizedPath = wsPath.startsWith('/') ? wsPath : `/${wsPath}`
    url = new URL(normalizedPath, `${baseProtocol}//${window.location.host}`)
  }

  url.searchParams.set('token', token)
  return url.toString()
}

export function useWebSocket(options: UseWebSocketOptions): UseWebSocketResult {
  const {
    token,
    enabled = true,
    wsPath = DEFAULT_WS_PATH,
    initialRetryDelayMs = DEFAULT_INITIAL_RETRY_DELAY_MS,
    maxRetryDelayMs = DEFAULT_MAX_RETRY_DELAY_MS,
    maxReconnectAttempts,
    onOpen,
    onClose,
    onError,
    onEvent,
    onMessageNew,
    onMessageEdited,
    onMessageDeleted,
    onReactionAdded,
    onReactionRemoved,
  } = options

  const [status, setStatus] = useState<WebSocketConnectionStatus>('idle')
  const [lastEvent, setLastEvent] = useState<ServerWebSocketEvent | null>(null)
  const [reconnectAttempts, setReconnectAttempts] = useState(0)
  const [reconnectSignal, setReconnectSignal] = useState(0)

  const socketRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const manualCloseRef = useRef(false)

  const shouldConnect = Boolean(enabled && token)

  const clearReconnectTimeout = useCallback(() => {
    if (reconnectTimeoutRef.current !== null && typeof window !== 'undefined') {
      window.clearTimeout(reconnectTimeoutRef.current)
    }
    reconnectTimeoutRef.current = null
  }, [])

  const closeSocket = useCallback(
    (isManualClose: boolean) => {
      manualCloseRef.current = isManualClose
      const activeSocket = socketRef.current
      socketRef.current = null
      if (activeSocket && (activeSocket.readyState === WebSocket.OPEN || activeSocket.readyState === WebSocket.CONNECTING)) {
        activeSocket.close()
      }
    },
    [],
  )

  useEffect(() => {
    reconnectAttemptsRef.current = 0
    setReconnectAttempts(0)
    setReconnectSignal(0)
    setLastEvent(null)
  }, [token, enabled, wsPath])

  useEffect(() => {
    clearReconnectTimeout()

    if (!shouldConnect) {
      closeSocket(true)
      setStatus('idle')
      return
    }

    const trimmedToken = token?.trim() ?? ''
    const wsURL = resolveWebSocketURL(trimmedToken, wsPath)
    if (!wsURL) {
      setStatus('idle')
      return
    }

    manualCloseRef.current = false
    setStatus(reconnectAttemptsRef.current > 0 ? 'reconnecting' : 'connecting')

    const socket = new WebSocket(wsURL)
    socketRef.current = socket

    socket.onopen = () => {
      reconnectAttemptsRef.current = 0
      setReconnectAttempts(0)
      setStatus('open')
      onOpen?.()
    }

    socket.onmessage = (event) => {
      let parsed: unknown
      try {
        parsed = JSON.parse(event.data as string)
      } catch {
        return
      }

      const serverEvent = parseServerEvent(parsed)
      if (!serverEvent) {
        return
      }

      setLastEvent(serverEvent)
      onEvent?.(serverEvent)

      if (isMessageNewEvent(serverEvent)) {
        onMessageNew?.(serverEvent)
      } else if (isMessageEditedEvent(serverEvent)) {
        onMessageEdited?.(serverEvent)
      } else if (isMessageDeletedEvent(serverEvent)) {
        onMessageDeleted?.(serverEvent)
      } else if (isReactionAddedEvent(serverEvent)) {
        onReactionAdded?.(serverEvent)
      } else if (isReactionRemovedEvent(serverEvent)) {
        onReactionRemoved?.(serverEvent)
      }
    }

    socket.onerror = (event) => {
      onError?.(event)
    }

    socket.onclose = (event) => {
      onClose?.(event)
      socketRef.current = null
      clearReconnectTimeout()

      if (manualCloseRef.current || !shouldConnect) {
        setStatus('closed')
        return
      }

      const nextAttempt = reconnectAttemptsRef.current + 1
      if (maxReconnectAttempts !== undefined && nextAttempt > maxReconnectAttempts) {
        setStatus('closed')
        return
      }

      reconnectAttemptsRef.current = nextAttempt
      setReconnectAttempts(nextAttempt)
      setStatus('reconnecting')

      const exponentialDelay = initialRetryDelayMs * 2 ** (nextAttempt - 1)
      const delay = Math.min(maxRetryDelayMs, exponentialDelay)
      reconnectTimeoutRef.current = window.setTimeout(() => {
        setReconnectSignal((value) => value + 1)
      }, delay)
    }

    return () => {
      clearReconnectTimeout()
      if (socketRef.current === socket) {
        closeSocket(true)
      }
    }
  }, [
    clearReconnectTimeout,
    closeSocket,
    initialRetryDelayMs,
    maxReconnectAttempts,
    maxRetryDelayMs,
    onClose,
    onError,
    onEvent,
    onMessageDeleted,
    onMessageEdited,
    onMessageNew,
    onOpen,
    onReactionAdded,
    onReactionRemoved,
    reconnectSignal,
    shouldConnect,
    token,
    wsPath,
  ])

  const sendEvent = useCallback((event: unknown): boolean => {
    const activeSocket = socketRef.current
    if (!activeSocket || activeSocket.readyState !== WebSocket.OPEN) {
      return false
    }
    activeSocket.send(JSON.stringify(event))
    return true
  }, [])

  const sendReadEvent = useCallback(
    (conversationId: string): boolean => {
      const normalizedConversationID = conversationId.trim()
      if (normalizedConversationID === '') {
        return false
      }

      const event: ReadEventPayload = {
        type: 'read',
        data: {
          conversation_id: normalizedConversationID,
        },
      }
      return sendEvent(event)
    },
    [sendEvent],
  )

  return useMemo<UseWebSocketResult>(
    () => ({
      status,
      reconnectAttempts,
      lastEvent,
      sendEvent,
      sendReadEvent,
    }),
    [lastEvent, reconnectAttempts, sendEvent, sendReadEvent, status],
  )
}
