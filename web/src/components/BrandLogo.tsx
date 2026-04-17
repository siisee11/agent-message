import styles from './BrandLogo.module.css'
import { useTheme } from '../theme'

interface BrandLogoProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
  withText?: boolean
}

export function BrandLogo({ className, size = 'md', withText = true }: BrandLogoProps) {
  const { colorMode } = useTheme()
  const classes = [styles.logo, styles[`size${size.toUpperCase()}`]]
  if (className) {
    classes.push(className)
  }

  const markSrc = colorMode === 'light' ? '/agent-message-logo-light.svg' : '/agent-message-logo.svg'

  return (
    <span className={classes.join(' ')}>
      <img aria-hidden="true" alt="" className={styles.mark} src={markSrc} />
      {withText ? <span className={styles.wordmark}>Agent Message</span> : null}
    </span>
  )
}
