import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { NavLink, useNavigate } from 'react-router-dom'
import { ApiError, type ConversationSummary, type Message, type UserProfile } from '../api'
import { apiClient } from '../api/runtime'
import { useAuth } from '../auth'
import { summarizeLastMessagePreview } from '../messages/messagePresentation'
import { useRealtime } from '../realtime'
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

function filterSearchResults(results: UserProfile[] | undefined, currentUserId: string | undefined): UserProfile[] {
  if (!results) {
    return []
  }
  if (!currentUserId) {
    return results
  }
  return results.filter((candidate) => candidate.id !== currentUserId)
}

function isConversationWithUser(summary: ConversationSummary, userId: string): boolean {
  return summary.other_user.id === userId
}

export function ChatShellPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { user, logout } = useAuth()
  const realtime = useRealtime()

  const [searchInput, setSearchInput] = useState('')
  const [startDmError, setStartDmError] = useState<string | null>(null)

  const normalizedSearchInput = searchInput.trim()

  const conversationsQuery = useQuery({
    queryKey: ['conversations'],
    queryFn: () => apiClient.listConversations(),
  })

  const userSearchQuery = useQuery({
    queryKey: ['users', 'search', normalizedSearchInput],
    queryFn: () =>
      apiClient.searchUsers({
        username: normalizedSearchInput,
        limit: 8,
      }),
    enabled: normalizedSearchInput.length > 0,
  })

  const userSearchResults = useMemo(
    () => filterSearchResults(userSearchQuery.data, user?.id),
    [user?.id, userSearchQuery.data],
  )

  const startConversationMutation = useMutation({
    mutationFn: (username: string) => apiClient.startConversation({ username }),
    onSuccess: async (conversationDetails) => {
      setStartDmError(null)
      setSearchInput('')
      await queryClient.invalidateQueries({ queryKey: ['conversations'] })
      navigate(`/dm/${conversationDetails.conversation.id}`)
    },
    onError: (error: unknown) => {
      setStartDmError(resolveErrorMessage(error, 'Failed to start the conversation.'))
    },
  })

  const logoutMutation = useMutation({
    mutationFn: () => logout(),
    onSuccess: () => {
      navigate('/login', { replace: true })
    },
    onError: (error: unknown) => {
      setStartDmError(resolveErrorMessage(error, 'Failed to sign out.'))
    },
  })

  function handleStartDmSubmission(event: React.FormEvent<HTMLFormElement>): void {
    event.preventDefault()
    const username = normalizedSearchInput
    if (username === '') {
      return
    }

    setStartDmError(null)
    startConversationMutation.mutate(username)
  }

  function handleStartDmWithCandidate(username: string): void {
    setStartDmError(null)
    startConversationMutation.mutate(username)
  }

  function renderConversationItem(summary: ConversationSummary) {
    const preview = summarizeLastMessagePreview(summary.last_message)
    const timestamp = formatLastMessageTime(summary.last_message)
    const conversationId = summary.conversation.id

    return (
      <NavLink
        className={({ isActive }) =>
          `${styles.conversationItem}${isActive ? ` ${styles.conversationItemActive}` : ''}`
        }
        key={summary.conversation.id}
        to={`/dm/${summary.conversation.id}`}
      >
        <div className={styles.conversationMeta}>
          <span className={styles.conversationName}>{summary.other_user.username}</span>
          <span className={styles.conversationTime}>{timestamp}</span>
        </div>
        <p className={styles.conversationPreview} title={preview}>
          {preview}
        </p>
        <span className={styles.conversationId}>{conversationId}</span>
      </NavLink>
    )
  }

  const conversations = conversationsQuery.data ?? []
  const hasSearchInput = normalizedSearchInput.length > 0

  return (
    <main className={styles.page}>
      <section className={styles.shell}>
        <header className={styles.header}>
          <div className={styles.headerTop}>
            <div>
              <p className={styles.eyebrow}>Messages</p>
              <h1 className={styles.brand}>Agent Messenger</h1>
            </div>
            <button
              className={styles.logoutButton}
              disabled={logoutMutation.isPending}
              onClick={() => logoutMutation.mutate()}
              type="button"
            >
              {logoutMutation.isPending ? 'Signing out...' : 'Logout'}
            </button>
          </div>
          <div className={styles.headerMeta}>
            <p className={styles.currentUser}>{user ? `@${user.username}` : 'Unknown user'}</p>
            <span className={styles.statusBadge}>
              {realtime.status === 'open'
                ? 'Live'
                : realtime.status === 'connecting'
                  ? 'Connecting'
                  : 'Offline'}
            </span>
          </div>
        </header>

        <section className={styles.composerSection}>
          <div className={styles.sectionHeading}>
            <h2 className={styles.sectionTitle}>Start a conversation</h2>
            <p className={styles.sectionCopy}>Enter a username to open a DM right away.</p>
          </div>
          <form className={styles.searchForm} onSubmit={handleStartDmSubmission}>
            <input
              aria-label="Search username"
              className={styles.searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder="@username"
              value={searchInput}
            />
            <button
              className={styles.searchSubmit}
              disabled={normalizedSearchInput.length === 0 || startConversationMutation.isPending}
              type="submit"
            >
              {startConversationMutation.isPending ? 'Opening...' : 'Open'}
            </button>
          </form>
          {startDmError ? <p className={styles.errorMessage}>{startDmError}</p> : null}
          {hasSearchInput && userSearchQuery.isLoading ? <p className={styles.statusText}>Searching users...</p> : null}
          {hasSearchInput && userSearchQuery.isError ? (
            <p className={styles.errorMessage}>
              {resolveErrorMessage(userSearchQuery.error, 'Failed to find users.')}
            </p>
          ) : null}
          {hasSearchInput && userSearchQuery.isSuccess ? (
            <ul className={styles.searchResults}>
              {userSearchResults.map((candidate) => {
                const alreadyOpen = conversations.some((summary) => isConversationWithUser(summary, candidate.id))
                return (
                  <li key={candidate.id}>
                    <button
                      className={styles.searchResultButton}
                      onClick={() => handleStartDmWithCandidate(candidate.username)}
                      type="button"
                    >
                      <span className={styles.searchResultName}>@{candidate.username}</span>
                      <span className={styles.searchResultAction}>{alreadyOpen ? 'Open chat' : 'New chat'}</span>
                    </button>
                  </li>
                )
              })}
              {userSearchResults.length === 0 ? <li className={styles.statusText}>No users found.</li> : null}
            </ul>
          ) : null}
        </section>

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
              <p className={styles.emptyTitle}>No conversations yet</p>
              <p className={styles.emptyCopy}>Search for a username above to start your first DM.</p>
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
