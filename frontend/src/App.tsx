import { Navigate, Route, Routes } from 'react-router-dom'
import Layout from './components/Layout'
import ProtectedRoute from './components/ProtectedRoute'
import { ToastHost } from './components/ui'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Peers from './pages/Peers'
import Servers from './pages/Servers'
import ServerDetail from './pages/ServerDetail'
import Users from './pages/Users'
import Profile from './pages/Profile'
import Configs from './pages/Configs'
import Logs from './pages/Logs'
import Settings from './pages/Settings'

export default function App() {
  return (
    <>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route element={<ProtectedRoute><Layout /></ProtectedRoute>}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/clients" element={<Peers />} />
          <Route path="/peers" element={<Navigate to="/clients" replace />} />
          <Route path="/servers" element={<Servers />} />
          <Route path="/servers/:id" element={<ServerDetail />} />
          <Route path="/configs" element={<Configs />} />
          <Route path="/logs" element={<Logs />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/users" element={<Users />} />
          <Route path="/profile" element={<Profile />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      <ToastHost />
    </>
  )
}
