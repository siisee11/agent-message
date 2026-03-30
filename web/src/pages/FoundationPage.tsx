import styles from './Page.module.css'

export function FoundationPage() {
  return (
    <section className={styles.card}>
      <h2>Web Client Foundation</h2>
      <p>
        Phase 4 scaffolding is in place. Next milestones will add API client, auth, protected routing,
        and WebSocket integration.
      </p>
    </section>
  )
}
