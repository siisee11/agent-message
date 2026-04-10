import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { ApiError } from '../api'
import { useAuth } from '../auth'
import { ThemeToggleButton } from '../components/ThemeToggleButton'
import { useDocumentSurface } from '../hooks'
import { useTheme } from '../theme'
import styles from './LoginPage.module.css'

interface LocationState {
  from?: {
    pathname?: string
  }
}

export function LoginPage() {
  const { themeColor } = useTheme()

  useDocumentSurface({
    backgroundColor: 'var(--app-surface-background)',
    themeColor,
  })

  const navigate = useNavigate()
  const location = useLocation()
  const { status, loginWithAutoRegister } = useAuth()
  const [accountId, setAccountId] = useState('')
  const [password, setPassword] = useState('')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const redirectPath = useMemo(() => {
    const state = location.state as LocationState | null
    const candidate = state?.from?.pathname
    if (!candidate || candidate === '/login') {
      return '/'
    }
    return candidate
  }, [location.state])

  useEffect(() => {
    if (status === 'authenticated') {
      void navigate(redirectPath, { replace: true })
    }
  }, [navigate, redirectPath, status])

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setErrorMessage(null)
    setSuccessMessage(null)
    setIsSubmitting(true)

    try {
      const result = await loginWithAutoRegister({ accountId, password })
      if (result.mode === 'register') {
        setSuccessMessage('Account created and signed in.')
      }
    } catch (error: unknown) {
      if (error instanceof ApiError) {
        setErrorMessage(error.message)
      } else if (error instanceof Error) {
        setErrorMessage(error.message)
      } else {
        setErrorMessage('Failed to sign in.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  const isLoadingAuth = status === 'loading'

  return (
    <section className={styles.page}>
      <div className={styles.wrapper}>
        <div className={styles.wrapperHeader}>
          <h2 className={styles.title}>Sign in</h2>
          <ThemeToggleButton />
        </div>
        <p className={styles.subtitle}>Enter your account ID and password.</p>
        <form className={styles.form} onSubmit={handleSubmit}>
          <label className={styles.field}>
            <span className={styles.label}>Account ID</span>
            <input
              autoComplete="username"
              className={styles.input}
              disabled={isSubmitting || isLoadingAuth}
              onChange={(event) => setAccountId(event.target.value)}
              placeholder="cva-chat_jay"
              required
              value={accountId}
            />
          </label>

          <label className={styles.field}>
            <span className={styles.label}>Password</span>
            <input
              autoComplete="current-password"
              className={styles.input}
              disabled={isSubmitting || isLoadingAuth}
              minLength={4}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="password"
              required
              type="password"
              value={password}
            />
          </label>

          <button className={styles.submit} disabled={isSubmitting || isLoadingAuth} type="submit">
            {isSubmitting ? 'Signing in...' : 'Continue'}
          </button>
        </form>

        <p className={styles.hint}>If no account exists, one will be created automatically.</p>
        {errorMessage ? <p className={styles.error}>{errorMessage}</p> : null}
        {successMessage ? <p className={styles.success}>{successMessage}</p> : null}
      </div>
    </section>
  )
}
