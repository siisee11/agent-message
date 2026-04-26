import { useEffect, useState, type CSSProperties, type WheelEvent } from 'react'
import type { BaseComponentProps } from '@json-render/react'
import styles from './MessageJsonRender.module.css'

interface MessageImageProps {
  alt?: string | null
  height?: number | null
  src?: string | null
  width?: number | null
}

const MIN_ZOOM = 1
const MAX_ZOOM = 4
const ZOOM_STEP = 0.5

function normalizeDimension(value: number | null | undefined): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : undefined
}

function clampZoom(value: number): number {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Number(value.toFixed(2))))
}

function formatZoom(value: number): string {
  return `${Math.round(value * 100)}%`
}

export function MessageImage({ props }: BaseComponentProps<MessageImageProps>) {
  const [isOpen, setIsOpen] = useState(false)
  const [zoom, setZoom] = useState(MIN_ZOOM)
  const src = typeof props.src === 'string' && props.src.trim() !== '' ? props.src : null
  const alt = typeof props.alt === 'string' ? props.alt : ''
  const width = normalizeDimension(props.width)
  const height = normalizeDimension(props.height)
  const imageStyle: CSSProperties =
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

  useEffect(() => {
    if (!isOpen) {
      setZoom(MIN_ZOOM)
    }
  }, [isOpen])

  const updateZoom = (delta: number) => {
    setZoom((current) => clampZoom(current + delta))
  }

  const handleWheel = (event: WheelEvent<HTMLDivElement>) => {
    if (!event.ctrlKey && !event.metaKey) {
      return
    }

    event.preventDefault()
    updateZoom(event.deltaY > 0 ? -0.2 : 0.2)
  }

  if (!src) {
    return (
      <div
        className={styles.imagePlaceholder}
        style={{
          aspectRatio: width && height ? `${width} / ${height}` : '4 / 3',
          width,
        }}
      >
        {alt || 'img'}
      </div>
    )
  }

  return (
    <>
      <button
        aria-label={alt ? `Open image: ${alt}` : 'Open image'}
        className={styles.imageButton}
        onClick={() => setIsOpen(true)}
        type="button"
      >
        <img
          alt={alt}
          className={styles.imagePreview}
          height={height}
          loading="lazy"
          src={src}
          style={imageStyle}
          width={width}
        />
      </button>
      {isOpen ? (
        <div
          aria-label={alt || 'Image preview'}
          aria-modal="true"
          className={styles.imageLightbox}
          onClick={(event) => {
            if (event.target === event.currentTarget) {
              setIsOpen(false)
            }
          }}
          role="dialog"
        >
          <div className={styles.imageLightboxToolbar}>
            <button
              aria-label="Zoom out"
              className={styles.imageLightboxControl}
              disabled={zoom <= MIN_ZOOM}
              onClick={() => updateZoom(-ZOOM_STEP)}
              type="button"
            >
              -
            </button>
            <span className={styles.imageLightboxZoom}>{formatZoom(zoom)}</span>
            <button
              aria-label="Zoom in"
              className={styles.imageLightboxControl}
              disabled={zoom >= MAX_ZOOM}
              onClick={() => updateZoom(ZOOM_STEP)}
              type="button"
            >
              +
            </button>
            <button
              aria-label="Reset zoom"
              className={styles.imageLightboxControl}
              disabled={zoom === MIN_ZOOM}
              onClick={() => setZoom(MIN_ZOOM)}
              type="button"
            >
              1:1
            </button>
            <button
              aria-label="Close image preview"
              className={styles.imageLightboxControl}
              onClick={() => setIsOpen(false)}
              type="button"
            >
              Close
            </button>
          </div>
          <div className={styles.imageLightboxScroller} onWheel={handleWheel}>
            <img
              alt={alt}
              className={styles.imageLightboxImage}
              height={height}
              src={src}
              style={{ '--message-image-zoom': zoom } as CSSProperties}
              width={width}
            />
          </div>
        </div>
      ) : null}
    </>
  )
}
