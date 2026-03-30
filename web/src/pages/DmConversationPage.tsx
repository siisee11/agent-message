import { useParams } from 'react-router-dom'
import styles from './ChatViewPage.module.css'

export function DmConversationPage(): JSX.Element {
  const { conversationId } = useParams()

  return (
    <section className={styles.page}>
      <div className={styles.card}>
        <h2 className={styles.title}>Conversation</h2>
        <p className={styles.copy}>Active DM: {conversationId}</p>
        <p className={styles.hint}>Message history and timeline behavior will be wired in the next milestone.</p>
      </div>
    </section>
  )
}
