import styles from './BrandLogo.module.css'

interface BrandLogoProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
  withText?: boolean
}

export function BrandLogo({ className, size = 'md', withText = true }: BrandLogoProps) {
  const classes = [styles.logo, styles[`size${size.toUpperCase()}`]]
  if (className) {
    classes.push(className)
  }

  return (
    <span className={classes.join(' ')}>
      <svg aria-hidden="true" className={styles.mark} viewBox="0 0 64 64">
        <rect className={styles.markFrame} x="4" y="4" width="56" height="56" />
        <path className={styles.markBubble} d="M16 18H48V38H30L18 48V38H16V18Z" />
        <rect className={styles.markBar} x="23" y="26" width="18" height="3" />
        <path className={styles.markPrompt} d="M24 34L29 39L24 44V34Z" />
        <rect className={styles.markCursor} x="32" y="40" width="11" height="3" />
      </svg>
      {withText ? <span className={styles.wordmark}>Agent Message</span> : null}
    </span>
  )
}
