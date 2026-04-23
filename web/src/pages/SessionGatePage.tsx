import { Navigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { BrandLogo } from '../components/BrandLogo'
import { useDocumentSurface } from '../hooks'
import { useTheme } from '../theme'
import { LandingPage } from './LandingPage'
import styles from './SessionGatePage.module.css'

const shouldRedirectUnauthenticatedRootToLogin = import.meta.env.VITE_AGENT_MESSAGE_SELFHOST === '1'

export function SessionGatePage() {
  const { status } = useAuth()
  const { themeColor } = useTheme()

  useDocumentSurface({
    backgroundColor: 'var(--app-surface-background)',
    themeColor,
  })

  if (status === 'authenticated') {
    return <Navigate replace to="/app" />
  }

  if (status === 'loading') {
    return (
      <main className={styles.page}>
        <div className={styles.panel}>
          <BrandLogo className={styles.brand} size="sm" />
          <h1 className={styles.title}>Restoring your session</h1>
          <p className={styles.description}>Checking local credentials before choosing where to send you.</p>
        </div>
      </main>
    )
  }

  if (shouldRedirectUnauthenticatedRootToLogin) {
    return <Navigate replace to="/login" />
  }

  return <LandingPage />
}
