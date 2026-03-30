import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../auth'

export function ProtectedRoute(): JSX.Element {
  const location = useLocation()
  const { status, isAuthenticated } = useAuth()

  if (status === 'loading') {
    return <p>Checking session...</p>
  }

  if (!isAuthenticated) {
    return <Navigate replace state={{ from: location }} to="/login" />
  }

  return <Outlet />
}

