import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth'
import { useDocumentSurface } from '../hooks'
import styles from './LandingPage.module.css'

const FEATURE_ITEMS = [
  {
    title: 'CLI native',
    description: 'Agents can send, read, and watch messages directly from the terminal.',
  },
  {
    title: 'JSON render',
    description: 'Receive reports as readable cards, tables, badges, and progress blocks.',
  },
  {
    title: 'Phone ready',
    description: 'Open the web app on your phone and keep working with agents anywhere.',
  },
]

const WRAPPER_ITEMS = [
  {
    title: 'codex-message',
    installScript: 'npm install -g agent-message codex-message',
    icon: '/codex-color.svg',
  },
  {
    title: 'claude-message',
    installScript: 'npm install -g agent-message claude-message',
    icon: '/claude-color.svg',
  },
]

const PHONE_SCREENSHOTS = {
  light: [
    '/phone-screens/IMG_light1.PNG',
    '/phone-screens/IMG_light2.PNG',
    '/phone-screens/IMG_light3.PNG',
    '/phone-screens/IMG_light4.PNG',
  ],
  dark: [
    '/phone-screens/IMG_dark1.PNG',
    '/phone-screens/IMG_dark2.PNG',
    '/phone-screens/IMG_dark3.PNG',
    '/phone-screens/IMG_dark4.PNG',
  ],
}

const DESKTOP_SCREENSHOTS: Record<LandingMode, string> = {
  dark: '/desktop-screens/screenshot_dark.png',
  light: '/desktop-screens/screenshot_light.png',
}

type LandingMode = keyof typeof PHONE_SCREENSHOTS

const LANDING_SURFACES: Record<LandingMode, string> = {
  dark: '#1f2228',
  light: '#f5f6f8',
}

const SETUP_PROMPT = `Set up https://github.com/siisee11/agent-message for me.

Install the agent-message skill first:

npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -g -y

Then read \`install.md\` and follow the self-host setup flow. Ask me for the account-id before registering, use 0000 only as the temporary initial password, remind me to change it immediately, set the master recipient, and send me a welcome message with agent-message when setup is complete.`

