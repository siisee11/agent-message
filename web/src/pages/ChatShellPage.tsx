import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { NavLink, useLocation, useNavigate } from 'react-router-dom'
import { ApiError, type ConversationSummary, type Message } from '../api'
import { apiClient } from '../api/runtime'
import { useAuth } from '../auth'
import { BrandLogo } from '../components/BrandLogo'
import { ChatAvatar } from '../components/ChatAvatar'
import { ThemeToggleButton } from '../components/ThemeToggleButton'
import { useDocumentSurface } from '../hooks'
import { summarizeConversationLabel, summarizeLastMessagePreview } from '../messages/messagePresentation'
import {
  disablePushNotifications,
  enablePushNotifications,
  getPushState,
  type PushState,
} from '../notifications/push'
import { formatRealtimeStatusLabel, useRealtime } from '../realtime'
import { useTheme } from '../theme'
import styles from './ChatShellPage.module.css'

const DATE_FORMATTER = new Intl.DateTimeFormat('en-US', {
  month: 'short',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    return error.message
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message
  }
  return fallback
}

function formatLastMessageTime(lastMessage?: Message): string {
  if (!lastMessage) {
    return ''
  }

  const parsed = Date.parse(lastMessage.created_at)
  if (Number.isNaN(parsed)) {
    return ''
  }

  return DATE_FORMATTER.format(new Date(parsed))
}

function getRealtimeStatusBadgeClass(status: ReturnType<typeof useRealtime>['status']): string {
  if (status === 'open') {
    return styles.statusBadgeLive
  }
  if (status === 'connecting') {
    return styles.statusBadgeConnecting
  }
  return styles.statusBadgeOffline
}

function LogoutIcon() {
  return (
    <svg aria-hidden="true" className={styles.logoutIcon} viewBox="0 0 24 24">
      <path
        d="M14 4h-4.75A2.25 2.25 0 0 0 7 6.25v11.5A2.25 2.25 0 0 0 9.25 20H14"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="1.8"
      />
      <path
        d="M10.5 12h9M16.5 8.5 20 12l-3.5 3.5"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.8"
      />
    </svg>
  )
}

function MoreIcon() {
  return (
    <svg aria-hidden="true" className={styles.conversationMenuIcon} viewBox="0 0 24 24">
      <circle cx="5" cy="12" r="1.8" fill="currentColor" />
      <circle cx="12" cy="12" r="1.8" fill="currentColor" />
      <circle cx="19" cy="12" r="1.8" fill="currentColor" />
    </svg>
  )
}

