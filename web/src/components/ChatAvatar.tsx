import type { CSSProperties } from 'react'
import styles from './ChatAvatar.module.css'

interface ChatAvatarProps {
  username: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

function resolveAvatarInitials(username: string): string {
  const normalized = username.trim().replace(/^@+/, '')
  if (normalized === '') {
    return '?'
  }

  const parts = normalized
    .split(/[^A-Za-z0-9]+/)
    .map((part) => part.trim())
    .filter(Boolean)

  if (parts.length >= 2) {
    return `${parts[0][0]}${parts[1][0]}`.toUpperCase()
  }

  return normalized.slice(0, 2).toUpperCase()
}

function resolveAvatarAccent(username: string): string {
  const normalized = username.trim().toLowerCase()
  let hash = 0

  for (const char of normalized) {
    hash = (hash * 31 + char.charCodeAt(0)) % 360
  }

  return `hsl(${hash} 72% 44%)`
}

export function ChatAvatar({ username, size = 'md', className }: ChatAvatarProps) {
  const initials = resolveAvatarInitials(username)
  const style = {
    '--chat-avatar-accent': resolveAvatarAccent(username),
  } as CSSProperties

  return (
    <span
      aria-hidden="true"
      className={`${styles.avatar} ${styles[`avatar${size.toUpperCase() as 'SM' | 'MD' | 'LG'}`]}${
        className ? ` ${className}` : ''
      }`}
      style={style}
      title={`@${username}`}
    >
      <span className={styles.label}>{initials}</span>
    </span>
  )
}