export function LandingPage() {
  const { isAuthenticated } = useAuth()
  const [hasCopiedSetupPrompt, setHasCopiedSetupPrompt] = useState(false)
  const [copiedWrapperTitle, setCopiedWrapperTitle] = useState<string | null>(null)
  const [landingMode, setLandingMode] = useState<LandingMode>('dark')
  const [activePhoneFrame, setActivePhoneFrame] = useState(0)

  useDocumentSurface({
    backgroundColor: LANDING_SURFACES[landingMode],
    themeColor: LANDING_SURFACES[landingMode],
  })

  const primaryHref = isAuthenticated ? '/app' : '#setup'
  const primaryLabel = isAuthenticated ? 'Open App' : 'Local Setup'
  const activePhoneScreenshots = PHONE_SCREENSHOTS[landingMode]
  const activePhoneScreenshot = activePhoneScreenshots[activePhoneFrame % activePhoneScreenshots.length]
  const activeDesktopScreenshot = DESKTOP_SCREENSHOTS[landingMode]
  const logoSrc = landingMode === 'dark' ? '/agent-message-logo.svg' : '/agent-message-logo-light.svg'
  const nextLandingMode: LandingMode = landingMode === 'dark' ? 'light' : 'dark'

  useEffect(() => {
    const timer = window.setInterval(() => {
      setActivePhoneFrame((frame) => (frame + 1) % activePhoneScreenshots.length)
    }, 2200)

    return () => window.clearInterval(timer)
  }, [activePhoneScreenshots.length])

  const copyText = async (text: string) => {
    const copyWithTextArea = () => {
      const textArea = document.createElement('textarea')
      textArea.value = text
      textArea.setAttribute('readonly', 'true')
      textArea.style.position = 'fixed'
      textArea.style.left = '-9999px'
      document.body.appendChild(textArea)
      textArea.select()
      document.execCommand('copy')
      document.body.removeChild(textArea)
    }

    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
      } else {
        copyWithTextArea()
      }
    } catch {
      copyWithTextArea()
    }
  }

  const copySetupPrompt = async () => {
    await copyText(SETUP_PROMPT)
    setHasCopiedSetupPrompt(true)
    window.setTimeout(() => setHasCopiedSetupPrompt(false), 1800)
  }

  const copyWrapperInstallScript = async (title: string, installScript: string) => {
    await copyText(installScript)
    setCopiedWrapperTitle(title)
    window.setTimeout(() => {
      setCopiedWrapperTitle((currentTitle) => (currentTitle === title ? null : currentTitle))
    }, 1800)
  }

  return (
    <main className={`${styles.page} ${landingMode === 'light' ? styles.lightPage : ''}`}>
      <div aria-hidden="true" className={styles.aurora} />
      <div aria-hidden="true" className={styles.gridGlow} />

      <header className={styles.nav}>
        <Link className={styles.brand} to="/">
          <img aria-hidden="true" alt="" className={styles.brandMark} src={logoSrc} />
          <span className={styles.brandText}>Agent Message</span>
        </Link>

        <div className={styles.navActions}>
          <button
            aria-label={`Switch to ${nextLandingMode} mode`}
            className={styles.themeToggleButton}
            onClick={() => {
              setLandingMode(nextLandingMode)
              setActivePhoneFrame(0)
            }}
            title={`Switch to ${nextLandingMode} mode`}
            type="button"
          >
            {landingMode === 'dark' ? (
              <svg aria-hidden="true" className={styles.themeToggleIcon} viewBox="0 0 24 24">
                <circle cx="12" cy="12" r="4" />
                <path d="M12 2v3" />
                <path d="M12 19v3" />
                <path d="M2 12h3" />
                <path d="M19 12h3" />
                <path d="m4.93 4.93 2.12 2.12" />
                <path d="m16.95 16.95 2.12 2.12" />
                <path d="m19.07 4.93-2.12 2.12" />
                <path d="m7.05 16.95-2.12 2.12" />
              </svg>
            ) : (
              <svg aria-hidden="true" className={styles.themeToggleIcon} viewBox="0 0 24 24">
                <path d="M20 14.5A8 8 0 0 1 9.5 4 8 8 0 1 0 20 14.5Z" />
              </svg>
            )}
          </button>
          <a className={styles.navLink} href="https://github.com/siisee11/agent-message">
            GitHub
          </a>
        </div>
      </header>

      <section className={styles.hero}>
        <div className={styles.heroCopy}>
          <p className={styles.eyebrow}>CLI / JSON render / mobile web</p>
          <h1 className={styles.title}>The messenger agents use.</h1>
          <p className={styles.description}>
            Agents use the CLI. Humans read structured json_render reports. The web app keeps the thread
            moving from your phone.
          </p>

        </div>

        <div className={styles.phoneShowcase}>
          <div aria-label={`${landingMode} mode mobile screenshots`} className={styles.phonePreview}>
            <div className={styles.phoneScreen}>
              <img
                alt={`Agent Message ${landingMode} mode mobile screenshot ${activePhoneFrame + 1}`}
                className={styles.phoneScreenshot}
                src={activePhoneScreenshot}
              />
            </div>
          </div>
        </div>

        <div className={styles.setupBlock} id="setup">
          <h2 className={styles.setupTitle}>Setup Prompt</h2>
          <div className={styles.installCard}>
            <div className={styles.installHeader}>
              <button className={styles.copyPromptButton} onClick={copySetupPrompt} type="button">
                {hasCopiedSetupPrompt ? 'Copied' : 'Copy & Paste'}
              </button>
            </div>
            <code className={styles.installCommand}>{SETUP_PROMPT}</code>
          </div>

          <p className={styles.installHint}>
            Or install manually with npm. Cloud service is coming soon.
          </p>
        </div>

      </section>

      <section className={styles.desktopPreviewSection} aria-label="Agent Message desktop preview">
        <img
          alt={`Agent Message ${landingMode} mode desktop screenshot`}
          className={styles.desktopPreviewImage}
          src={activeDesktopScreenshot}
        />
      </section>

      <section className={styles.featureSection}>
        <div className={styles.sectionHeader}>
          <p className={styles.sectionEyebrow}>Why Agent Message</p>
          <h2 className={styles.sectionTitle}>Built for how agents actually work.</h2>
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

      <section className={styles.workflowSection}>
        <div className={styles.workflowIntro}>
          <p className={styles.sectionEyebrow}>Wrappers</p>
          <h2 className={styles.sectionTitle}>Natively support Codex and Claude Code.</h2>
          <p className={styles.workflowCopy}>
            `codex-message` and `claude-message` connect agent runtimes to Agent Message so people can
            steer work from the web app or phone.
          </p>
        </div>

        <div className={styles.guideGrid}>
          {WRAPPER_ITEMS.map((item) => (
            <article className={styles.guideCard} key={item.title}>
              <div className={styles.guideHeader}>
                <div className={styles.guideTitleGroup}>
                  <img alt="" className={styles.guideIcon} src={item.icon} />
                  <p className={styles.guideTitle}>{item.title}</p>
                </div>
                <button
                  aria-label={`Copy ${item.title} install script`}
                  className={styles.guideCopyButton}
                  onClick={() => copyWrapperInstallScript(item.title, item.installScript)}
                  title={`Copy ${item.title} install script`}
                  type="button"
                >
                  {copiedWrapperTitle === item.title ? (
                    <svg aria-hidden="true" className={styles.guideCopyIcon} viewBox="0 0 24 24">
                      <path d="m5 12 4 4 10-10" />
                    </svg>
                  ) : (
                    <svg aria-hidden="true" className={styles.guideCopyIcon} viewBox="0 0 24 24">
                      <rect height="11" width="11" x="8" y="7" />
                      <path d="M5 17V4h13" />
                    </svg>
                  )}
                </button>
              </div>
              <code className={styles.guideCommand}>{item.installScript}</code>
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
