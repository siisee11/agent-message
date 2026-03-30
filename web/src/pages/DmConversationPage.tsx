import {
  type InfiniteData,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
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

interface EditTarget {
  messageId: string
}

interface ActionMenuState {
  messageId: string
  x: number
  y: number
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

function fallbackSender(message: Message): UserProfile {
  return {
    id: message.sender_id,
    username: 'me',
    created_at: message.created_at,
  }
}

function prependMessageToPages(
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

function replaceMessageInPages(
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

export function DmConversationPage(): JSX.Element {
  const { conversationId } = useParams()
  const { user } = useAuth()
  const queryClient = useQueryClient()

  const timelineRef = useRef<HTMLDivElement | null>(null)
  const initialScrollConversationRef = useRef<string | null>(null)
  const loadOlderAnchorRef = useRef<ScrollAnchorSnapshot | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const actionMenuRef = useRef<HTMLDivElement | null>(null)

  const [composerText, setComposerText] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [composerError, setComposerError] = useState<string | null>(null)
  const [editingTarget, setEditingTarget] = useState<EditTarget | null>(null)
  const [actionMenu, setActionMenu] = useState<ActionMenuState | null>(null)

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
  const isSubmitting = messagePagesQuery.isFetchingNextPage

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
      if (!conversationId) {
        return
      }

      setComposerError(null)
      setEditingTarget(null)
      setComposerText('')
      setSelectedFile(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }

      const createdDetails: MessageDetails = {
        message: createdMessage,
        sender: user ?? fallbackSender(createdMessage),
      }

      queryClient.setQueryData<InfiniteData<MessageDetails[]>>(['messages', conversationId], (current) =>
        prependMessageToPages(current, createdDetails),
      )
      await queryClient.invalidateQueries({ queryKey: ['conversations'] })

      const timeline = timelineRef.current
      if (timeline) {
        timeline.scrollTop = timeline.scrollHeight
      }
    },
    onError: (error: unknown) => {
      setComposerError(resolveErrorMessage(error, '메시지를 전송하지 못했습니다.'))
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
      setComposerError(resolveErrorMessage(error, '메시지를 수정하지 못했습니다.'))
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
      setComposerError(resolveErrorMessage(error, '메시지를 삭제하지 못했습니다.'))
    },
  })

  const actionMenuMessage = useMemo(() => {
    if (!actionMenu) {
      return null
    }
    return messagesAscending.find((details) => details.message.id === actionMenu.messageId) ?? null
  }, [actionMenu, messagesAscending])

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
    if (details.message.sender_id !== user?.id || details.message.deleted) {
      return
    }
    event.preventDefault()
    openActionMenu(details.message.id, event.clientX, event.clientY)
  }

  function handleToggleActionMenu(button: HTMLButtonElement, details: MessageDetails): void {
    if (details.message.sender_id !== user?.id || details.message.deleted) {
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

  function handleComposerSubmit(event: React.FormEvent<HTMLFormElement>): void {
    event.preventDefault()
    setComposerError(null)

    const trimmedContent = composerText.trim()
    if (editingTarget) {
      if (trimmedContent === '') {
        setComposerError('수정 메시지는 비어 있을 수 없습니다.')
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
    sendMessageMutation.isPending || editMessageMutation.isPending || deleteMessageMutation.isPending || isSubmitting

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
                    onContextMenu={(event) => handleOpenContextMenu(event, details)}
                  >
                    <div className={styles.messageBubble}>
                      <div className={styles.timelineMeta}>
                        <span className={styles.sender}>{details.sender.username}</span>
                        <span className={styles.timelineMetaRight}>
                          <span className={styles.timestamp}>{formatMessageTimestamp(details.message)}</span>
                          {!details.message.deleted && details.message.edited ? (
                            <span className={styles.editedBadge}>[수정됨]</span>
                          ) : null}
                          {details.message.sender_id === user?.id && !details.message.deleted ? (
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

          <form className={styles.composerForm} onSubmit={handleComposerSubmit}>
            {editingTarget ? (
              <div className={styles.editingBanner}>
                <span>Editing message</span>
                <button className={styles.editingCancelButton} onClick={cancelEdit} type="button">
                  Cancel
                </button>
              </div>
            ) : null}

            <textarea
              className={styles.composerInput}
              disabled={disableComposerActions}
              onChange={(event) => setComposerText(event.target.value)}
              placeholder={editingTarget ? 'Edit message...' : 'Type a message...'}
              rows={2}
              value={composerText}
            />

            <div className={styles.composerControls}>
              <label
                className={`${styles.attachLabel}${editingTarget ? ` ${styles.attachLabelDisabled}` : ''}`}
              >
                <input
                  className={styles.attachInput}
                  disabled={disableComposerActions || Boolean(editingTarget)}
                  onChange={handleSelectAttachment}
                  ref={fileInputRef}
                  type="file"
                />
                Attach file
              </label>

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

              <button
                className={styles.submitButton}
                disabled={disableComposerActions || !hasComposerContent}
                type="submit"
              >
                {editMessageMutation.isPending
                  ? 'Saving...'
                  : sendMessageMutation.isPending
                    ? 'Sending...'
                    : editingTarget
                      ? 'Save edit'
                      : 'Send'}
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
          <button
            className={styles.contextMenuAction}
            disabled={disableComposerActions}
            onClick={() => beginEdit(actionMenuMessage)}
            type="button"
          >
            Edit
          </button>
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
