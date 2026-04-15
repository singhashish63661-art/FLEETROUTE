import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import api from '../../shared/api/client'
import { Bell, CheckCheck, AlertTriangle, Zap, MapPin } from 'lucide-react'

interface Alert {
  id: string
  device_id: string
  alert_type: string
  severity: 'info' | 'warning' | 'critical'
  message: string
  triggered_at: string
  acknowledged_at?: string
}

const severityConfig = {
  info:     { color: 'var(--color-info)',    icon: Bell,          badge: 'badge-info' },
  warning:  { color: 'var(--color-warning)', icon: AlertTriangle, badge: 'badge-warning' },
  critical: { color: 'var(--color-danger)',  icon: Zap,           badge: 'badge-danger' },
}

export default function AlertsPage() {
  const [filter, setFilter] = useState<'all' | 'unacknowledged'>('unacknowledged')

  const { data: alerts = [], refetch } = useQuery<Alert[]>({
    queryKey: ['alerts'],
    queryFn: async () => {
      const res = await api.get('/alerts')
      return res.data.data ?? []
    },
    refetchInterval: 10_000,
  })

  const filtered = filter === 'unacknowledged'
    ? alerts.filter((a) => !a.acknowledged_at)
    : alerts

  const acknowledge = async (id: string) => {
    await api.post(`/alerts/${id}/acknowledge`)
    refetch()
  }

  return (
    <div className="page" style={{ padding: 'var(--space-6)' }}>
      <div className="flex items-center justify-between" style={{ marginBottom: 'var(--space-6)' }}>
        <div>
          <h1 style={{ fontSize: 'var(--text-2xl)', fontWeight: 700 }}>Alerts</h1>
          <p className="text-muted text-sm" style={{ marginTop: 4 }}>
            {filtered.length} {filter === 'unacknowledged' ? 'unacknowledged' : 'total'} alerts
          </p>
        </div>
        <div className="flex gap-2">
          <button
            id="filter-all"
            className={`btn ${filter === 'all' ? 'btn-primary' : 'btn-secondary'}`}
            onClick={() => setFilter('all')}
          >
            All
          </button>
          <button
            id="filter-unack"
            className={`btn ${filter === 'unacknowledged' ? 'btn-primary' : 'btn-secondary'}`}
            onClick={() => setFilter('unacknowledged')}
          >
            Unacknowledged
          </button>
        </div>
      </div>

      <div className="card animate-fade-in">
        {filtered.length === 0 ? (
          <div style={{ padding: 'var(--space-12)', textAlign: 'center' }}>
            <CheckCheck size={40} color="var(--color-success)" style={{ margin: '0 auto 16px' }} />
            <p style={{ color: 'var(--color-text-muted)' }}>All clear — no active alerts</p>
          </div>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Severity</th>
                  <th>Device</th>
                  <th>Type</th>
                  <th>Message</th>
                  <th>Triggered</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((alert) => {
                  const cfg = severityConfig[alert.severity] ?? severityConfig.info
                  const Icon = cfg.icon
                  return (
                    <tr key={alert.id}>
                      <td>
                        <span className={`badge ${cfg.badge}`}>
                          <Icon size={11} />
                          {alert.severity}
                        </span>
                      </td>
                      <td>
                        <div className="flex items-center gap-2">
                          <MapPin size={13} color="var(--color-text-muted)" />
                          <span className="font-mono text-sm">{alert.device_id.slice(0, 8)}…</span>
                        </div>
                      </td>
                      <td>
                        <span className="badge badge-muted">{alert.alert_type}</span>
                      </td>
                      <td style={{ maxWidth: 280 }}>
                        <span className="truncate" style={{ display: 'block' }}>{alert.message}</span>
                      </td>
                      <td>
                        <span className="text-xs text-muted font-mono">
                          {new Date(alert.triggered_at).toLocaleString()}
                        </span>
                      </td>
                      <td>
                        {alert.acknowledged_at ? (
                          <span className="badge badge-success">acknowledged</span>
                        ) : (
                          <span className="badge badge-warning">pending</span>
                        )}
                      </td>
                      <td>
                        {!alert.acknowledged_at && (
                          <button
                            id={`ack-${alert.id}`}
                            className="btn btn-secondary btn-sm"
                            onClick={() => acknowledge(alert.id)}
                          >
                            <CheckCheck size={13} />
                            Ack
                          </button>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
