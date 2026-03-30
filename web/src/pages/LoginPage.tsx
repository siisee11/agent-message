import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { ApiError } from '../api'
import { useAuth } from '../auth'
import styles from './LoginPage.module.css'

interface LocationState {
  from?: {
    pathname?: string
  }
}

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { status, loginWithAutoRegister } = useAuth()
  const [username, setUsername] = useState('')
  const [pin, setPin] = useState('')
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
      const result = await loginWithAutoRegister({ username, pin })
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
    <section className={styles.wrapper}>
      <h2 className={styles.title}>Sign in</h2>
      <p className={styles.subtitle}>Enter your username and 4-6 digit PIN.</p>
      <form className={styles.form} onSubmit={handleSubmit}>
        <label className={styles.field}>
          <span className={styles.label}>Username</span>
          <input
            autoComplete="username"
            className={styles.input}
            disabled={isSubmitting || isLoadingAuth}
            onChange={(event) => setUsername(event.target.value)}
            placeholder="alice"
            required
            value={username}
          />
        </label>

        <label className={styles.field}>
          <span className={styles.label}>PIN</span>
          <input
            autoComplete="current-password"
            className={styles.input}
            disabled={isSubmitting || isLoadingAuth}
            inputMode="numeric"
            maxLength={6}
            minLength={4}
            onChange={(event) => setPin(event.target.value)}
            pattern="[0-9]{4,6}"
            placeholder="1234"
            required
            type="password"
            value={pin}
          />
        </label>

        <button className={styles.submit} disabled={isSubmitting || isLoadingAuth} type="submit">
          {isSubmitting ? 'Signing in...' : 'Continue'}
        </button>
      </form>

      <p className={styles.hint}>If no account exists, one will be created automatically.</p>
      {errorMessage ? <p className={styles.error}>{errorMessage}</p> : null}
      {successMessage ? <p className={styles.success}>{successMessage}</p> : null}
    </section>
  )
}

