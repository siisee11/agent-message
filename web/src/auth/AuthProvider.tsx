import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from 'react'
import { ApiError, type AuthResponse, type UserProfile } from '../api'
import { apiClient } from '../api/runtime'

const AUTH_TOKEN_STORAGE_KEY = 'agent_message.auth_token'

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated'

export interface LoginCredentials {
  username: string
  pin: string
}

export interface LoginResult {
  mode: 'login' | 'register'
}

interface AuthContextValue {
  status: AuthStatus
  isAuthenticated: boolean
  token: string | null
  user: UserProfile | null
  loginWithAutoRegister: (credentials: LoginCredentials) => Promise<LoginResult>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

function readStoredToken(): string | null {
  try {
    return window.localStorage.getItem(AUTH_TOKEN_STORAGE_KEY)
  } catch {
    return null
  }
}

function writeStoredToken(token: string | null): void {
  try {
    if (token === null) {
      window.localStorage.removeItem(AUTH_TOKEN_STORAGE_KEY)
      return
    }
    window.localStorage.setItem(AUTH_TOKEN_STORAGE_KEY, token)
  } catch {
    // Local storage may be unavailable in private mode or restricted contexts.
  }
}

export function AuthProvider({ children }: PropsWithChildren) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [token, setToken] = useState<string | null>(null)
  const [user, setUser] = useState<UserProfile | null>(null)

  const applyAuthenticatedState = useCallback((response: AuthResponse) => {
    apiClient.setAuthToken(response.token)
    writeStoredToken(response.token)
    setToken(response.token)
    setUser(response.user)
    setStatus('authenticated')
  }, [])

  const clearAuthState = useCallback(() => {
    apiClient.setAuthToken(null)
    writeStoredToken(null)
    setToken(null)
    setUser(null)
    setStatus('unauthenticated')
  }, [])

  useEffect(() => {
    const storedToken = readStoredToken()
    if (!storedToken) {
      setStatus('unauthenticated')
      return
    }

    let cancelled = false
    apiClient.setAuthToken(storedToken)
    setToken(storedToken)

    void apiClient
      .getMe()
      .then((profile) => {
        if (cancelled) {
          return
        }
        setUser(profile)
        setStatus('authenticated')
      })
      .catch(() => {
        if (cancelled) {
          return
        }
        clearAuthState()
      })

    return () => {
      cancelled = true
    }
  }, [clearAuthState])

  const loginWithAutoRegister = useCallback(
    async (credentials: LoginCredentials): Promise<LoginResult> => {
      const payload = {
        username: credentials.username.trim(),
        pin: credentials.pin.trim(),
      }

      try {
        const loginResponse = await apiClient.login(payload)
        applyAuthenticatedState(loginResponse)
        return { mode: 'login' }
      } catch (error: unknown) {
        if (!(error instanceof ApiError) || error.status !== 401) {
          throw error
        }

        try {
          const registerResponse = await apiClient.register(payload)
          applyAuthenticatedState(registerResponse)
          return { mode: 'register' }
        } catch (registerError: unknown) {
          if (registerError instanceof ApiError && registerError.status === 409) {
            throw new ApiError('invalid credentials', 401, '/api/auth/login')
          }
          throw registerError
        }
      }
    },
    [applyAuthenticatedState],
  )

  const logout = useCallback(async (): Promise<void> => {
    try {
      await apiClient.logout()
    } catch (error: unknown) {
      if (!(error instanceof ApiError) || error.status !== 401) {
        throw error
      }
    } finally {
      clearAuthState()
    }
  }, [clearAuthState])

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      isAuthenticated: status === 'authenticated',
      token,
      user,
      loginWithAutoRegister,
      logout,
    }),
    [loginWithAutoRegister, logout, status, token, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
