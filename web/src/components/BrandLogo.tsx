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
      <img aria-hidden="true" alt="" className={styles.mark} src="/agent-message-logo.svg" />
      {withText ? <span className={styles.wordmark}>Agent Message</span> : null}
    </span>
  )
}
