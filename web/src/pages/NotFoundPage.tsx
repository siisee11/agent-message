import { Link } from 'react-router-dom'
import styles from './Page.module.css'

export function NotFoundPage() {
  return (
    <section className={styles.card}>
      <h2>Page not found</h2>
      <p>The requested route does not exist.</p>
      <Link className={styles.link} to="/">
        Back to home
      </Link>
    </section>
  )
}
