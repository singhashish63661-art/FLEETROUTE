import { useEffect, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import api from '../../shared/api/client'

interface SettingsUser {
  id: string
  email: string
  name: string
  role: string
  phone?: string
  is_active: boolean
}

interface SettingsOverview {
  tenant: {
    id: string
    name: string
    slug: string
    plan: string
    settings?: {
      speed_limit_kmh?: number
      idle_timeout_mins?: number
    }
    max_devices: number
    is_active: boolean
  }
  user: {
    id: string
    email: string
    name: string
    role: string
  }
}

export function SettingsPage() {
  const [speedLimit, setSpeedLimit] = useState(100)
  const [idleTimeout, setIdleTimeout] = useState(5)
  const [saveMsg, setSaveMsg] = useState('')

  const { data: overview } = useQuery<SettingsOverview>({
    queryKey: ['settings-overview'],
    queryFn: async () => {
      const res = await api.get('/settings/overview')
      return res.data.data
    },
  })

  const { data: users = [] } = useQuery<SettingsUser[]>({
    queryKey: ['settings-users'],
    queryFn: async () => {
      const res = await api.get('/settings/users')
      return res.data.data ?? []
    },
  })

  useEffect(() => {
    const settings = overview?.tenant?.settings
    if (typeof settings?.speed_limit_kmh === 'number') {
      setSpeedLimit(settings.speed_limit_kmh)
    }
    if (typeof settings?.idle_timeout_mins === 'number') {
      setIdleTimeout(settings.idle_timeout_mins)
    }
  }, [overview])

  const savePrefs = useMutation({
    mutationFn: async () =>
      api.put('/settings/preferences', {
        speed_limit_kmh: speedLimit,
        idle_timeout_mins: idleTimeout,
      }),
    onSuccess: () => setSaveMsg('Preferences saved.'),
    onError: () => setSaveMsg('Failed to save preferences.'),
  })

  return (
    <div style={{ padding: '24px', color: '#fff' }}>
      <h1>Settings & Organization</h1>

      <div style={{ marginTop: 8, color: '#aaa' }}>
        Tenant: <strong>{overview?.tenant?.name ?? 'Loading...'}</strong>
        {' · '}
        Plan: <strong>{overview?.tenant?.plan ?? '-'}</strong>
        {' · '}
        Current user: <strong>{overview?.user?.email ?? '-'}</strong>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: '24px', marginTop: '24px' }}>
        <div style={{ backgroundColor: '#1e1e1e', padding: '16px', borderRadius: '8px' }}>
          <h3>Role Based Access Control</h3>
          <p style={{ color: '#aaa', fontSize: '0.9em' }}>Users from backend tenant scope.</p>

          <ul style={{ listStyle: 'none', padding: 0, marginTop: '16px' }}>
            {users.map((u) => (
              <li key={u.id} style={{ padding: '12px 0', borderBottom: '1px solid #333' }}>
                <strong>{u.name || u.email}</strong> ({u.email}) -{' '}
                <span style={{ color: '#10b981' }}>{u.role}</span>
              </li>
            ))}
            {users.length === 0 && (
              <li style={{ padding: '12px 0', color: '#aaa' }}>No users found.</li>
            )}
          </ul>
        </div>

        <div style={{ backgroundColor: '#1e1e1e', padding: '16px', borderRadius: '8px' }}>
          <h3>Device Configuration</h3>
          <p style={{ color: '#aaa', fontSize: '0.9em' }}>Persisted to backend tenant settings.</p>

          <div style={{ marginTop: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px' }}>Default Speed Limit (km/h)</label>
            <input
              type="number"
              value={speedLimit}
              onChange={(e) => setSpeedLimit(Number(e.target.value))}
              style={{ padding: '8px', width: '120px', backgroundColor: '#111', color: '#fff', border: '1px solid #333', borderRadius: '4px' }}
            />
          </div>

          <div style={{ marginTop: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px' }}>Idle Timeout Threshold (Mins)</label>
            <input
              type="number"
              value={idleTimeout}
              onChange={(e) => setIdleTimeout(Number(e.target.value))}
              style={{ padding: '8px', width: '120px', backgroundColor: '#111', color: '#fff', border: '1px solid #333', borderRadius: '4px' }}
            />
          </div>

          <button
            onClick={() => savePrefs.mutate()}
            disabled={savePrefs.isPending}
            style={{ marginTop: '24px', backgroundColor: '#10b981', color: '#fff', border: 'none', padding: '8px 16px', borderRadius: '4px', cursor: 'pointer' }}
          >
            {savePrefs.isPending ? 'Saving...' : 'Save Global Settings'}
          </button>
          {saveMsg && <div style={{ marginTop: 12, color: '#aaa', fontSize: '0.9em' }}>{saveMsg}</div>}
        </div>
      </div>
    </div>
  )
}
