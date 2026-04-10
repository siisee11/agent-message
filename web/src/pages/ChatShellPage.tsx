import { useEffect, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { NavLink, useNavigate } from 'react-router-dom'
import { ApiError, type ConversationSummary, type Message } from '../api'
import { apiClient } from '../api/runtime'
import { useAuth } from '../auth'
import { ThemeToggleButton } from '../components/ThemeToggleButton'
import { useDocumentSurface } from '../hooks'
import { summarizeLastMessagePreview } from '../messages/messagePresentation'
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

export function ChatShellPage() {
  const { themeColor } = useTheme()

  useDocumentSurface({
    backgroundColor: 'var(--app-surface-background)',
    themeColor,
  })

  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const realtime = useRealtime()

  const [pageError, setPageError] = useState<string | null>(null)
  const [pushState, setPushState] = useState<PushState>({
    supported: false,
    configured: false,
    enabled: false,
    permission: 'unsupported',
  })
  const [pushError, setPushError] = useState<string | null>(null)
  const [pushLoading, setPushLoading] = useState(true)

  const conversationsQuery = useQuery({
    queryKey: ['conversations'],
    queryFn: () => apiClient.listConversations(),
  })

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

  function renderConversationItem(summary: ConversationSummary) {
    const preview = summarizeLastMessagePreview(summary.last_message)
    const timestamp = formatLastMessageTime(summary.last_message)
    const title = summary.conversation.title?.trim() ?? ''

    return (
      <NavLink
        className={({ isActive }) =>
          `${styles.conversationItem}${isActive ? ` ${styles.conversationItemActive}` : ''}`
        }
        key={summary.conversation.id}
        to={`/dm/${summary.conversation.id}`}
      >
        {title ? (
          <p className={styles.conversationTitle} title={title}>
            {title}
          </p>
        ) : null}
        <div className={styles.conversationMeta}>
          <span className={styles.conversationName}>{summary.other_user.username}</span>
          <span className={styles.conversationTime}>{timestamp}</span>
        </div>
        <p className={styles.conversationPreview} title={preview}>
          {preview}
        </p>
      </NavLink>
    )
  }

  const conversations = conversationsQuery.data ?? []

  return (
    <main className={styles.page}>
      <section className={styles.shell}>
        <header className={styles.header}>
          <div className={styles.headerTop}>
            <div>
              <p className={styles.eyebrow}>Messages</p>
              <h1 className={styles.brand}>Agent Message</h1>
            </div>
            <div className={styles.headerActions}>
              <ThemeToggleButton />
              <button
                className={styles.logoutButton}
                disabled={logoutMutation.isPending}
                onClick={() => logoutMutation.mutate()}
                type="button"
              >
                {logoutMutation.isPending ? 'Signing out...' : 'Logout'}
              </button>
            </div>
          </div>
          <div className={styles.headerMeta}>
            <p className={styles.currentUser}>{user ? `@${user.username}` : 'Unknown user'}</p>
            <span className={styles.statusBadge}>{formatRealtimeStatusLabel(realtime.status)}</span>
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
