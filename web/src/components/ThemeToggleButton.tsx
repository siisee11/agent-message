import { useEffect, useRef, useState } from 'react'
import { useTheme } from '../theme'
import styles from './ThemeToggleButton.module.css'

function SunIcon() {
  return (
    <svg aria-hidden="true" className={styles.icon} viewBox="0 0 24 24">
      <circle cx="12" cy="12" r="4.5" fill="none" stroke="currentColor" strokeWidth="1.75" />
      <path
        d="M12 2.75v2.5M12 18.75v2.5M21.25 12h-2.5M5.25 12H2.75M18.54 5.46l-1.77 1.77M7.23 16.77l-1.77 1.77M18.54 18.54l-1.77-1.77M7.23 7.23 5.46 5.46"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="1.75"
      />
    </svg>
  )
}

function MoonIcon() {
  return (
    <svg aria-hidden="true" className={styles.icon} viewBox="0 0 24 24">
      <path
        d="M15.25 3.4a8.95 8.95 0 1 0 5.35 15.95 8.3 8.3 0 0 1-2.5.38c-4.82 0-8.73-3.96-8.73-8.86 0-3.02 1.47-5.69 3.73-7.2a8.8 8.8 0 0 1 2.15-.27Z"
        fill="none"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.75"
      />
    </svg>
  )
}

interface ThemeToggleButtonProps {
  className?: string
}

export function ThemeToggleButton({ className }: ThemeToggleButtonProps) {
  const { availableThemes, colorMode, setTheme, theme, toggleColorMode } = useTheme()
  const classes = className ? `${styles.root} ${className}` : styles.root
  const [isMenuOpen, setIsMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!isMenuOpen) {
      return
    }

    function handlePointerDown(event: MouseEvent) {
      if (!menuRef.current?.contains(event.target as Node)) {
        setIsMenuOpen(false)
      }
    }

    function handleEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setIsMenuOpen(false)
      }
    }

    window.addEventListener('mousedown', handlePointerDown)
    window.addEventListener('keydown', handleEscape)
    return () => {
      window.removeEventListener('mousedown', handlePointerDown)
      window.removeEventListener('keydown', handleEscape)
    }
  }, [isMenuOpen])

  return (
    <div className={classes}>
      <div className={styles.menuWrap} ref={menuRef}>
        <button
          aria-expanded={isMenuOpen}
          aria-haspopup="menu"
          className={styles.trigger}
          onClick={() => setIsMenuOpen((current) => !current)}
          type="button"
        >
          <span>Theme</span>
          <span aria-hidden="true" className={styles.chevron}>
            ▾
          </span>
        </button>
        {isMenuOpen ? (
          <div className={styles.menu} role="menu">
            {availableThemes.map((themeOption) => {
              const isActive = themeOption.id === theme
              const optionClassName = isActive ? `${styles.option} ${styles.optionActive}` : styles.option

              return (
                <button
                  aria-pressed={isActive}
                  className={optionClassName}
                  key={themeOption.id}
                  onClick={() => {
                    setTheme(themeOption.id)
                    setIsMenuOpen(false)
                  }}
                  role="menuitemradio"
                  type="button"
                >
                  <span>{themeOption.label}</span>
                  {isActive ? <span className={styles.checkmark}>Selected</span> : null}
                </button>
              )
            })}
          </div>
        ) : null}
      </div>
      <button
        aria-label={colorMode === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        className={styles.iconButton}
        onClick={toggleColorMode}
        type="button"
      >
        {colorMode === 'dark' ? <SunIcon /> : <MoonIcon />}
        <span className={styles.srOnly}>
          {colorMode === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        </span>
      </button>
    </div>
  )
}
