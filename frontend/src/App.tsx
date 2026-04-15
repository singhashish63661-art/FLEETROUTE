import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './shared/store/authStore'
import AppShell from './shared/components/AppShell'
import LoginPage from './features/auth/LoginPage'
import LiveTracking from './features/tracking/LiveTracking'
import FleetPage from './features/fleet/FleetPage'
import AlertsPage from './features/alerts/AlertsPage'
import { ReportsPage } from './features/reports/ReportsPage'
import { SettingsPage } from './features/settings/SettingsPage'

export default function App() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  if (!isAuthenticated) {
    return (
      <Routes>
        <Route path="/login" element={<LoginPage />}/>
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    )
  }

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<Navigate to="/tracking" replace />} />
        <Route path="/tracking" element={<LiveTracking />} />
        <Route path="/fleet" element={<FleetPage />} />
        <Route path="/alerts" element={<AlertsPage />} />
        <Route path="/reports" element={<ReportsPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="*" element={<Navigate to="/tracking" replace />} />
      </Route>
    </Routes>
  )
}
