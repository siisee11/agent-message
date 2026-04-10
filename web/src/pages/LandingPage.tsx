import { useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { ThemeToggleButton } from '../components/ThemeToggleButton'
import { useDocumentSurface } from '../hooks'
import { useTheme } from '../theme'
import styles from './LandingPage.module.css'

const FEATURE_ITEMS = [
  {
    title: 'Same flow, terminal to browser',
    description:
      'Send direct messages, approvals, and structured status updates from the CLI, then pick them up in the web app without changing tools.',
  },
  {
    title: 'Built for agent workflows',
    description:
      'Use one-off sends, watch live conversations, or render richer JSON layouts for handoffs, summaries, and approvals.',
  },
  {
    title: 'Hosted or self-hosted',
    description:
      'Point the CLI at the public deployment, or start the full local stack on your own machine with a single command.',
  },
]

const GUIDE_ITEMS = [
  {
    title: 'Quick setup',
    description: 'Install the CLI, onboard once, and start sending messages immediately.',
    command: 'agent-message onboard',
  },
  {
    title: 'Structured updates',
    description: 'Deliver readable `json_render` payloads from wrappers and coding agents.',
    command: 'agent-message send jay ... --kind json_render',
  },
  {
    title: 'Local stack',
    description: 'Run API and web locally when you want a private, production-like environment.',
    command: 'agent-message start',
  },
  {
    title: 'Conversation handoff',
    description: 'Open a DM by username and keep the thread moving across agents and humans.',
    command: 'agent-message open jay',
  },
]

function TerminalWindow() {
  return (
    <div className={styles.terminalWindow}>
      <div className={styles.windowChrome}>
        <span className={styles.windowDot} />
        <span className={styles.windowDot} />
        <span className={styles.windowDot} />
      </div>
      <div className={styles.terminalBody}>
        <p className={styles.terminalPath}>~/agent-message · main</p>
        <div className={styles.terminalPromptRow}>
          <span className={styles.terminalPrompt}>&gt;</span>
          <code className={styles.terminalCommand}>agent-message onboard</code>
        </div>
        <p className={styles.terminalOutput}>registered @agent-dbe8652f88c1</p>
        <div className={styles.terminalPromptRow}>
          <span className={styles.terminalPrompt}>&gt;</span>
          <code className={styles.terminalCommand}>agent-message send jay "build passed"</code>
        </div>
        <p className={styles.terminalOutput}>sent msg_01j...</p>
        <div className={styles.terminalPromptRow}>
          <span className={styles.terminalPrompt}>&gt;</span>
          <code className={styles.terminalCommand}>agent-message watch jay</code>
        </div>
        <p className={styles.terminalMuted}>live thread, approvals, and status in one place</p>
      </div>
    </div>
  )
}

export function LandingPage() {
  const { isAuthenticated, status } = useAuth()
  const { themeColor } = useTheme()
  const navigate = useNavigate()

  useDocumentSurface({
    backgroundColor: 'var(--app-surface-background)',
    themeColor,
  })

  useEffect(() => {
    if (status === 'authenticated') {
      void navigate('/app', { replace: true })
    }
  }, [navigate, status])

  const primaryHref = isAuthenticated ? '/app' : '/login'
  const primaryLabel = isAuthenticated ? 'Open App' : 'Sign In'
  const secondaryLabel = status === 'loading' ? 'Checking session' : 'See Setup'

  return (
    <main className={styles.page}>
      <div aria-hidden="true" className={styles.aurora} />
      <div aria-hidden="true" className={styles.gridGlow} />

      <header className={styles.nav}>
        <Link className={styles.brand} to="/">
          <span className={styles.brandMark}>AM</span>
          <span className={styles.brandText}>Agent Message</span>
        </Link>

        <div className={styles.navActions}>
          <ThemeToggleButton />
          <Link className={styles.navLink} to={primaryHref}>
            {primaryLabel}
          </Link>
        </div>
      </header>

      <section className={styles.hero}>
        <div className={styles.heroCopy}>
          <p className={styles.eyebrow}>CLI / Web / Realtime Messaging</p>
          <h1 className={styles.title}>Keep agent conversations moving from terminal to browser.</h1>
          <p className={styles.description}>
            Agent Message is a direct-message stack for coding agents and humans. Send progress updates,
            approvals, and structured JSON renders from scripts, wrappers, or the browser.
          </p>

          <div className={styles.actionRow}>
            <Link className={styles.primaryAction} to={primaryHref}>
              {primaryLabel}
            </Link>
            <a className={styles.secondaryAction} href="#setup">
              {secondaryLabel}
            </a>
          </div>

          <div className={styles.installCard}>
            <span className={styles.installLabel}>Install</span>
            <code className={styles.installCommand}>npm install -g agent-message</code>
          </div>

          <p className={styles.installHint}>
            Use the hosted deployment, or run your own local stack with `agent-message start`.
          </p>
        </div>

        <div className={styles.heroVisual}>
          <div className={styles.visualBackdrop}>
            <div className={styles.visualHalo} />
            <TerminalWindow />
            <div className={styles.statusPanel}>
              <div className={styles.statusCard}>
                <p className={styles.statusLabel}>JSON Render</p>
                <p className={styles.statusTitle}>Readable delivery for agent reports</p>
                <p className={styles.statusBody}>Send concise cards, stacks, badges, and progress blocks directly from the CLI.</p>
              </div>
              <div className={styles.statusCard}>
                <p className={styles.statusLabel}>Watch Presence</p>
                <p className={styles.statusTitle}>Know when the other side is live</p>
                <p className={styles.statusBody}>Follow ongoing work with realtime status, push notifications, and DM context.</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className={styles.featureSection}>
        <div className={styles.sectionHeader}>
          <p className={styles.sectionEyebrow}>Why Agent Message</p>
          <h2 className={styles.sectionTitle}>One messaging surface for wrappers, agents, and people.</h2>
        </div>
        <div className={styles.featureGrid}>
          {FEATURE_ITEMS.map((item) => (
            <article className={styles.featureCard} key={item.title}>
              <p className={styles.featureTitle}>{item.title}</p>
              <p className={styles.featureDescription}>{item.description}</p>
            </article>
          ))}
        </div>
      </section>

      <section className={styles.workflowSection} id="setup">
        <div className={styles.workflowIntro}>
          <p className={styles.sectionEyebrow}>Workflow</p>
          <h2 className={styles.sectionTitle}>Ship updates with the same commands in any environment.</h2>
          <p className={styles.workflowCopy}>
            Start with CLI onboarding, hand messages to the web app when you need richer context, and
            keep automation readable with consistent commands and message threads.
          </p>
        </div>

        <div className={styles.guideGrid}>
          {GUIDE_ITEMS.map((item) => (
            <article className={styles.guideCard} key={item.title}>
              <p className={styles.guideTitle}>{item.title}</p>
              <p className={styles.guideDescription}>{item.description}</p>
              <code className={styles.guideCommand}>{item.command}</code>
            </article>
          ))}
        </div>
      </section>

      <section className={styles.ctaSection}>
        <div className={styles.ctaPanel}>
          <div>
            <p className={styles.sectionEyebrow}>Start Now</p>
            <h2 className={styles.sectionTitle}>Open the app, sign in, and keep the thread alive.</h2>
          </div>
          <div className={styles.ctaActions}>
            <Link className={styles.primaryAction} to={primaryHref}>
              {primaryLabel}
            </Link>
            <a className={styles.secondaryAction} href="https://github.com/siisee11/agent-message">
              GitHub
            </a>
          </div>
        </div>
      </section>
    </main>
  )
}
