import { type InfiniteData, useQueryClient } from '@tanstack/react-query'
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  type PropsWithChildren,
  useState,
} from 'react'
import { useLocation } from 'react-router-dom'
import type {
  ConversationDetails,
  Message,
  MessageDetails,
  Reaction,
  WatcherPresence,
  WatcherPresenceEvent,
} from '../api'
import { useAuth } from '../auth'
import { useEventStream, type EventStreamConnectionStatus } from '../hooks'
import {
  addUnreadConversation,
  addReactionToPages,
  announceRealtimeMessageWillAppend,
  markMessageDeletedInPages,
  prependMessageToPages,
  removeUnreadConversation,
  removeReactionFromPages,
  replaceMessageInPages,
  resolveRealtimeSender,
  shouldMarkConversationUnread,
} from './state'

interface RealtimeContextValue {
  status: EventStreamConnectionStatus
  unreadConversationIds: ReadonlySet<string>
}

const RealtimeContext = createContext<RealtimeContextValue | undefined>(undefined)

export function RealtimeProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient()
  const { isAuthenticated, token, user } = useAuth()
  const location = useLocation()
  const [unreadConversationIds, setUnreadConversationIds] = useState<Set<string>>(new Set())
  const activeConversationId = location.pathname.startsWith('/dm/')
    ? location.pathname.slice('/dm/'.length)
    : null

  const invalidateConversations = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ['conversations'] })
  }, [queryClient])

  const invalidateConversationMessages = useCallback(
    (conversationId: string | null) => {
      if (!conversationId) {
        return
      }

      void queryClient.invalidateQueries({ queryKey: ['messages', conversationId] })
    },
    [queryClient],
  )

  const invalidateConversationDetails = useCallback(
    (conversationId: string | null) => {
      if (!conversationId) {
        return
      }

      void queryClient.invalidateQueries({ queryKey: ['conversation', conversationId] })
    },
    [queryClient],
  )

  const refreshActiveConversation = useCallback(() => {
    invalidateConversations()
    invalidateConversationDetails(activeConversationId)
    invalidateConversationMessages(activeConversationId)
  }, [activeConversationId, invalidateConversationDetails, invalidateConversationMessages, invalidateConversations])

  const handleMessageNew = useCallback(
    (incomingMessage: Message) => {
      announceRealtimeMessageWillAppend(incomingMessage.conversation_id)
      if (shouldMarkConversationUnread(incomingMessage, user?.id, activeConversationId)) {
        setUnreadConversationIds((current) => addUnreadConversation(current, incomingMessage.conversation_id))
      }
      const key = ['messages', incomingMessage.conversation_id] as const
      const existingCache = queryClient.getQueryData<InfiniteData<MessageDetails[]>>(key)
      if (existingCache !== undefined) {
        const conversationDetails = queryClient.getQueryData<ConversationDetails>([
          'conversation',
          incomingMessage.conversation_id,
        ])
        queryClient.setQueryData<InfiniteData<MessageDetails[]>>(key, (current) =>
          prependMessageToPages(current, {
            message: incomingMessage,
            sender: resolveRealtimeSender(incomingMessage, user, conversationDetails),
            reactions: [],
          }),
        )
      } else {
        invalidateConversationMessages(incomingMessage.conversation_id)
      }
      invalidateConversations()
    },
    [activeConversationId, invalidateConversationMessages, invalidateConversations, queryClient, user],
  )

  const handleMessageEdited = useCallback(
    (updatedMessage: Message) => {
      queryClient.setQueryData<InfiniteData<MessageDetails[]>>(
        ['messages', updatedMessage.conversation_id],
        (current) => replaceMessageInPages(current, updatedMessage),
      )
      invalidateConversations()
    },
    [invalidateConversations, queryClient],
  )

  const handleMessageDeleted = useCallback(
    (messageId: string) => {
      queryClient.setQueriesData<InfiniteData<MessageDetails[]>>({ queryKey: ['messages'] }, (current) =>
        markMessageDeletedInPages(current, messageId),
      )
      invalidateConversations()
    },
    [invalidateConversations, queryClient],
  )

  const applyReactionAdded = useCallback((reaction: Reaction) => {
    queryClient.setQueriesData<InfiniteData<MessageDetails[]>>({ queryKey: ['messages'] }, (current) =>
      addReactionToPages(current, reaction),
    )
  }, [queryClient])

  const applyReactionRemoved = useCallback((removal: { message_id: string; emoji: string; user_id: string }) => {
    queryClient.setQueriesData<InfiniteData<MessageDetails[]>>({ queryKey: ['messages'] }, (current) =>
      removeReactionFromPages(current, removal),
    )
  }, [queryClient])

  const handleMessageNewEvent = useCallback(
    ({ data }: { data: Message }) => {
      handleMessageNew(data)
    },
    [handleMessageNew],
  )

  const handleMessageEditedEvent = useCallback(
    ({ data }: { data: Message }) => {
      handleMessageEdited(data)
    },
    [handleMessageEdited],
  )

  const handleMessageDeletedEvent = useCallback(
    ({ data }: { data: { id: string } }) => {
      handleMessageDeleted(data.id)
    },
    [handleMessageDeleted],
  )

  const handleReactionAddedEvent = useCallback(
    ({ data }: { data: Reaction }) => {
      applyReactionAdded(data)
    },
    [applyReactionAdded],
  )

  const handleReactionRemovedEvent = useCallback(
    ({ data }: { data: { message_id: string; emoji: string; user_id: string } }) => {
      applyReactionRemoved(data)
    },
    [applyReactionRemoved],
  )

  const handlePresenceUpdated = useCallback(
    (presence: WatcherPresenceEvent) => {
      queryClient.setQueryData<ConversationDetails>(['conversation', presence.conversation_id], (current) => {
        if (!current) {
          return current
        }

        const otherParticipant =
          current.participant_a.id === user?.id ? current.participant_b : current.participant_a
        if (otherParticipant.id !== presence.user_id || presence.client_kind !== 'watcher') {
          return current
        }

        const nextWatcherPresence: WatcherPresence = {
          user_id: presence.user_id,
          client_kind: presence.client_kind,
          online: presence.online,
        }

        if (
          current.watcher_presence?.user_id === nextWatcherPresence.user_id &&
          current.watcher_presence?.client_kind === nextWatcherPresence.client_kind &&
          current.watcher_presence?.online === nextWatcherPresence.online
        ) {
          return current
        }

        return {
          ...current,
          watcher_presence: nextWatcherPresence,
        }
      })
    },
    [queryClient, user],
  )

  const handlePresenceUpdatedEvent = useCallback(
    ({ data }: { data: WatcherPresenceEvent }) => {
      handlePresenceUpdated(data)
    },
    [handlePresenceUpdated],
  )

  useEffect(() => {
    setUnreadConversationIds((current) => removeUnreadConversation(current, activeConversationId))
  }, [activeConversationId])

  useEffect(() => {
    if (isAuthenticated) {
      return
    }

    setUnreadConversationIds(new Set())
  }, [isAuthenticated])

  const eventStream = useEventStream({
    token,
    enabled: isAuthenticated,
    onOpen: refreshActiveConversation,
    onMessageNew: handleMessageNewEvent,
    onMessageEdited: handleMessageEditedEvent,
    onMessageDeleted: handleMessageDeletedEvent,
    onReactionAdded: handleReactionAddedEvent,
    onReactionRemoved: handleReactionRemovedEvent,
    onPresenceUpdated: handlePresenceUpdatedEvent,
  })

  const value = useMemo<RealtimeContextValue>(
    () => ({
      status: eventStream.status,
      unreadConversationIds,
    }),
    [eventStream.status, unreadConversationIds],
  )

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>
}

export function useRealtime(): RealtimeContextValue {
  const context = useContext(RealtimeContext)
  if (!context) {
    throw new Error('useRealtime must be used within RealtimeProvider')
  }
  return context
}
