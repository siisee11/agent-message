import { Route, Routes } from 'react-router-dom'
import { ChatShellPage } from './pages/ChatShellPage'
import { DmConversationPage } from './pages/DmConversationPage'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { RealtimeProvider } from './realtime'
import { ProtectedRoute } from './routes'
import { useTheme } from './theme'
import styles from './App.module.css'

export function App() {
  const { resolvedTheme, toggleTheme } = useTheme()

  return (
    <div className={styles.app}>
      <button
        aria-label={resolvedTheme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        className={styles.themeToggle}
        onClick={toggleTheme}
        type="button"
      >
        {resolvedTheme === 'dark' ? 'Light' : 'Dark'}
      </button>
      <RealtimeProvider>
        <Routes>
          <Route element={<LoginPage />} path="/login" />
          <Route element={<ProtectedRoute />}>
            <Route element={<ChatShellPage />} path="/" />
            <Route element={<DmConversationPage />} path="/dm/:conversationId" />
          </Route>
          <Route element={<NotFoundPage />} path="*" />
        </Routes>
      </RealtimeProvider>
    </div>
  )
}
