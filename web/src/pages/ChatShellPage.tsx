import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { ApiError, type ConversationSummary, type Message, type UserProfile } from '../api'
import { apiClient } from '../api/runtime'
import { useAuth } from '../auth'
import { RealtimeProvider } from '../realtime'
import styles from './ChatShellPage.module.css'

const MESSAGE_PREVIEW_EMPTY = '대화를 시작해 보세요'
const DATE_FORMATTER = new Intl.DateTimeFormat(undefined, {
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

function summarizeLastMessage(lastMessage?: Message): string {
  if (!lastMessage) {
    return MESSAGE_PREVIEW_EMPTY
  }

  if (lastMessage.deleted) {
    return '삭제된 메시지입니다'
  }

  const content = lastMessage.content?.trim()
  const attachmentLabel =
    lastMessage.attachment_type === 'image'
      ? '[이미지]'
      : lastMessage.attachment_type === 'file'
        ? '[파일]'
        : undefined

  if (attachmentLabel && content) {
    return `${attachmentLabel} ${content}`
  }

  if (content) {
    return content
  }

  if (attachmentLabel) {
    return attachmentLabel
  }

  return MESSAGE_PREVIEW_EMPTY
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

export function ChatShellPage(): JSX.Element {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { user, logout } = useAuth()

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
      setStartDmError(resolveErrorMessage(error, '대화를 시작하지 못했습니다.'))
    },
  })

  const logoutMutation = useMutation({
    mutationFn: () => logout(),
    onSuccess: () => {
      navigate('/login', { replace: true })
    },
    onError: (error: unknown) => {
      setStartDmError(resolveErrorMessage(error, '로그아웃하지 못했습니다.'))
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

  function renderConversationItem(summary: ConversationSummary): JSX.Element {
    const preview = summarizeLastMessage(summary.last_message)
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
    <RealtimeProvider>
      <div className={styles.shell}>
        <aside className={styles.sidebar}>
          <header className={styles.sidebarHeader}>
            <div>
              <h1 className={styles.brand}>Agent Messenger</h1>
              <p className={styles.currentUser}>{user ? `@${user.username}` : 'Unknown user'}</p>
            </div>
            <button
              className={styles.logoutButton}
              disabled={logoutMutation.isPending}
              onClick={() => logoutMutation.mutate()}
              type="button"
            >
              {logoutMutation.isPending ? 'Signing out...' : 'Logout'}
            </button>
          </header>

          <section className={styles.section}>
            <h2 className={styles.sectionTitle}>Start new DM</h2>
            <form className={styles.searchForm} onSubmit={handleStartDmSubmission}>
              <input
                aria-label="Search username"
                className={styles.searchInput}
                onChange={(event) => setSearchInput(event.target.value)}
                placeholder="Search by username"
                value={searchInput}
              />
              <button
                className={styles.searchSubmit}
                disabled={normalizedSearchInput.length === 0 || startConversationMutation.isPending}
                type="submit"
              >
                {startConversationMutation.isPending ? 'Starting...' : 'Start'}
              </button>
            </form>
            {startDmError ? <p className={styles.errorMessage}>{startDmError}</p> : null}
            {hasSearchInput && userSearchQuery.isLoading ? <p className={styles.statusText}>Searching users...</p> : null}
            {hasSearchInput && userSearchQuery.isError ? (
              <p className={styles.errorMessage}>{resolveErrorMessage(userSearchQuery.error, '사용자를 찾을 수 없습니다.')}</p>
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
                        <span>@{candidate.username}</span>
                        <span>{alreadyOpen ? 'Open DM' : 'New DM'}</span>
                      </button>
                    </li>
                  )
                })}
                {userSearchResults.length === 0 ? <li className={styles.statusText}>No users found.</li> : null}
              </ul>
            ) : null}
          </section>

          <section className={styles.section}>
            <h2 className={styles.sectionTitle}>Conversations</h2>
            {conversationsQuery.isLoading ? <p className={styles.statusText}>Loading conversations...</p> : null}
            {conversationsQuery.isError ? (
              <p className={styles.errorMessage}>
                {resolveErrorMessage(conversationsQuery.error, '대화 목록을 불러오지 못했습니다.')}
              </p>
            ) : null}
            {conversationsQuery.isSuccess && conversations.length === 0 ? (
              <p className={styles.statusText}>No conversations yet. Start one above.</p>
            ) : null}
            {conversationsQuery.isSuccess && conversations.length > 0 ? (
              <nav aria-label="Conversation list" className={styles.conversationList}>
                {conversations.map(renderConversationItem)}
              </nav>
            ) : null}
          </section>
        </aside>

        <main className={styles.mainPane}>
          <Outlet />
        </main>
      </div>
    </RealtimeProvider>
  )
}
