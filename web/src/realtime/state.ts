import type { InfiniteData } from '@tanstack/react-query'
import type { ConversationDetails, Message, MessageDetails, Reaction, UserProfile } from '../api'

export type MessageReactionsState = Record<string, Reaction[]>

export function fallbackSender(message: Message): UserProfile {
  return {
    id: message.sender_id,
    username: 'me',
    created_at: message.created_at,
  }
}

export function prependMessageToPages(
  current: InfiniteData<MessageDetails[]> | undefined,
  createdDetails: MessageDetails,
): InfiniteData<MessageDetails[]> {
  if (!current || current.pages.length === 0) {
    return {
      pageParams: [undefined],
      pages: [[createdDetails]],
    }
  }

  const alreadyExists = current.pages.some((page) =>
    page.some((details) => details.message.id === createdDetails.message.id),
  )
  if (alreadyExists) {
    return current
  }

  return {
    ...current,
    pages: [[createdDetails, ...current.pages[0]], ...current.pages.slice(1)],
  }
}

export function markMessageDeletedInPages(
  current: InfiniteData<MessageDetails[]> | undefined,
  messageId: string,
): InfiniteData<MessageDetails[]> | undefined {
  if (!current) {
    return current
  }

  let replaced = false
  const nextPages = current.pages.map((page) =>
    page.map((details) => {
      if (details.message.id !== messageId) {
        return details
      }

      replaced = true
      return {
        ...details,
        message: {
          ...details.message,
          deleted: true,
          content: undefined,
          attachment_url: undefined,
          attachment_type: undefined,
        },
      }
    }),
  )

  if (!replaced) {
    return current
  }

  return {
    ...current,
    pages: nextPages,
  }
}

export function replaceMessageInPages(
  current: InfiniteData<MessageDetails[]> | undefined,
  updatedMessage: Message,
): InfiniteData<MessageDetails[]> | undefined {
  if (!current) {
    return current
  }

  let replaced = false
  const nextPages = current.pages.map((page) =>
    page.map((details) => {
      if (details.message.id !== updatedMessage.id) {
        return details
      }
      replaced = true
      return {
        ...details,
        message: updatedMessage,
      }
    }),
  )

  if (!replaced) {
    return current
  }

  return {
    ...current,
    pages: nextPages,
  }
}

export function resolveRealtimeSender(
  message: Message,
  currentUser: UserProfile | null,
  conversationDetails: ConversationDetails | undefined,
): UserProfile {
  if (currentUser && message.sender_id === currentUser.id) {
    return currentUser
  }
  if (conversationDetails?.participant_a.id === message.sender_id) {
    return conversationDetails.participant_a
  }
  if (conversationDetails?.participant_b.id === message.sender_id) {
    return conversationDetails.participant_b
  }
  return fallbackSender(message)
}

export function addReactionToState(state: MessageReactionsState, reaction: Reaction): MessageReactionsState {
  const current = state[reaction.message_id] ?? []
  const alreadyExists = current.some(
    (existing) => existing.emoji === reaction.emoji && existing.user_id === reaction.user_id,
  )
  if (alreadyExists) {
    return state
  }
  return {
    ...state,
    [reaction.message_id]: [...current, reaction],
  }
}

export function removeReactionFromState(
  state: MessageReactionsState,
  removal: { message_id: string; emoji: string; user_id: string },
): MessageReactionsState {
  const current = state[removal.message_id] ?? []
  const next = current.filter(
    (reaction) => !(reaction.emoji === removal.emoji && reaction.user_id === removal.user_id),
  )

  if (next.length === current.length) {
    return state
  }

  if (next.length === 0) {
    const { [removal.message_id]: _removed, ...rest } = state
    return rest
  }

  return {
    ...state,
    [removal.message_id]: next,
  }
}
