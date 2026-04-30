import { Navigate } from 'react-router-dom'
import { PropsWithChildren, useEffect } from 'react'
import { useAuth } from '../store/auth'
import { api } from '../api/client'

export default function ProtectedRoute({ children }: PropsWithChildren) {
  const token = useAuth((s) => s.accessToken)
  const user = useAuth((s) => s.user)
  const setUser = useAuth((s) => s.setUser)

  useEffect(() => {
    if (token && !user) {
      api.me.get().then(setUser).catch(() => undefined)
    }
  }, [token, user, setUser])

  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}
