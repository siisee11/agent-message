import type { BaseComponentProps } from '@json-render/react'
import { useEffect, useId, useState } from 'react'
import styles from './MessageJsonRender.module.css'

interface MessageImageProps {
  alt: string
  height?: number | null
  src?: string | null
  width?: number | null
}

function resolveText(value: unknown): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function resolveDimension(value: unknown): number | undefined {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return undefined
  }
  return value
}

export function MessageImage({ props }: BaseComponentProps<MessageImageProps>) {
  const [zoomed, setZoomed] = useState(false)
  const titleId = useId()
  const src = resolveText(props.src)
  const alt = resolveText(props.alt) ?? 'Image'
  const width = resolveDimension(props.width)
  const height = resolveDimension(props.height)
  const aspectRatio = width && height ? `${width} / ${height}` : undefined

  useEffect(() => {
    if (!zoomed || typeof window === 'undefined') {
      return
    }

    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setZoomed(false)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [zoomed])

  if (!src) {
    return (
      <div className={styles.imagePlaceholder} style={aspectRatio ? { aspectRatio } : undefined}>
        {alt}
      </div>
    )
  }

  return (
    <>
      <button
        aria-label={`Zoom image: ${alt}`}
        className={styles.imageButton}
        onClick={() => {
          setZoomed(true)
        }}
        style={aspectRatio ? { aspectRatio } : undefined}
        type="button"
      >
        <img alt={alt} className={styles.image} height={height} loading="lazy" src={src} width={width} />
      </button>

      {zoomed ? (
        <div
          aria-labelledby={titleId}
          aria-modal="true"
          className={styles.imageZoomOverlay}
          onClick={() => {
            setZoomed(false)
          }}
          role="dialog"
        >
          <div className={styles.imageZoomBackdrop} />
          <figure
            className={styles.imageZoomFrame}
            onClick={(event) => {
              event.stopPropagation()
            }}
          >
            <img alt={alt} className={styles.imageZoomImage} height={height} src={src} width={width} />
            <figcaption className={styles.imageZoomCaption} id={titleId}>
              {alt}
            </figcaption>
            <button
              aria-label="Close image preview"
              className={styles.imageZoomClose}
              onClick={() => {
                setZoomed(false)
              }}
              type="button"
            >
              ×
            </button>
          </figure>
        </div>
      ) : null}
    </>
  )
}
