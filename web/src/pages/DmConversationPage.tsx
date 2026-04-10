import {
  type InfiniteData,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ApiError,
  type ConversationDetails,
  type Message,
  type MessageDetails,
  type Reaction,
  type UserProfile,
} from '../api'
import { apiClient } from '../api/runtime'
import { useAuth } from '../auth'
import { MessageJsonRender } from '../components/MessageJsonRender'
import { ThemeToggleButton } from '../components/ThemeToggleButton'
import {
  canDeleteMessageForUser,
  canEditMessageForUser,
  extractMessageCwd,
  extractMessageHostname,
  MESSAGE_PREVIEW_DELETED,
  resolveMessageRenderContent,
} from '../messages/messagePresentation'
import { formatRealtimeStatusLabel, useRealtime } from '../realtime'
import { useTheme } from '../theme'
import { useDocumentSurface } from '../hooks'
import {
  fallbackSender,
  prependMessageToPages,
  replaceMessageInPages,
} from '../realtime/state'
import styles from './DmConversationPage.module.css'

const MESSAGE_PAGE_SIZE = 20
const TIMELINE_PULL_TRIGGER_PX = 32
const TIMESTAMP_FORMATTER = new Intl.DateTimeFormat('en-US', {
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

interface EditTarget {
  messageId: string
}

interface ActionMenuState {
  messageId: string
  x: number
  y: number
}

interface ReactionGroup {
  emoji: string
  count: number
  reactedByMe: boolean
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

function inferAttachmentType(file: File): 'image' | 'file' {
  return file.type.startsWith('image/') ? 'image' : 'file'
}

function PlusIcon() {
  return (
    <svg aria-hidden="true" fill="none" height="14" viewBox="0 0 22 22" width="14">
      <path
        d="M11 4.75v12.5M4.75 11h12.5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="2.2"
      />
    </svg>
  )
}

function SendIcon() {
  return (
    <svg aria-hidden="true" fill="none" height="16" viewBox="0 0 26 26" width="16">
      <path
        d="M5.8 13.2 19.9 6.6c.77-.36 1.54.4 1.18 1.18l-6.6 14.1c-.39.83-1.61.73-1.86-.16l-1.4-5.02a1.2 1.2 0 0 0-.84-.84l-5.02-1.4c-.89-.25-.99-1.47-.16-1.86Z"
        fill="currentColor"
      />
      <path d="m11.2 14.8 4.8-4.8" stroke="#ffffff" strokeLinecap="round" strokeWidth="1.8" />
    </svg>
  )
}

function scrollTimelineToBottom(timeline: HTMLDivElement | null): void {
  if (!timeline) {
    return
  }

  timeline.scrollTop = timeline.scrollHeight
}

function isTimelineNearBottom(timeline: HTMLDivElement, threshold = 48): boolean {
  const distanceFromBottom = timeline.scrollHeight - timeline.clientHeight - timeline.scrollTop
  return distanceFromBottom <= threshold
}

function groupReactionsByEmoji(
  reactions: Reaction[] | undefined,
  currentUserId: string | undefined,
): ReactionGroup[] {
  if (!reactions || reactions.length === 0) {
    return []
  }

  const grouped = new Map<string, ReactionGroup>()
  for (const reaction of reactions) {
    const existing = grouped.get(reaction.emoji)
    if (!existing) {
      grouped.set(reaction.emoji, {
        emoji: reaction.emoji,
        count: 1,
        reactedByMe: currentUserId === reaction.user_id,
      })
      continue
    }

    existing.count += 1
    if (currentUserId === reaction.user_id) {
      existing.reactedByMe = true
    }
  }

  return Array.from(grouped.values())
}

export function DmConversationPage() {
  const { themeColor } = useTheme()

  useDocumentSurface({
    backgroundColor: 'var(--app-surface-background)',
    themeColor,
  })

  const navigate = useNavigate()
  const { conversationId } = useParams()
  const { user } = useAuth()
  const realtime = useRealtime()
  const queryClient = useQueryClient()

  const timelineRef = useRef<HTMLDivElement | null>(null)
  const timelineContentRef = useRef<HTMLDivElement | null>(null)
  const initialScrollConversationRef = useRef<string | null>(null)
  const loadOlderAnchorRef = useRef<ScrollAnchorSnapshot | null>(null)
  const shouldStickToBottomRef = useRef(true)
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const actionMenuRef = useRef<HTMLDivElement | null>(null)
  const composerInputRef = useRef<HTMLTextAreaElement | null>(null)

  const [composerText, setComposerText] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [composerError, setComposerError] = useState<string | null>(null)
  const [editingTarget, setEditingTarget] = useState<EditTarget | null>(null)
  const [actionMenu, setActionMenu] = useState<ActionMenuState | null>(null)
  const [isComposerFocused, setIsComposerFocused] = useState(false)
  const [unreadCount, setUnreadCount] = useState(0)
  const prevMessageCountRef = useRef(0)

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

  const conversationCwd = useMemo(() => {
    for (let index = messagesAscending.length - 1; index >= 0; index -= 1) {
      const cwd = extractMessageCwd(messagesAscending[index].message)
      if (cwd) {
        return cwd
      }
    }
    return null
  }, [messagesAscending])

  const conversationHostname = useMemo(() => {
    for (let index = messagesAscending.length - 1; index >= 0; index -= 1) {
      const hostname = extractMessageHostname(messagesAscending[index].message)
      if (hostname) {
        return hostname
      }
    }
    return null
  }, [messagesAscending])

  const hasOlderMessages = Boolean(messagePagesQuery.hasNextPage)

  const handleMessageCreated = useCallback(
    async (createdMessage: Message, options?: { resetComposer?: boolean }) => {
      if (!conversationId) {
        return
      }

      setComposerError(null)
      if (options?.resetComposer ?? false) {
        setEditingTarget(null)
        setComposerText('')
        setSelectedFile(null)
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
      }

      const createdDetails: MessageDetails = {
        message: createdMessage,
        sender: user ?? fallbackSender(createdMessage),
        reactions: [],
      }

      queryClient.setQueryData<InfiniteData<MessageDetails[]>>(['messages', conversationId], (current) =>
        prependMessageToPages(current, createdDetails),
      )
      await queryClient.invalidateQueries({ queryKey: ['conversations'] })

      shouldStickToBottomRef.current = true
      scrollTimelineToBottom(timelineRef.current)
    },
    [conversationId, queryClient, user],
  )

  const sendMessageMutation = useMutation({
    mutationFn: async (input: { content: string; attachment: File | null }) => {
      if (!conversationId) {
        throw new Error('Conversation id is missing.')
      }

      const trimmedContent = input.content.trim()
      if (input.attachment) {
        const uploadedAttachment = await apiClient.uploadFile(input.attachment, 'file')
        return apiClient.sendMessage(conversationId, {
          content: trimmedContent === '' ? undefined : trimmedContent,
          attachmentUrl: uploadedAttachment.url,
          attachmentType: inferAttachmentType(input.attachment),
        })
      }

      return apiClient.sendMessage(conversationId, {
        content: trimmedContent,
      })
    },
    onSuccess: async (createdMessage) => {
      await handleMessageCreated(createdMessage, { resetComposer: true })
    },
    onError: (error: unknown) => {
      setComposerError(resolveErrorMessage(error, 'Failed to send the message.'))
    },
  })

  const approvalResponseMutation = useMutation({
    mutationFn: async (value: string) => {
      if (!conversationId) {
        throw new Error('Conversation id is missing.')
      }

      return apiClient.sendMessage(conversationId, {
        content: value.trim(),
      })
    },
    onSuccess: async (createdMessage) => {
      await handleMessageCreated(createdMessage)
    },
  })

  const editMessageMutation = useMutation({
    mutationFn: async (input: { messageId: string; content: string }) =>
      apiClient.editMessage(input.messageId, { content: input.content }),
    onSuccess: async (updatedMessage) => {
      if (!conversationId) {
        return
      }

      setComposerError(null)
      setEditingTarget(null)
      setComposerText('')
      queryClient.setQueryData<InfiniteData<MessageDetails[]>>(['messages', conversationId], (current) =>
        replaceMessageInPages(current, updatedMessage),
      )
      await queryClient.invalidateQueries({ queryKey: ['conversations'] })
    },
    onError: (error: unknown) => {
      setComposerError(resolveErrorMessage(error, 'Failed to edit the message.'))
    },
  })

  const deleteMessageMutation = useMutation({
    mutationFn: async (messageId: string) => apiClient.deleteMessage(messageId),
    onSuccess: async (deletedMessage) => {
      if (!conversationId) {
        return
      }

      if (editingTarget?.messageId === deletedMessage.id) {
        setEditingTarget(null)
        setComposerText('')
      }
      setComposerError(null)
      setActionMenu(null)
      queryClient.setQueryData<InfiniteData<MessageDetails[]>>(['messages', conversationId], (current) =>
        replaceMessageInPages(current, deletedMessage),
      )
      await queryClient.invalidateQueries({ queryKey: ['conversations'] })
    },
    onError: (error: unknown) => {
      setComposerError(resolveErrorMessage(error, 'Failed to delete the message.'))
    },
  })

  const actionMenuMessage = useMemo(() => {
    if (!actionMenu) {
      return null
    }
    return messagesAscending.find((details) => details.message.id === actionMenu.messageId) ?? null
  }, [actionMenu, messagesAscending])

  const loadOlderMessages = useCallback((): void => {
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

    shouldStickToBottomRef.current = false
    void messagePagesQuery.fetchNextPage()
  }, [conversationId, hasOlderMessages, messagePagesQuery])

  const resizeComposerInput = useCallback((element: HTMLTextAreaElement | null): void => {
    if (!element) {
      return
    }

    element.style.height = '0px'
    element.style.height = `${element.scrollHeight}px`
  }, [])

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
      shouldStickToBottomRef.current = true
      scrollTimelineToBottom(timeline)
      initialScrollConversationRef.current = conversationId
    }
  }, [conversationId, messagePagesQuery.isFetchingNextPage, messagesAscending.length])

  useLayoutEffect(() => {
    resizeComposerInput(composerInputRef.current)
  }, [composerText, resizeComposerInput])

  useEffect(() => {
    shouldStickToBottomRef.current = true
    loadOlderAnchorRef.current = null
    setUnreadCount(0)
    prevMessageCountRef.current = 0
  }, [conversationId])

  useEffect(() => {
    const prevCount = prevMessageCountRef.current
    const currentCount = messagesAscending.length
    if (prevCount > 0 && currentCount > prevCount && !shouldStickToBottomRef.current) {
      setUnreadCount((prev) => prev + (currentCount - prevCount))
    }
    prevMessageCountRef.current = currentCount
  }, [messagesAscending.length])

  useEffect(() => {
    const timeline = timelineRef.current
    if (!timeline) {
      return
    }

    const handleScroll = () => {
      const isNearBottom = isTimelineNearBottom(timeline)
      shouldStickToBottomRef.current = isNearBottom
      if (isNearBottom) {
        setUnreadCount(0)
      }
      if (timeline.scrollTop <= TIMELINE_PULL_TRIGGER_PX) {
        loadOlderMessages()
      }
    }

    handleScroll()
    timeline.addEventListener('scroll', handleScroll)
    return () => {
      timeline.removeEventListener('scroll', handleScroll)
    }
  }, [conversationId, loadOlderMessages])

  useEffect(() => {
    const timeline = timelineRef.current
    if (!timeline) {
      return
    }
    if (timeline.scrollHeight <= timeline.clientHeight + TIMELINE_PULL_TRIGGER_PX) {
      loadOlderMessages()
    }
  }, [conversationId, loadOlderMessages, messagesAscending.length])

  useEffect(() => {
    const timeline = timelineRef.current
    const timelineContent = timelineContentRef.current
    if (!timeline || !timelineContent || typeof ResizeObserver === 'undefined') {
      return
    }

    const observer = new ResizeObserver(() => {
      if (shouldStickToBottomRef.current) {
        scrollTimelineToBottom(timeline)
      }
    })

    observer.observe(timelineContent)
    return () => {
      observer.disconnect()
    }
  }, [conversationId, messagesAscending.length])

  useEffect(() => {
    if (!actionMenu) {
      return
    }

    const handleWindowMouseDown = (event: MouseEvent) => {
      const target = event.target as Node | null
      if (target && actionMenuRef.current?.contains(target)) {
        return
      }
      setActionMenu(null)
    }

    const handleWindowKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setActionMenu(null)
      }
    }

    window.addEventListener('mousedown', handleWindowMouseDown)
    window.addEventListener('keydown', handleWindowKeyDown)
    return () => {
      window.removeEventListener('mousedown', handleWindowMouseDown)
      window.removeEventListener('keydown', handleWindowKeyDown)
    }
  }, [actionMenu])

  useEffect(() => {
    if (!actionMenuMessage) {
      setActionMenu(null)
    }
  }, [actionMenuMessage])

  function handleScrollToBottom(): void {
    shouldStickToBottomRef.current = true
    setUnreadCount(0)
    scrollTimelineToBottom(timelineRef.current)
  }

  function openActionMenu(messageId: string, x: number, y: number): void {
    const padding = 8
    const menuWidth = 200
    const menuHeight = 90
    const nextX = Math.min(x, window.innerWidth - menuWidth - padding)
    const nextY = Math.min(y, window.innerHeight - menuHeight - padding)
    setActionMenu({
      messageId,
      x: Math.max(padding, nextX),
      y: Math.max(padding, nextY),
    })
  }

  function handleOpenContextMenu(event: React.MouseEvent, details: MessageDetails): void {
    if (!canDeleteMessageForUser(details.message, user?.id)) {
      return
    }
    event.preventDefault()
    openActionMenu(details.message.id, event.clientX, event.clientY)
  }

  function handleToggleActionMenu(button: HTMLButtonElement, details: MessageDetails): void {
    if (!canDeleteMessageForUser(details.message, user?.id)) {
      return
    }
    if (actionMenu?.messageId === details.message.id) {
      setActionMenu(null)
      return
    }
    const bounds = button.getBoundingClientRect()
    openActionMenu(details.message.id, bounds.right, bounds.bottom + 4)
  }

  function beginEdit(details: MessageDetails): void {
    if (!canEditMessageForUser(details.message, user?.id)) {
      setComposerError('json-render messages cannot be edited.')
      setActionMenu(null)
      return
    }

    setEditingTarget({
      messageId: details.message.id,
    })
    setComposerText(details.message.content?.trim() ?? '')
    setSelectedFile(null)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
    setComposerError(null)
    setActionMenu(null)
  }

  function cancelEdit(): void {
    setEditingTarget(null)
    setComposerText('')
    setComposerError(null)
  }

  function handleSelectAttachment(event: React.ChangeEvent<HTMLInputElement>): void {
    const file = event.target.files?.[0] ?? null
    setSelectedFile(file)
  }

  function clearSelectedAttachment(): void {
    setSelectedFile(null)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  function handleOpenFilePicker(): void {
    if (disableComposerActions || editingTarget) {
      return
    }

    fileInputRef.current?.click()
  }

  function handleComposerSubmit(event: React.FormEvent<HTMLFormElement>): void {
    event.preventDefault()
    submitComposer()
  }

  function submitComposer(): void {
    setComposerError(null)

    const trimmedContent = composerText.trim()
    if (editingTarget) {
      const editingDetails = messagesAscending.find((details) => details.message.id === editingTarget.messageId)
      if (!editingDetails || !canEditMessageForUser(editingDetails.message, user?.id)) {
        setEditingTarget(null)
        setComposerText('')
        setComposerError('json-render messages cannot be edited.')
        return
      }

      if (trimmedContent === '') {
        setComposerError('Edited messages cannot be empty.')
        return
      }
      editMessageMutation.mutate({
        messageId: editingTarget.messageId,
        content: trimmedContent,
      })
      return
    }

    if (trimmedContent === '' && !selectedFile) {
      return
    }

    sendMessageMutation.mutate({
      content: composerText,
      attachment: selectedFile,
    })
  }

  function handleComposerKeyDown(event: React.KeyboardEvent<HTMLTextAreaElement>): void {
    if (event.key !== 'Enter' || event.shiftKey || event.nativeEvent.isComposing) {
      return
    }

    event.preventDefault()
    submitComposer()
  }

  function handleComposerChange(event: React.ChangeEvent<HTMLTextAreaElement>): void {
    resizeComposerInput(event.target)
    setComposerText(event.target.value)
  }

  function handleComposerFieldPointerDown(event: React.PointerEvent<HTMLDivElement>): void {
    if (event.target instanceof HTMLTextAreaElement || disableComposerActions) {
      return
    }

    composerInputRef.current?.focus()
  }

  const handleApprovalAction = useCallback(
    async (value: string) => {
      await approvalResponseMutation.mutateAsync(value)
    },
    [approvalResponseMutation],
  )

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
  const hasComposerContent = composerText.trim() !== '' || Boolean(selectedFile)
  const disableComposerActions =
    sendMessageMutation.isPending ||
    approvalResponseMutation.isPending ||
    editMessageMutation.isPending ||
    deleteMessageMutation.isPending
  const headerTitle = conversationQuery.isLoading
    ? 'Loading conversation...'
    : conversationQuery.isError
      ? 'Conversation unavailable'
      : otherParticipant
        ? `@${otherParticipant.username}`
        : 'Conversation'
  const headerStatus = conversationQuery.isError
    ? resolveErrorMessage(conversationQuery.error, 'Failed to load conversation.')
    : formatRealtimeStatusLabel(realtime.status)
  const watcherPresence = conversationQuery.data?.watcher_presence
  const watcherStatusLabel =
    conversationQuery.isError || !watcherPresence
      ? null
      : watcherPresence.online
        ? 'Watcher online'
        : 'Watcher offline'
  const headerCwdValue = conversationQuery.isLoading
    ? 'loading...'
    : conversationQuery.isError
      ? 'unavailable'
      : conversationCwd ?? 'unavailable'
  const headerHostnameValue = conversationQuery.isLoading
    ? 'loading...'
    : conversationQuery.isError
      ? 'unavailable'
      : conversationHostname ?? 'unavailable'

  return (
    <section className={styles.page}>
      <div className={styles.panel}>
        <header className={styles.header}>
          <div className={styles.headerBar}>
            <button
              aria-label="Back to conversations"
              className={styles.backButton}
              onClick={() => navigate('/')}
              type="button"
            >
              ←
            </button>
            <div className={styles.headerCopy}>
              <div className={styles.headerTitleRow}>
                <h2 className={styles.title}>{headerTitle}</h2>
                <span
                  className={`${styles.headerStatusBadge}${
                    conversationQuery.isError ? ` ${styles.headerStatusBadgeError}` : ''
                  }`}
                >
                  {headerStatus}
                </span>
                {watcherStatusLabel ? (
                  <span
                    className={`${styles.watcherStatusBadge} ${
                      watcherPresence?.online ? styles.watcherStatusBadgeOnline : styles.watcherStatusBadgeOffline
                    }`}
                  >
                    {watcherStatusLabel}
                  </span>
                ) : null}
              </div>
              <p className={styles.headerCwd} title={`cwd: ${headerCwdValue}`}>
                {`cwd: ${headerCwdValue}`}
              </p>
              <p className={styles.headerHostname} title={`hostname: ${headerHostnameValue}`}>
                {`hostname: ${headerHostnameValue}`}
              </p>
            </div>
            <div className={styles.headerActions}>
              <ThemeToggleButton />
            </div>
          </div>
        </header>

        <section className={styles.timelineSection}>
          <div className={styles.timelineViewport} ref={timelineRef}>
            <div className={styles.timelineContent} ref={timelineContentRef}>
              {messagePagesQuery.isFetchingNextPage ? (
                <p className={styles.timelinePullStatus}>Loading older messages...</p>
              ) : null}
              {timelineError ? <p className={styles.error}>{timelineError}</p> : null}
              {messagePagesQuery.isLoading ? <p className={styles.muted}>Loading messages...</p> : null}
              {messagePagesQuery.isSuccess && messagesAscending.length === 0 ? (
                <p className={styles.muted}>No messages yet in this conversation.</p>
              ) : null}
              {messagesAscending.length > 0 ? (
                <ol className={styles.timelineList}>
                  {messagesAscending.map((details: MessageDetails) => {
                    const renderContent = resolveMessageRenderContent(details.message)
                    const reactionGroups = groupReactionsByEmoji(details.reactions, user?.id)
                    const isOwnMessage = details.message.sender_id === user?.id
                    const isAgentMessage = !isOwnMessage
                    const messageSurfaceClassName = isOwnMessage
                      ? styles.messageBubbleOwnFull
                      : styles.messageBubbleAgent
                    const timelineMetaClassName = isOwnMessage ? styles.timelineMetaOwn : styles.timelineMetaAgent
                    const messageTextClassName = isOwnMessage ? styles.messageTextOwn : styles.messageTextAgent

                    return (
                      <li
                        className={`${styles.timelineItem} ${
                          isOwnMessage ? styles.timelineItemOwn : styles.timelineItemOther
                        } ${styles.timelineItemFullWidth}${isAgentMessage ? ` ${styles.timelineItemAgent}` : ''}`}
                        key={details.message.id}
                        onContextMenu={(event) => handleOpenContextMenu(event, details)}
                      >
                        <div className={`${styles.messageBubble} ${messageSurfaceClassName}`}>
                          <div className={`${styles.timelineMeta} ${timelineMetaClassName}`}>
                            <span className={styles.sender}>{details.sender.username}</span>
                            <span className={styles.timelineMetaRight}>
                              <span className={styles.timestamp}>{formatMessageTimestamp(details.message)}</span>
                              {!details.message.deleted && details.message.edited ? (
                                <span className={styles.editedBadge}>[edited]</span>
                              ) : null}
                              {canDeleteMessageForUser(details.message, user?.id) ? (
                                <button
                                  aria-label="Message actions"
                                  className={styles.messageActionsTrigger}
                                  onClick={(event) => {
                                    handleToggleActionMenu(event.currentTarget, details)
                                  }}
                                  type="button"
                                >
                                  ⋯
                                </button>
                              ) : null}
                            </span>
                          </div>

                          {renderContent.variant === 'deleted' ? (
                            <p
                              className={`${styles.messageText} ${styles.messageTextDeleted}${
                                messageTextClassName ? ` ${messageTextClassName}` : ''
                              }`}
                            >
                              {MESSAGE_PREVIEW_DELETED}
                            </p>
                          ) : null}

                          {renderContent.variant === 'text' && renderContent.textContent ? (
                            <p className={`${styles.messageText} ${messageTextClassName}`}>
                              {renderContent.textContent}
                            </p>
                          ) : null}

                          {renderContent.variant === 'json_render' ? (
                            <MessageJsonRender
                              approvalDisabled={approvalResponseMutation.isPending || sendMessageMutation.isPending}
                              onApprovalAction={handleApprovalAction}
                              spec={renderContent.jsonRenderSpec}
                            />
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
                              Download attachment
                            </a>
                          ) : null}

                          {!details.message.deleted && reactionGroups.length > 0 ? (
                            <div className={styles.reactionSection}>
                              <div className={styles.reactionGroups}>
                                {reactionGroups.map((group) => (
                                  <span
                                    className={`${styles.reactionChip}${
                                      group.reactedByMe ? ` ${styles.reactionChipOwn}` : ''
                                    }`}
                                    key={group.emoji}
                                  >
                                    <span>{group.emoji}</span>
                                    <span>{group.count}</span>
                                  </span>
                                ))}
                              </div>
                            </div>
                          ) : null}
                        </div>
                      </li>
                    )
                  })}
                </ol>
              ) : null}
            </div>
          </div>

          <form
            className={`${styles.composerForm}${isComposerFocused ? ` ${styles.composerFormKeyboardOpen}` : ''}`}
            onSubmit={handleComposerSubmit}
          >
            {unreadCount > 0 ? (
              <button className={styles.unreadBanner} onClick={handleScrollToBottom} type="button">
                {unreadCount === 1 ? '1 unread message' : `${unreadCount} unread messages`}
              </button>
            ) : null}
            {editingTarget ? (
              <div className={styles.editingBanner}>
                <span>Editing message</span>
                <button className={styles.editingCancelButton} onClick={cancelEdit} type="button">
                  Cancel
                </button>
              </div>
            ) : null}

            {selectedFile ? (
              <div className={styles.attachmentChip}>
                <span className={styles.attachmentName}>{selectedFile.name}</span>
                <button
                  className={styles.attachmentRemove}
                  disabled={disableComposerActions}
                  onClick={clearSelectedAttachment}
                  type="button"
                >
                  Remove
                </button>
              </div>
            ) : null}

            <div className={styles.composerDock}>
              <input
                className={styles.attachInput}
                disabled={disableComposerActions || Boolean(editingTarget)}
                onChange={handleSelectAttachment}
                ref={fileInputRef}
                type="file"
              />

              <button
                aria-label="Attach file"
                className={`${styles.iconButton} ${styles.attachButton}${
                  editingTarget ? ` ${styles.attachButtonDisabled}` : ''
                }`}
                disabled={disableComposerActions || Boolean(editingTarget)}
                onClick={handleOpenFilePicker}
                title={editingTarget ? 'Attachments are unavailable while editing.' : 'Attach file'}
                type="button"
              >
                <PlusIcon />
              </button>

              <div className={styles.composerField} onPointerDown={handleComposerFieldPointerDown}>
                <textarea
                  className={styles.composerInput}
                  disabled={disableComposerActions}
                  onChange={handleComposerChange}
                  onBlur={() => setIsComposerFocused(false)}
                  onFocus={() => setIsComposerFocused(true)}
                  onKeyDown={handleComposerKeyDown}
                  placeholder={editingTarget ? 'Edit message...' : 'Message'}
                  ref={composerInputRef}
                  rows={1}
                  value={composerText}
                />
              </div>

              <button
                aria-label={
                  editMessageMutation.isPending
                    ? 'Saving message'
                    : sendMessageMutation.isPending
                      ? 'Sending message'
                      : editingTarget
                        ? 'Save edit'
                        : 'Send message'
                }
                className={styles.submitButton}
                disabled={disableComposerActions || !hasComposerContent}
                title={
                  editingTarget
                    ? editMessageMutation.isPending
                      ? 'Saving...'
                      : 'Save edit'
                    : sendMessageMutation.isPending
                      ? 'Sending...'
                      : 'Send message'
                }
                type="submit"
              >
                <SendIcon />
              </button>
            </div>

            {composerError ? <p className={styles.error}>{composerError}</p> : null}
          </form>
        </section>
      </div>

      {actionMenu && actionMenuMessage ? (
        <div
          className={styles.contextMenu}
          ref={actionMenuRef}
          style={{ left: `${actionMenu.x}px`, top: `${actionMenu.y}px` }}
        >
          {canEditMessageForUser(actionMenuMessage.message, user?.id) ? (
            <button
              className={styles.contextMenuAction}
              disabled={disableComposerActions}
              onClick={() => beginEdit(actionMenuMessage)}
              type="button"
            >
              Edit
            </button>
          ) : null}
          <button
            className={`${styles.contextMenuAction} ${styles.contextMenuActionDanger}`}
            disabled={disableComposerActions}
            onClick={() => deleteMessageMutation.mutate(actionMenuMessage.message.id)}
            type="button"
          >
            Delete
          </button>
        </div>
      ) : null}
    </section>
  )
}
