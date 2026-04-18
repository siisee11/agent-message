import { useMutation } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../api'
import { apiClient } from '../api/runtime'
import { useAuth } from '../auth'
import { ThemeToggleButton } from '../components/ThemeToggleButton'
import { useDocumentSurface } from '../hooks'
import { useTheme } from '../theme'
import styles from './ProfilePage.module.css'

const DATE_FORMATTER = new Intl.DateTimeFormat('en-US', {
  year: 'numeric',
  month: 'short',
  day: 'numeric',
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

function BackIcon() {
  return (
    <svg aria-hidden="true" fill="none" height="16" viewBox="0 0 24 24" width="16">
      <path
        d="M14.5 5.5 8 12l6.5 6.5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
      />
    </svg>
  )
}

function LogoutIcon() {
  return (
    <svg aria-hidden="true" fill="none" height="16" viewBox="0 0 24 24" width="16">
      <path
        d="M14 4h-4.75A2.25 2.25 0 0 0 7 6.25v11.5A2.25 2.25 0 0 0 9.25 20H14"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="1.8"
      />
      <path
        d="M10.5 12h9M16.5 8.5 20 12l-3.5 3.5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.8"
      />
    </svg>
  )
}

export function ProfilePage() {
  const { themeColor } = useTheme()

  useDocumentSurface({
    backgroundColor: 'var(--app-surface-background)',
    themeColor,
  })

  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [pageError, setPageError] = useState<string | null>(null)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)

  const memberSinceLabel = useMemo(() => {
    if (!user) {
      return 'Unavailable'
    }

    const parsed = Date.parse(user.created_at)
    if (Number.isNaN(parsed)) {
      return user.created_at
    }
    return DATE_FORMATTER.format(new Date(parsed))
  }, [user])

  const passwordMutation = useMutation({
    mutationFn: async () => {
      await apiClient.updatePassword({
        current_password: currentPassword,
        new_password: newPassword,
      })
    },
    onSuccess: () => {
      setPageError(null)
      setSuccessMessage('Password updated.')
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
    },
    onError: (error: unknown) => {
      setSuccessMessage(null)
      setPageError(resolveErrorMessage(error, 'Failed to update password.'))
    },
  })

  const logoutMutation = useMutation({
    mutationFn: () => logout(),
    onSuccess: () => {
      navigate('/login', { replace: true })
    },
    onError: (error: unknown) => {
      setSuccessMessage(null)
      setPageError(resolveErrorMessage(error, 'Failed to sign out.'))
    },
  })

  function handleBack(): void {
    if (window.history.length > 1) {
      navigate(-1)
      return
    }
    navigate('/app')
  }

  function handlePasswordSubmit(event: React.FormEvent<HTMLFormElement>): void {
    event.preventDefault()
    setPageError(null)
    setSuccessMessage(null)

    if (newPassword !== confirmPassword) {
      setPageError('New passwords do not match.')
      return
    }

    passwordMutation.mutate()
  }

  return (
    <main className={styles.page}>
      <section className={styles.shell}>
        <header className={styles.header}>
          <button className={styles.backButton} onClick={handleBack} type="button">
            <BackIcon />
            <span>Back</span>
          </button>
          <ThemeToggleButton />
        </header>

        <section className={styles.hero}>
          <p className={styles.eyebrow}>Profile</p>
          <h1 className={styles.title}>{user ? `@${user.username}` : 'Account'}</h1>
          <p className={styles.subtitle}>Manage account details, security, and your current session.</p>
        </section>

        <section className={styles.card}>
          <h2 className={styles.sectionTitle}>Account</h2>
          <dl className={styles.infoGrid}>
            <div className={styles.infoItem}>
              <dt className={styles.infoLabel}>Username</dt>
              <dd className={styles.infoValue}>{user ? `@${user.username}` : 'Unavailable'}</dd>
            </div>
            <div className={styles.infoItem}>
              <dt className={styles.infoLabel}>Account ID</dt>
              <dd className={`${styles.infoValue} ${styles.infoCode}`}>{user?.account_id ?? 'Unavailable'}</dd>
            </div>
            <div className={styles.infoItem}>
              <dt className={styles.infoLabel}>Member since</dt>
              <dd className={styles.infoValue}>{memberSinceLabel}</dd>
            </div>
          </dl>
        </section>

        <section className={styles.card}>
          <div className={styles.sectionHeader}>
            <div>
              <h2 className={styles.sectionTitle}>Password</h2>
              <p className={styles.sectionCopy}>Use your current password to set a new one.</p>
            </div>
          </div>

          <form className={styles.form} onSubmit={handlePasswordSubmit}>
            <label className={styles.field}>
              <span className={styles.label}>Current password</span>
              <input
                autoComplete="current-password"
                className={styles.input}
                disabled={passwordMutation.isPending || logoutMutation.isPending}
                minLength={4}
                onChange={(event) => setCurrentPassword(event.target.value)}
                required
                type="password"
                value={currentPassword}
              />
            </label>

            <label className={styles.field}>
              <span className={styles.label}>New password</span>
              <input
                autoComplete="new-password"
                className={styles.input}
                disabled={passwordMutation.isPending || logoutMutation.isPending}
                minLength={4}
                onChange={(event) => setNewPassword(event.target.value)}
                required
                type="password"
                value={newPassword}
              />
            </label>

            <label className={styles.field}>
              <span className={styles.label}>Confirm new password</span>
              <input
                autoComplete="new-password"
                className={styles.input}
                disabled={passwordMutation.isPending || logoutMutation.isPending}
                minLength={4}
                onChange={(event) => setConfirmPassword(event.target.value)}
                required
                type="password"
                value={confirmPassword}
              />
            </label>

            <button
              className={styles.primaryButton}
              disabled={passwordMutation.isPending || logoutMutation.isPending}
              type="submit"
            >
              {passwordMutation.isPending ? 'Updating password...' : 'Update password'}
            </button>
          </form>
        </section>

        <section className={styles.card}>
          <div className={styles.sectionHeader}>
            <div>
              <h2 className={styles.sectionTitle}>Session</h2>
              <p className={styles.sectionCopy}>Sign out on this device.</p>
            </div>
          </div>

          <button
            className={styles.logoutButton}
            disabled={logoutMutation.isPending || passwordMutation.isPending}
            onClick={() => logoutMutation.mutate()}
            type="button"
          >
            <LogoutIcon />
            <span>{logoutMutation.isPending ? 'Signing out...' : 'Logout'}</span>
          </button>
        </section>

        {pageError ? <p className={styles.error}>{pageError}</p> : null}
        {successMessage ? <p className={styles.success}>{successMessage}</p> : null}
      </section>
    </main>
  )
}
