import { useDeviceStore } from '../../shared/store/deviceStore'
import { Activity, Wifi, AlertTriangle } from 'lucide-react'

export default function MapStatsBar() {
  const devices = useDeviceStore((s) => s.devices)
  const deviceList = Object.values(devices)
  const online = deviceList.filter((d) => d.online).length
  const moving = deviceList.filter((d) => d.position && d.position.speed > 0).length
  const alerts = deviceList.filter((d) => d.position?.sos_event).length

  return (
    <div className="map-stats-bar animate-fade-in">
      <div className="map-stat">
        <Wifi size={13} color="var(--color-success)" />
        <span style={{ color: 'var(--color-text-secondary)' }}>Online</span>
        <span className="map-stat-value">{online}</span>
      </div>
      <div className="map-stat">
        <Activity size={13} color="var(--color-accent)" />
        <span style={{ color: 'var(--color-text-secondary)' }}>Moving</span>
        <span className="map-stat-value">{moving}</span>
      </div>
      {alerts > 0 && (
        <div className="map-stat">
          <AlertTriangle size={13} color="var(--color-danger)" />
          <span style={{ color: 'var(--color-text-secondary)' }}>SOS</span>
          <span className="map-stat-value" style={{ color: 'var(--color-danger)' }}>{alerts}</span>
        </div>
      )}
    </div>
  )
}
