import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from 'react'

const THEME_STORAGE_KEY = 'agent_message.theme'
const COLOR_MODE_STORAGE_KEY = 'agent_message.color_mode'

export const AVAILABLE_THEMES = [
  {
    id: 'default',
    label: 'Default',
    themeColorByMode: {
      light: '#f3efe7',
      dark: '#1f2228',
    },
  },
  {
    id: 'ibm',
    label: 'IBM',
    themeColorByMode: {
      light: '#ffffff',
      dark: '#161616',
    },
  },
  {
    id: 'ferrari',
    label: 'Ferrari',
    themeColorByMode: {
      light: '#ffffff',
      dark: '#000000',
    },
  },
  {
    id: 'opencode',
    label: 'OpenCode',
    themeColorByMode: {
      light: '#fdfcfc',
      dark: '#201d1d',
    },
  },
  {
    id: 'neo-brutalism',
    label: 'Neo-Brutalism',
    themeColorByMode: {
      light: '#ffbe0b',
      dark: '#151515',
    },
  },
] as const

export type ThemeName = (typeof AVAILABLE_THEMES)[number]['id']
export type ColorMode = 'light' | 'dark'
type ThemeDefinition = (typeof AVAILABLE_THEMES)[number]

interface ThemeContextValue {
  availableThemes: readonly ThemeDefinition[]
  colorMode: ColorMode
  setTheme: (theme: ThemeName) => void
  theme: ThemeName
  themeColor: string
  toggleColorMode: () => void
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined)
const THEMES_BY_ID = new Map<ThemeName, ThemeDefinition>(
  AVAILABLE_THEMES.map((theme) => [theme.id, theme]),
)

function readStoredTheme(): ThemeName | null {
  try {
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY)
    if (
      stored === 'default' ||
      stored === 'ibm' ||
      stored === 'ferrari' ||
      stored === 'opencode' ||
      stored === 'neo-brutalism'
    ) {
      return stored
    }
  } catch {
    // Ignore storage failures.
  }
  return null
}

function writeStoredTheme(theme: ThemeName): void {
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    // Ignore storage failures.
  }
}

function readStoredColorMode(): ColorMode | null {
  try {
    const storedColorMode = window.localStorage.getItem(COLOR_MODE_STORAGE_KEY)
    if (storedColorMode === 'light' || storedColorMode === 'dark') {
      return storedColorMode
    }

    const legacyThemeValue = window.localStorage.getItem(THEME_STORAGE_KEY)
    if (legacyThemeValue === 'light' || legacyThemeValue === 'dark') {
      return legacyThemeValue
    }
  } catch {
    // Ignore storage failures.
  }

  return null
}

function writeStoredColorMode(colorMode: ColorMode): void {
  try {
    window.localStorage.setItem(COLOR_MODE_STORAGE_KEY, colorMode)
  } catch {
    // Ignore storage failures.
  }
}

export function ThemeProvider({ children }: PropsWithChildren) {
  const [theme, setTheme] = useState<ThemeName>(() => readStoredTheme() ?? 'default')
  const [colorMode, setColorMode] = useState<ColorMode>(() => readStoredColorMode() ?? 'light')
  const themeDefinition = THEMES_BY_ID.get(theme) ?? AVAILABLE_THEMES[0]

  useEffect(() => {
    const root = document.documentElement
    root.dataset.theme = themeDefinition.id
    root.dataset.colorMode = colorMode
    root.style.colorScheme = colorMode
    writeStoredTheme(theme)
    writeStoredColorMode(colorMode)
  }, [colorMode, theme, themeDefinition])

  const value = useMemo<ThemeContextValue>(
    () => ({
      availableThemes: AVAILABLE_THEMES,
      colorMode,
      setTheme,
      theme,
      themeColor: themeDefinition.themeColorByMode[colorMode],
      toggleColorMode: () => {
        setColorMode((current) => (current === 'dark' ? 'light' : 'dark'))
      },
    }),
    [colorMode, theme, themeDefinition],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider')
  }
  return context
}
