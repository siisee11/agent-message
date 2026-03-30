import { Route, Routes } from 'react-router-dom'
import { ChatShellPage } from './pages/ChatShellPage'
import { DmConversationPage } from './pages/DmConversationPage'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { RealtimeProvider } from './realtime'
import { ProtectedRoute } from './routes'
import styles from './App.module.css'

export function App() {
  return (
    <div className={styles.app}>
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
