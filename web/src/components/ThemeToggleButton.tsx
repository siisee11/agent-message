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
  const { resolvedTheme, toggleTheme } = useTheme()
  const classes = className ? `${styles.button} ${className}` : styles.button

  return (
    <button
      aria-label={resolvedTheme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
      className={classes}
      onClick={toggleTheme}
      type="button"
    >
      {resolvedTheme === 'dark' ? <SunIcon /> : <MoonIcon />}
      <span className={styles.srOnly}>
        {resolvedTheme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
      </span>
    </button>
  )
}
