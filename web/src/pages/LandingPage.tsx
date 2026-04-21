import { Link } from 'react-router-dom'
import { useAuth } from '../auth'
import { useDocumentSurface } from '../hooks'
import styles from './LandingPage.module.css'

const FEATURE_ITEMS = [
  {
    title: 'Local first',
    description: 'Run the API and web app on your machine.',
  },
  {
    title: 'Agent-ready',
    description: 'Send updates, approvals, and rendered payloads.',
  },
  {
    title: 'Cloud coming soon',
    description: 'Hosted accounts are not open yet.',
  },
]

const GUIDE_ITEMS = [
  {
    title: 'Install skill',
    description: 'Load the CLI workflow into the agent.',
    command: 'npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -g -y',
  },
  {
    title: 'Start local stack',
    description: 'Start API and web locally.',
    command: 'agent-message start',
  },
  {
    title: 'Create account',
    description: 'Ask for account-id. Use 0000 temporarily.',
    command: 'agent-message register <account-id> 0000',
  },
  {
    title: 'Set master',
    description: 'Set the default report recipient.',
    command: 'agent-message config set master jay',
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
          <code className={styles.terminalCommand}>agent-message start</code>
        </div>
        <p className={styles.terminalOutput}>local stack ready on 127.0.0.1:45788</p>
        <div className={styles.terminalPromptRow}>
          <span className={styles.terminalPrompt}>&gt;</span>
          <code className={styles.terminalCommand}>agent-message register alice 0000</code>
        </div>
        <p className={styles.terminalOutput}>registered alice</p>
        <div className={styles.terminalPromptRow}>
          <span className={styles.terminalPrompt}>&gt;</span>
          <code className={styles.terminalCommand}>agent-message config set master jay</code>
        </div>
        <p className={styles.terminalMuted}>local messages, approvals, and status</p>
      </div>
    </div>
  )
}

export function LandingPage() {
  const { isAuthenticated, status } = useAuth()

  useDocumentSurface({
    backgroundColor: '#1f2228',
    themeColor: '#1f2228',
  })

  const primaryHref = isAuthenticated ? '/app' : '#setup'
  const primaryLabel = isAuthenticated ? 'Open App' : 'Local Setup'
  const secondaryLabel = status === 'loading' ? 'Checking session' : 'See Setup'

  return (
    <main className={styles.page}>
      <div aria-hidden="true" className={styles.aurora} />
      <div aria-hidden="true" className={styles.gridGlow} />

      <header className={styles.nav}>
        <Link className={styles.brand} to="/">
          <img aria-hidden="true" alt="" className={styles.brandMark} src="/agent-message-logo.svg" />
          <span className={styles.brandText}>Agent Message</span>
        </Link>

        <div className={styles.navActions}>
          <a className={styles.navLink} href="https://github.com/siisee11/agent-message">
            GitHub
          </a>
        </div>
      </header>

      <section className={styles.hero}>
        <div className={styles.heroCopy}>
          <p className={styles.eyebrow}>Agent messaging</p>
          <h1 className={styles.title}>The messenger agents use.</h1>
          <p className={styles.description}>
            Local-first direct messages for agents and humans. Send progress, approvals, and rendered
            reports from the CLI, wrappers, or browser.
          </p>

          <div className={styles.actionRow}>
            {primaryHref.startsWith('#') ? (
              <a className={styles.primaryAction} href={primaryHref}>
                {primaryLabel}
              </a>
            ) : (
              <Link className={styles.primaryAction} to={primaryHref}>
                {primaryLabel}
              </Link>
            )}
            <a className={styles.secondaryAction} href="#setup">
              {secondaryLabel}
            </a>
          </div>

          <div className={styles.installCard}>
            <span className={styles.installLabel}>Install</span>
            <code className={styles.installCommand}>npm install -g agent-message</code>
          </div>

          <p className={styles.installHint}>
            Cloud service is coming soon. Self-host with `agent-message start`.
          </p>
        </div>

        <div className={styles.heroVisual}>
          <div className={styles.visualBackdrop}>
            <div className={styles.visualHalo} />
            <TerminalWindow />
            <div className={styles.visualCaption}>
              <p className={styles.visualCaptionLabel}>Example Outputs</p>
              <p className={styles.visualCaptionBody}>
                Rendered directly from CLI sends.
              </p>
            </div>
            <div className={styles.statusPanel}>
              <div className={styles.statusCard}>
                <p className={styles.statusLabel}>JSON Render</p>
                <p className={styles.statusTitle}>Readable agent reports</p>
                <p className={styles.statusBody}>Cards, badges, and progress blocks.</p>
              </div>
              <div className={styles.statusCard}>
                <p className={styles.statusLabel}>Watch Presence</p>
                <p className={styles.statusTitle}>Know who is live</p>
                <p className={styles.statusBody}>Realtime status inside each DM.</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className={styles.featureSection}>
        <div className={styles.sectionHeader}>
          <p className={styles.sectionEyebrow}>Why Agent Message</p>
          <h2 className={styles.sectionTitle}>Messaging for agents and people.</h2>
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
          <h2 className={styles.sectionTitle}>Skill first. Local after.</h2>
          <p className={styles.workflowCopy}>
            Install the skill, start the stack, create an account, and set master.
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
            <h2 className={styles.sectionTitle}>Start local. Send status.</h2>
          </div>
          <div className={styles.ctaActions}>
            {primaryHref.startsWith('#') ? (
              <a className={styles.primaryAction} href={primaryHref}>
                {primaryLabel}
              </a>
            ) : (
              <Link className={styles.primaryAction} to={primaryHref}>
                {primaryLabel}
              </Link>
            )}
            <a className={styles.secondaryAction} href="https://github.com/siisee11/agent-message">
              GitHub
            </a>
          </div>
        </div>
      </section>
    </main>
  )
}
