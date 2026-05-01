import { useEffect, useState, type CSSProperties } from 'react'
import type { BaseComponentProps } from '@json-render/react'
import styles from './MessageJsonRender.module.css'

interface MessageGifProps {
  alt?: string | null
  height?: number | null
  src?: string | null
  width?: number | null
}

function normalizeDimension(value: number | null | undefined): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : undefined
}

export function MessageGif({ props }: BaseComponentProps<MessageGifProps>) {
  const [isOpen, setIsOpen] = useState(false)
  const src = typeof props.src === 'string' && props.src.trim() !== '' ? props.src : null
  const alt = typeof props.alt === 'string' ? props.alt : ''
  const width = normalizeDimension(props.width)
  const height = normalizeDimension(props.height)
  const mediaStyle: CSSProperties =
    width || height ? { aspectRatio: width && height ? `${width} / ${height}` : undefined } : {}

  useEffect(() => {
    if (!isOpen) {
      return
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false)
      }
    }

    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    window.addEventListener('keydown', handleKeyDown)

    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [isOpen])

  if (!src) {
    return (
      <div
        className={`${styles.imagePlaceholder} ${styles.gifPlaceholder}`}
        style={{
          aspectRatio: width && height ? `${width} / ${height}` : '4 / 3',
          width,
        }}
      >
        {alt || 'gif'}
      </div>
    )
  }

  return (
    <>
      <button
        aria-label={alt ? `Open GIF: ${alt}` : 'Open GIF'}
        className={`${styles.imageButton} ${styles.gifButton}`}
        onClick={() => setIsOpen(true)}
        type="button"
      >
        <img
          alt={alt}
          className={`${styles.imagePreview} ${styles.gifPreview}`}
          height={height}
          loading="lazy"
          src={src}
          style={mediaStyle}
          width={width}
        />
      </button>
      {isOpen ? (
        <div
          aria-label={alt || 'GIF preview'}
          aria-modal="true"
          className={`${styles.imageLightbox} ${styles.gifLightbox}`}
          onClick={(event) => {
            if (event.target === event.currentTarget) {
              setIsOpen(false)
            }
          }}
          role="dialog"
        >
          <div className={styles.imageLightboxToolbar}>
            <button
              aria-label="Close GIF preview"
              className={styles.imageLightboxControl}
              onClick={() => setIsOpen(false)}
              type="button"
            >
              Close
            </button>
          </div>
          <div className={styles.imageLightboxScroller}>
            <img
              alt={alt}
              className={`${styles.imageLightboxImage} ${styles.gifLightboxImage}`}
              height={height}
              src={src}
              width={width}
            />
          </div>
        </div>
      ) : null}
    </>
  )
}
