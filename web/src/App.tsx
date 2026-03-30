import { Route, Routes } from 'react-router-dom'
import { ChatIndexPage } from './pages/ChatIndexPage'
import { ChatShellPage } from './pages/ChatShellPage'
import { DmConversationPage } from './pages/DmConversationPage'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { ProtectedRoute } from './routes'
import styles from './App.module.css'

export function App(): JSX.Element {
  return (
    <div className={styles.app}>
      <Routes>
        <Route element={<LoginPage />} path="/login" />
        <Route element={<ProtectedRoute />}>
          <Route element={<ChatShellPage />}>
            <Route element={<ChatIndexPage />} index />
            <Route element={<DmConversationPage />} path="dm/:conversationId" />
          </Route>
        </Route>
        <Route element={<NotFoundPage />} path="*" />
      </Routes>
    </div>
  )
}
