import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { ChatShellPage } from './pages/ChatShellPage'
import { DmConversationPage } from './pages/DmConversationPage'
import { LandingPage } from './pages/LandingPage'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { ProfilePage } from './pages/ProfilePage'
import { RealtimeProvider } from './realtime'
import { ProtectedRoute } from './routes'
import styles from './App.module.css'

export function App() {
  const location = useLocation()
  const isFixedShellRoute =
    location.pathname.startsWith('/chats') || location.pathname.startsWith('/dm/')

  return (
    <div className={`${styles.app} ${isFixedShellRoute ? styles.appShell : styles.appDocument}`}>
      <RealtimeProvider>
        <Routes>
          <Route element={<LandingPage />} path="/" />
          <Route element={<LoginPage />} path="/login" />
          <Route element={<Navigate replace to="/chats" />} path="/app" />
          <Route element={<ProtectedRoute />}>
            <Route element={<ChatShellPage />} path="/chats" />
            <Route element={<DmConversationPage />} path="/dm/:conversationId" />
            <Route element={<ProfilePage />} path="/profile" />
          </Route>
          <Route element={<NotFoundPage />} path="*" />
        </Routes>
      </RealtimeProvider>
    </div>
  )
}
