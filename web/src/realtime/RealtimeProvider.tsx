import { type InfiniteData, useQueryClient } from '@tanstack/react-query'
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from 'react'
import type { ConversationDetails, Message, MessageDetails, Reaction } from '../api'
import { useAuth } from '../auth'
import { useEventStream, type EventStreamConnectionStatus } from '../hooks'
import {
  addReactionToState,
  type MessageReactionsState,
  markMessageDeletedInPages,
  prependMessageToPages,
  removeReactionFromState,
  replaceMessageInPages,
  resolveRealtimeSender,
} from './state'

interface RealtimeContextValue {
  status: EventStreamConnectionStatus
  messageReactions: MessageReactionsState
  applyReactionAdded: (reaction: Reaction) => void
  applyReactionRemoved: (removal: { message_id: string; emoji: string; user_id: string }) => void
}

const RealtimeContext = createContext<RealtimeContextValue | undefined>(undefined)

export function RealtimeProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient()
  const { isAuthenticated, token, user } = useAuth()
  const [messageReactions, setMessageReactions] = useState<MessageReactionsState>({})

  const invalidateConversations = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ['conversations'] })
  }, [queryClient])

  const handleMessageNew = useCallback(
    (incomingMessage: Message) => {
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
          }),
        )
      }
      invalidateConversations()
    },
    [invalidateConversations, queryClient, user],
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
    setMessageReactions((state) => addReactionToState(state, reaction))
  }, [])

  const applyReactionRemoved = useCallback((removal: { message_id: string; emoji: string; user_id: string }) => {
    setMessageReactions((state) => removeReactionFromState(state, removal))
  }, [])

  useEffect(() => {
    if (!token) {
      setMessageReactions({})
    }
  }, [token])

  const eventStream = useEventStream({
    token,
    enabled: Boolean(isAuthenticated && token),
    onMessageNew: ({ data }) => handleMessageNew(data),
    onMessageEdited: ({ data }) => handleMessageEdited(data),
    onMessageDeleted: ({ data }) => handleMessageDeleted(data.id),
    onReactionAdded: ({ data }) => applyReactionAdded(data),
    onReactionRemoved: ({ data }) => applyReactionRemoved(data),
  })

  const value = useMemo<RealtimeContextValue>(
    () => ({
      status: eventStream.status,
      messageReactions,
      applyReactionAdded,
      applyReactionRemoved,
    }),
    [applyReactionAdded, applyReactionRemoved, eventStream.status, messageReactions],
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
