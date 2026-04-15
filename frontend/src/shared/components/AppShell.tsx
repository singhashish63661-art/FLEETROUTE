import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { useWebSocket } from '../websocket/useWebSocket'
import {
  MapPin, Truck, Bell, BarChart2, Settings, LogOut,
  Activity, Zap
} from 'lucide-react'
import { useDeviceStore } from '../store/deviceStore'
import { useEffect, useState } from 'react'

const navItems = [
  { to: '/tracking', icon: MapPin,  label: 'Live Tracking' },
  { to: '/fleet',    icon: Truck,   label: 'Fleet' },
  { to: '/alerts',   icon: Bell,    label: 'Alerts', badge: true },
  { to: '/reports',  icon: BarChart2, label: 'Reports' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

export default function AppShell() {
  const { subscribe } = useWebSocket() // Connect WebSocket once at shell level

  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()
  const devices = useDeviceStore((s) => s.devices)
  const [alertCount] = useState(3) // TODO: from alert store

  useEffect(() => {
    subscribe(Object.keys(devices))
  }, [devices, subscribe])

  const onlineCount = Object.values(devices).filter((d) => d.online).length

  return (
    <div className="app-shell">
      {/* ── Sidebar ────────────────────────────────────────────────────── */}
      <aside className="sidebar">
        <div className="sidebar-logo">
          <div className="logo-icon">
            <Zap size={18} color="#080D1A" strokeWidth={2.5} />
          </div>
          <div className="logo-text">Fleet<span>OS</span></div>
        </div>

        <nav className="sidebar-nav">
          <div className="nav-section-label">Navigation</div>
          {navItems.map(({ to, icon: Icon, label, badge }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
            >
              <Icon size={18} className="nav-icon" />
              {label}
              {badge && alertCount > 0 && (
                <span className="nav-badge">{alertCount}</span>
              )}
            </NavLink>
          ))}

          <div className="nav-section-label" style={{ marginTop: 16 }}>Status</div>
          <div className="nav-item" style={{ cursor: 'default' }}>
            <Activity size={18} className="nav-icon" style={{ color: 'var(--color-success)' }} />
            <span style={{ fontSize: 'var(--text-sm)' }}>
              <span style={{ color: 'var(--color-success)', fontWeight: 700 }}>{onlineCount}</span>
              <span style={{ color: 'var(--color-text-muted)' }}> / {Object.keys(devices).length} online</span>
            </span>
          </div>
        </nav>

        <div className="sidebar-footer">
          <div className="user-card" onClick={() => navigate('/settings')}>
            <div className="user-avatar">
              {user?.email?.[0]?.toUpperCase() ?? 'U'}
            </div>
            <div className="user-info">
              <div className="user-name">{user?.email ?? 'User'}</div>
              <div className="user-role">{user?.role ?? 'fleet_manager'}</div>
            </div>
            <button
              className="btn btn-ghost btn-sm"
              onClick={(e) => { e.stopPropagation(); logout() }}
              title="Logout"
            >
              <LogOut size={15} />
            </button>
          </div>
        </div>
      </aside>

      {/* ── Main Content ────────────────────────────────────────────────── */}
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  )
}
