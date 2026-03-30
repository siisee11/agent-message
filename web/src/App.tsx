import { Link, Route, Routes } from 'react-router-dom'
import { FoundationPage } from './pages/FoundationPage'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import styles from './App.module.css'

export function App() {
  return (
    <div className={styles.layout}>
      <header className={styles.header}>
        <h1 className={styles.title}>Agent Messenger</h1>
        <nav aria-label="Primary">
          <Link className={styles.navLink} to="/">
            Home
          </Link>
          {' · '}
          <Link className={styles.navLink} to="/login">
            Login
          </Link>
        </nav>
      </header>

      <main className={styles.main}>
        <Routes>
          <Route element={<LoginPage />} path="/login" />
          <Route element={<FoundationPage />} path="/" />
          <Route element={<NotFoundPage />} path="*" />
        </Routes>
      </main>
    </div>
  )
}
