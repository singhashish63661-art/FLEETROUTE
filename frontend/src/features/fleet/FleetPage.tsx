import { useQuery } from '@tanstack/react-query'
import api from '../../shared/api/client'
import { Truck, Plus, Signal, Battery } from 'lucide-react'

interface Device {
  id: string
  imei: string
  name: string
  protocol: string
  online: boolean
  last_seen_at?: string
}

export default function FleetPage() {
  const { data: devices = [] } = useQuery<Device[]>({
    queryKey: ['devices-fleet'],
    queryFn: async () => {
      const res = await api.get('/devices')
      return res.data.data ?? []
    },
  })

  const online = devices.filter((d) => d.online).length

  return (
    <div className="page" style={{ padding: 'var(--space-6)' }}>
      {/* Header */}
      <div className="flex items-center justify-between" style={{ marginBottom: 'var(--space-6)' }}>
        <div>
          <h1 style={{ fontSize: 'var(--text-2xl)', fontWeight: 700 }}>Fleet</h1>
          <p className="text-muted text-sm" style={{ marginTop: 4 }}>
            {devices.length} devices · {online} online
          </p>
        </div>
        <button id="add-device" className="btn btn-primary">
          <Plus size={16} />
          Add Device
        </button>
      </div>

      {/* Stats */}
      <div className="stat-grid" style={{ marginBottom: 'var(--space-6)' }}>
        <div className="stat-card success">
          <div className="stat-label">Online</div>
          <div className="stat-value" style={{ color: 'var(--color-success)' }}>{online}</div>
          <div className="stat-change up">↑ 2 from yesterday</div>
        </div>
        <div className="stat-card accent">
          <div className="stat-label">Total Devices</div>
          <div className="stat-value">{devices.length}</div>
          <div className="stat-change">registered</div>
        </div>
        <div className="stat-card warning">
          <div className="stat-label">Offline</div>
          <div className="stat-value" style={{ color: 'var(--color-warning)' }}>
            {devices.length - online}
          </div>
          <div className="stat-change down">needs attention</div>
        </div>
      </div>

      {/* Device Table */}
      <div className="card animate-fade-in">
        <div className="card-header">
          <h2 className="card-title">Devices</h2>
        </div>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Status</th>
                <th>Name</th>
                <th>IMEI</th>
                <th>Protocol</th>
                <th>Last Seen</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {devices.length === 0 && (
                <tr>
                  <td colSpan={6} style={{ textAlign: 'center', color: 'var(--color-text-muted)', padding: 32 }}>
                    No devices registered
                  </td>
                </tr>
              )}
              {devices.map((device) => (
                <tr key={device.id} id={`device-row-fleet-${device.id}`}>
                  <td>
                    <div className="flex items-center gap-2">
                      <div className={`dot ${device.online ? 'dot-online' : 'dot-offline'}`} />
                      <span className={`badge ${device.online ? 'badge-success' : 'badge-muted'}`}>
                        {device.online ? 'online' : 'offline'}
                      </span>
                    </div>
                  </td>
                  <td>
                    <div className="flex items-center gap-2">
                      <Truck size={15} color="var(--color-text-muted)" />
                      <span style={{ fontWeight: 500 }}>{device.name}</span>
                    </div>
                  </td>
                  <td>
                    <span className="font-mono text-sm text-muted">{device.imei}</span>
                  </td>
                  <td>
                    <span className="badge badge-info">{device.protocol}</span>
                  </td>
                  <td>
                    <span className="text-xs text-muted">
                      {device.last_seen_at
                        ? new Date(device.last_seen_at).toLocaleString()
                        : 'Never'}
                    </span>
                  </td>
                  <td>
                    <div className="flex gap-2">
                      <button className="btn btn-ghost btn-sm" title="Signal">
                        <Signal size={13} />
                      </button>
                      <button className="btn btn-ghost btn-sm" title="Battery">
                        <Battery size={13} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
