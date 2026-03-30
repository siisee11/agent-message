import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { useLayoutEffect, useMemo, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError, type ConversationDetails, type Message, type MessageDetails, type UserProfile } from '../api'
import { apiClient } from '../api/runtime'
import { useAuth } from '../auth'
import styles from './DmConversationPage.module.css'

const MESSAGE_PAGE_SIZE = 20
const TIMESTAMP_FORMATTER = new Intl.DateTimeFormat(undefined, {
  month: 'short',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

interface ScrollAnchorSnapshot {
  conversationId: string
  scrollHeight: number
  scrollTop: number
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    return error.message
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message
  }
  return fallback
}

function resolveOtherParticipant(details: ConversationDetails, currentUser: UserProfile | null): UserProfile {
  if (!currentUser) {
    return details.participant_a
  }

  if (details.participant_a.id === currentUser.id) {
    return details.participant_b
  }

  if (details.participant_b.id === currentUser.id) {
    return details.participant_a
  }

  return details.participant_a
}

function formatMessageTimestamp(message: Message): string {
  const parsed = Date.parse(message.created_at)
  if (Number.isNaN(parsed)) {
    return message.created_at
  }
  return TIMESTAMP_FORMATTER.format(new Date(parsed))
}

export function DmConversationPage(): JSX.Element {
  const { conversationId } = useParams()
  const { user } = useAuth()

  const timelineRef = useRef<HTMLDivElement | null>(null)
  const initialScrollConversationRef = useRef<string | null>(null)
  const loadOlderAnchorRef = useRef<ScrollAnchorSnapshot | null>(null)

  const conversationQuery = useQuery({
    queryKey: ['conversation', conversationId],
    queryFn: () => apiClient.getConversation(conversationId as string),
    enabled: Boolean(conversationId),
  })

  const messagePagesQuery = useInfiniteQuery({
    queryKey: ['messages', conversationId],
    queryFn: ({ pageParam }) =>
      apiClient.listMessages(conversationId as string, {
        before: typeof pageParam === 'string' ? pageParam : undefined,
        limit: MESSAGE_PAGE_SIZE,
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      if (lastPage.length < MESSAGE_PAGE_SIZE) {
        return undefined
      }
      const oldestLoadedMessage = lastPage[lastPage.length - 1]
      return oldestLoadedMessage?.message.id
    },
    enabled: Boolean(conversationId),
  })

  const messagesAscending = useMemo(() => {
    const newestFirstMessages = (messagePagesQuery.data?.pages ?? []).flatMap((page) => page)
    return [...newestFirstMessages].reverse()
  }, [messagePagesQuery.data])

  const otherParticipant = useMemo(() => {
    if (!conversationQuery.data) {
      return null
    }
    return resolveOtherParticipant(conversationQuery.data, user)
  }, [conversationQuery.data, user])

  const hasOlderMessages = Boolean(messagePagesQuery.hasNextPage)

  useLayoutEffect(() => {
    const timeline = timelineRef.current
    if (!timeline || !conversationId) {
      return
    }

    const anchor = loadOlderAnchorRef.current
    if (anchor && anchor.conversationId === conversationId && !messagePagesQuery.isFetchingNextPage) {
      const nextScrollTop = timeline.scrollHeight - anchor.scrollHeight + anchor.scrollTop
      timeline.scrollTop = nextScrollTop
      loadOlderAnchorRef.current = null
      return
    }

    if (messagesAscending.length === 0) {
      return
    }

    if (initialScrollConversationRef.current !== conversationId) {
      timeline.scrollTop = timeline.scrollHeight
      initialScrollConversationRef.current = conversationId
    }
  }, [conversationId, messagePagesQuery.isFetchingNextPage, messagesAscending.length])

  function handleLoadOlder(): void {
    if (!conversationId || !hasOlderMessages || messagePagesQuery.isFetchingNextPage) {
      return
    }

    const timeline = timelineRef.current
    if (timeline) {
      loadOlderAnchorRef.current = {
        conversationId,
        scrollHeight: timeline.scrollHeight,
        scrollTop: timeline.scrollTop,
      }
    }

    void messagePagesQuery.fetchNextPage()
  }

  if (!conversationId) {
    return (
      <section className={styles.page}>
        <div className={styles.card}>
          <h2 className={styles.title}>Invalid conversation route</h2>
          <p className={styles.muted}>Conversation id is missing.</p>
        </div>
      </section>
    )
  }

  const timelineError = messagePagesQuery.isError
    ? resolveErrorMessage(messagePagesQuery.error, 'Failed to load messages.')
    : null

  return (
    <section className={styles.page}>
      <div className={styles.panel}>
        <header className={styles.header}>
          {conversationQuery.isLoading ? <h2 className={styles.title}>Loading conversation...</h2> : null}
          {conversationQuery.isError ? (
            <>
              <h2 className={styles.title}>Conversation unavailable</h2>
              <p className={styles.error}>{resolveErrorMessage(conversationQuery.error, 'Failed to load conversation.')}</p>
            </>
          ) : null}
          {conversationQuery.isSuccess && otherParticipant ? (
            <>
              <h2 className={styles.title}>@{otherParticipant.username}</h2>
              <p className={styles.muted}>Conversation ID: {conversationId}</p>
            </>
          ) : null}
        </header>

        <section className={styles.timelineSection}>
          <div className={styles.timelineToolbar}>
            <button
              className={styles.loadOlderButton}
              disabled={!hasOlderMessages || messagePagesQuery.isFetchingNextPage || messagePagesQuery.isLoading}
              onClick={handleLoadOlder}
              type="button"
            >
              {messagePagesQuery.isFetchingNextPage ? 'Loading older...' : 'Load older'}
            </button>
            <span className={styles.muted}>{messagesAscending.length} loaded</span>
          </div>

          {timelineError ? <p className={styles.error}>{timelineError}</p> : null}

          <div className={styles.timelineViewport} ref={timelineRef}>
            {messagePagesQuery.isLoading ? <p className={styles.muted}>Loading messages...</p> : null}
            {messagePagesQuery.isSuccess && messagesAscending.length === 0 ? (
              <p className={styles.muted}>No messages yet in this conversation.</p>
            ) : null}
            {messagesAscending.length > 0 ? (
              <ol className={styles.timelineList}>
                {messagesAscending.map((details: MessageDetails) => (
                  <li
                    className={`${styles.timelineItem} ${
                      details.message.sender_id === user?.id ? styles.timelineItemOwn : styles.timelineItemOther
                    }`}
                    key={details.message.id}
                  >
                    <div className={styles.messageBubble}>
                      <div className={styles.timelineMeta}>
                        <span className={styles.sender}>{details.sender.username}</span>
                        <span className={styles.timelineMetaRight}>
                          <span className={styles.timestamp}>{formatMessageTimestamp(details.message)}</span>
                          {!details.message.deleted && details.message.edited ? (
                            <span className={styles.editedBadge}>[수정됨]</span>
                          ) : null}
                        </span>
                      </div>

                      {details.message.deleted ? (
                        <p className={`${styles.messageText} ${styles.messageTextDeleted}`}>삭제된 메시지입니다</p>
                      ) : null}

                      {!details.message.deleted && details.message.content?.trim() ? (
                        <p className={styles.messageText}>{details.message.content.trim()}</p>
                      ) : null}

                      {!details.message.deleted &&
                      details.message.attachment_type === 'image' &&
                      details.message.attachment_url ? (
                        <a
                          className={styles.imageAttachmentLink}
                          href={details.message.attachment_url}
                          rel="noreferrer"
                          target="_blank"
                        >
                          <img
                            alt="Message attachment"
                            className={styles.imageAttachment}
                            loading="lazy"
                            src={details.message.attachment_url}
                          />
                        </a>
                      ) : null}

                      {!details.message.deleted &&
                      details.message.attachment_type === 'file' &&
                      details.message.attachment_url ? (
                        <a
                          className={styles.fileAttachmentLink}
                          download
                          href={details.message.attachment_url}
                          rel="noreferrer"
                          target="_blank"
                        >
                          첨부 파일 다운로드
                        </a>
                      ) : null}
                    </div>
                  </li>
                ))}
              </ol>
            ) : null}
          </div>
        </section>
      </div>
    </section>
  )
}
