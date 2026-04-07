import { useLayoutEffect } from 'react'

const SURFACE_BACKGROUND_VARIABLE = '--app-surface-background'

interface UseDocumentSurfaceOptions {
  backgroundColor: string
  themeColor?: string
}

export function useDocumentSurface({
  backgroundColor,
  themeColor = backgroundColor,
}: UseDocumentSurfaceOptions) {
  useLayoutEffect(() => {
    const root = document.documentElement
    const body = document.body
    const themeColorMeta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')

    const previousRootBackground = root.style.getPropertyValue(SURFACE_BACKGROUND_VARIABLE)
    const previousBodyBackground = body.style.backgroundColor
    const previousThemeColor = themeColorMeta?.getAttribute('content')

    root.style.setProperty(SURFACE_BACKGROUND_VARIABLE, backgroundColor)
    body.style.backgroundColor = backgroundColor
    themeColorMeta?.setAttribute('content', themeColor)

    return () => {
      if (previousRootBackground.trim() === '') {
        root.style.removeProperty(SURFACE_BACKGROUND_VARIABLE)
      } else {
        root.style.setProperty(SURFACE_BACKGROUND_VARIABLE, previousRootBackground)
      }

      body.style.backgroundColor = previousBodyBackground

      if (!themeColorMeta) {
        return
      }

      if (previousThemeColor && previousThemeColor.trim() !== '') {
        themeColorMeta.setAttribute('content', previousThemeColor)
      } else {
        themeColorMeta.removeAttribute('content')
      }
    }
  }, [backgroundColor, themeColor])
}
