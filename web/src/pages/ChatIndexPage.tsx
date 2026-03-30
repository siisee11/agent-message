import styles from './ChatViewPage.module.css'

export function ChatIndexPage(): JSX.Element {
  return (
    <section className={styles.page}>
      <div className={styles.card}>
        <h2 className={styles.title}>Choose a conversation</h2>
        <p className={styles.copy}>Select a DM from the sidebar, or start a new conversation by username.</p>
      </div>
    </section>
  )
}
