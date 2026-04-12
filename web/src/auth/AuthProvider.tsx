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
import { disablePushNotifications } from '../notifications/push'

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated'

export interface LoginCredentials {
  accountId: string
  password: string
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

export function AuthProvider({ children }: PropsWithChildren) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [token, setToken] = useState<string | null>(null)
  const [user, setUser] = useState<UserProfile | null>(null)

  const applyAuthenticatedState = useCallback((response: AuthResponse) => {
    apiClient.setAuthToken(null)
    setToken(null)
    setUser(response.user)
    setStatus('authenticated')
  }, [])

  const clearAuthState = useCallback(() => {
    apiClient.setAuthToken(null)
    setToken(null)
    setUser(null)
    setStatus('unauthenticated')
  }, [])

  useEffect(() => {
    let cancelled = false
    apiClient.setAuthToken(null)

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
        account_id: credentials.accountId.trim(),
        password: credentials.password,
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
      await disablePushNotifications()
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