export function ChatShellPage() {
  const { themeColor } = useTheme()

  useDocumentSurface({
    backgroundColor: 'var(--app-surface-background)',
    themeColor,
  })

  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const realtime = useRealtime()
  const queryClient = useQueryClient()

  const [pageError, setPageError] = useState<string | null>(null)
  const [pushState, setPushState] = useState<PushState>({
    supported: false,
    configured: false,
    enabled: false,
    permission: 'unsupported',
  })
  const [pushError, setPushError] = useState<string | null>(null)
  const [pushLoading, setPushLoading] = useState(true)
  const [openConversationMenuId, setOpenConversationMenuId] = useState<string | null>(null)

  const conversationMenuRef = useRef<HTMLDivElement | null>(null)

  const conversationsQuery = useQuery({
    queryKey: ['conversations'],
    queryFn: () => apiClient.listConversations(),
  })
  const conversations = conversationsQuery.data ?? []
  const activeConversationId = location.pathname.startsWith('/dm/')
    ? location.pathname.slice('/dm/'.length)
    : null

  const logoutMutation = useMutation({
    mutationFn: () => logout(),
    onSuccess: () => {
      navigate('/login', { replace: true })
    },
    onError: (error: unknown) => {
      setPageError(resolveErrorMessage(error, 'Failed to sign out.'))
    },
  })

  const pushMutation = useMutation({
    mutationFn: async () => {
      if (pushState.enabled) {
        await disablePushNotifications()
        return getPushState()
      }
      return enablePushNotifications()
    },
    onSuccess: (nextPushState) => {
      setPageError(null)
      setPushError(null)
      setPushState(nextPushState)
    },
    onError: (error: unknown) => {
      setPushError(resolveErrorMessage(error, 'Failed to update notification settings.'))
    },
  })

  const deleteConversationMutation = useMutation({
    mutationFn: async (conversationId: string) => {
      await apiClient.deleteConversation(conversationId)
    },
    onSuccess: async (_data, deletedConversationId) => {
      setPageError(null)
      setOpenConversationMenuId((current) => (current === deletedConversationId ? null : current))
      queryClient.removeQueries({ queryKey: ['conversation', deletedConversationId] })
      queryClient.removeQueries({ queryKey: ['messages', deletedConversationId] })
      await queryClient.invalidateQueries({ queryKey: ['conversations'] })
      if (activeConversationId === deletedConversationId) {
        void navigate('/app', { replace: true })
      }
    },
    onError: (error: unknown) => {
      setPageError(resolveErrorMessage(error, 'Failed to delete the conversation.'))
    },
  })

  useEffect(() => {
    let cancelled = false

    void getPushState()
      .then((state) => {
        if (cancelled) {
          return
        }
        setPushState(state)
        setPushLoading(false)
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        setPushError(resolveErrorMessage(error, 'Failed to load notification settings.'))
        setPushLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!openConversationMenuId) {
      return
    }

    const handleWindowMouseDown = (event: MouseEvent) => {
      const target = event.target as Node | null
      if (target && conversationMenuRef.current?.contains(target)) {
        return
      }
      setOpenConversationMenuId(null)
    }

    const handleWindowKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpenConversationMenuId(null)
      }
    }

    window.addEventListener('mousedown', handleWindowMouseDown)
    window.addEventListener('keydown', handleWindowKeyDown)
    return () => {
      window.removeEventListener('mousedown', handleWindowMouseDown)
      window.removeEventListener('keydown', handleWindowKeyDown)
    }
  }, [openConversationMenuId])

  useEffect(() => {
    if (!openConversationMenuId) {
      return
    }

    const menuConversationExists = conversations.some(
      (summary) => summary.conversation.id === openConversationMenuId,
    )
    if (!menuConversationExists) {
      setOpenConversationMenuId(null)
    }
  }, [conversations, openConversationMenuId])

  function resolvePushButtonLabel(): string {
    if (pushMutation.isPending || pushLoading) {
      return 'Checking...'
    }
    if (pushState.enabled) {
      return 'Disable notifications'
    }
    if (!pushState.supported) {
      return 'Notifications unavailable'
    }
    if (!pushState.configured) {
      return 'Notifications unavailable'
    }
    if (pushState.permission === 'denied') {
      return 'Notifications blocked'
    }
    return 'Enable notifications'
  }

  function resolvePushSummaryLabel(): string {
    if (pushMutation.isPending || pushLoading) {
      return 'Checking'
    }
    if (pushState.enabled) {
      return 'On'
    }
    if (!pushState.supported || !pushState.configured) {
      return 'Unavailable'
    }
    if (pushState.permission === 'denied') {
      return 'Blocked'
    }
    return 'Off'
  }

  function resolvePushStatusText(): string {
    if (pushState.enabled) {
      return 'Installed device notifications are active.'
    }
    if (!pushState.supported) {
      return 'This browser does not support push notifications.'
    }
    if (!pushState.configured) {
      return 'Push notifications are not configured on the server.'
    }
    if (pushState.permission === 'denied') {
      return 'Browser permission is blocked for this app.'
    }
    return 'Enable push alerts for new messages.'
  }

  const isPushToggleDisabled =
    pushLoading ||
    pushMutation.isPending ||
    !pushState.supported ||
    !pushState.configured ||
    (pushState.permission === 'denied' && !pushState.enabled)

  function toggleConversationMenu(conversationId: string): void {
    setOpenConversationMenuId((current) => (current === conversationId ? null : conversationId))
  }

  function handleDeleteConversation(conversationId: string): void {
    if (deleteConversationMutation.isPending) {
      return
    }

    const confirmed = window.confirm(
      'Delete this chat from your list? It will reappear if a new message arrives or you open the chat again.',
    )
    if (!confirmed) {
      return
    }

    deleteConversationMutation.mutate(conversationId)
  }

  function renderConversationItem(summary: ConversationSummary) {
    const conversationId = summary.conversation.id
    const preview = summarizeLastMessagePreview(summary.last_message)
    const timestamp = formatLastMessageTime(summary.last_message)
    const conversationLabel = summarizeConversationLabel(summary)
    const isMenuOpen = openConversationMenuId === conversationId
    const isActive = activeConversationId === conversationId
    const isDeletingConversation = deleteConversationMutation.isPending && isMenuOpen
    const hasUnread = realtime.unreadConversationIds.has(conversationId)

    return (
      <div
        className={`${styles.conversationItem}${isActive ? ` ${styles.conversationItemActive}` : ''}`}
        key={conversationId}
      >
        <NavLink
          className={styles.conversationLink}
          onClick={() => setOpenConversationMenuId(null)}
          to={`/dm/${conversationId}`}
        >
          <ChatAvatar className={styles.conversationAvatar} size="md" username={summary.other_user.username} />
          <div className={styles.conversationBody}>
            <div className={styles.conversationMeta}>
              <span className={styles.conversationNameRow}>
                <span className={styles.conversationName} title={conversationLabel}>
                  {conversationLabel}
                </span>
                {hasUnread ? <span aria-label="Unread conversation" className={styles.unreadDot} /> : null}
              </span>
              <span className={styles.conversationTime}>{timestamp}</span>
            </div>
            <p className={styles.conversationPreview} title={preview}>
              {preview}
            </p>
          </div>
        </NavLink>
        <div className={styles.conversationActions} ref={isMenuOpen ? conversationMenuRef : undefined}>
          <button
            aria-expanded={isMenuOpen}
            aria-haspopup="menu"
            aria-label={`Conversation actions for @${summary.other_user.username}`}
            className={styles.conversationMenuTrigger}
            disabled={deleteConversationMutation.isPending}
            onClick={(event) => {
              event.preventDefault()
              event.stopPropagation()
              toggleConversationMenu(conversationId)
            }}
            type="button"
          >
            <MoreIcon />
          </button>
          {isMenuOpen ? (
            <div aria-label="Conversation actions" className={styles.conversationMenu} role="menu">
              <button
                className={`${styles.conversationMenuItem} ${styles.conversationMenuItemDanger}`}
                disabled={deleteConversationMutation.isPending}
                onClick={(event) => {
                  event.preventDefault()
                  event.stopPropagation()
                  handleDeleteConversation(conversationId)
                }}
                role="menuitem"
                type="button"
              >
                {isDeletingConversation ? 'Deleting...' : 'Delete chat'}
              </button>
            </div>
          ) : null}
        </div>
      </div>
    )
  }

  const realtimeStatusBadgeClassName = `${styles.statusBadge} ${getRealtimeStatusBadgeClass(realtime.status)}`

  return (
    <main className={styles.page}>
      <section className={styles.shell}>
        <header className={styles.header}>
          <div className={styles.headerTop}>
            <div>
              <p className={styles.eyebrow}>Messages</p>
              <h1 className={styles.brand}>
                <BrandLogo size="lg" />
              </h1>
            </div>
            <div className={styles.headerActions}>
              <ThemeToggleButton />
              <button
                aria-label={logoutMutation.isPending ? 'Signing out' : 'Logout'}
                className={styles.logoutButton}
                disabled={logoutMutation.isPending}
                onClick={() => logoutMutation.mutate()}
                title={logoutMutation.isPending ? 'Signing out...' : 'Logout'}
                type="button"
              >
                <LogoutIcon />
                <span className={styles.srOnly}>{logoutMutation.isPending ? 'Signing out...' : 'Logout'}</span>
              </button>
            </div>
          </div>
          <div className={styles.headerMeta}>
            <p className={styles.currentUser}>{user ? `@${user.username}` : 'Unknown user'}</p>
            <span className={realtimeStatusBadgeClassName}>{formatRealtimeStatusLabel(realtime.status)}</span>
          </div>
          <div className={styles.notificationRow}>
            <div className={styles.notificationText} title={resolvePushStatusText()}>
              <p className={styles.notificationLabel}>Notifications</p>
              <p className={styles.notificationSummary}>{resolvePushSummaryLabel()}</p>
            </div>
            <button
              aria-checked={pushState.enabled}
              aria-label={resolvePushButtonLabel()}
              className={`${styles.pushSwitch}${pushState.enabled ? ` ${styles.pushSwitchEnabled}` : ''}`}
              disabled={isPushToggleDisabled}
              onClick={() => pushMutation.mutate()}
              role="switch"
              type="button"
            >
              <span className={styles.pushSwitchThumb} />
            </button>
          </div>
          {pageError ? <p className={styles.errorMessage}>{pageError}</p> : null}
          {pushError ? <p className={styles.errorMessage}>{pushError}</p> : null}
        </header>

        <section className={styles.listSection}>
          <div className={styles.sectionHeading}>
            <h2 className={styles.sectionTitle}>Conversations</h2>
            <p className={styles.sectionCopy}>
              {conversations.length > 0
                ? `${conversations.length} conversation${conversations.length === 1 ? '' : 's'}`
                : 'No open conversations yet.'}
            </p>
          </div>
          {conversationsQuery.isLoading ? <p className={styles.statusText}>Loading conversations...</p> : null}
          {conversationsQuery.isError ? (
            <p className={styles.errorMessage}>
              {resolveErrorMessage(conversationsQuery.error, 'Failed to load conversations.')}
            </p>
          ) : null}
          {conversationsQuery.isSuccess && conversations.length === 0 ? (
            <div className={styles.emptyState}>
              <p className={styles.emptyTitle}>No reports yet</p>
              <p className={styles.emptyCopy}>Agent work updates will appear here when a conversation is created.</p>
            </div>
          ) : null}
          {conversationsQuery.isSuccess && conversations.length > 0 ? (
            <nav aria-label="Conversation list" className={styles.conversationList}>
              {conversations.map(renderConversationItem)}
            </nav>
          ) : null}
        </section>
      </section>
    </main>
  )
}
