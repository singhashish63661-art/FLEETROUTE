import { useState } from 'react'
import { useDeviceStore } from '../../shared/store/deviceStore'
import { useQuery } from '@tanstack/react-query'
import api from '../../shared/api/client'
import { Search } from 'lucide-react'

export default function DevicePanel() {
  const [search, setSearch] = useState('')
  const devices = useDeviceStore((s) => s.devices)
  const selectedId = useDeviceStore((s) => s.selectedDeviceId)
  const selectDevice = useDeviceStore((s) => s.selectDevice)
  const setDevices = useDeviceStore((s) => s.setDevices)

  // Fetch device list on mount
  useQuery({
    queryKey: ['devices'],
    queryFn: async () => {
      const res = await api.get('/devices')
      const list = res.data.data ?? []
      setDevices(list)
      return list
    },
    refetchInterval: 60_000,
  })

  const filtered = Object.values(devices).filter(
    (d) =>
      !search ||
      d.name.toLowerCase().includes(search.toLowerCase()) ||
      d.imei.includes(search),
  )

  const onlineCount = filtered.filter((d) => d.online).length

  return (
    <div className="device-panel">
      <div className="device-panel-header">
        <div className="flex items-center justify-between">
          <h2 className="device-panel-title">Vehicles</h2>
          <div className="flex gap-2">
            <span className="badge badge-success">{onlineCount} online</span>
            <span className="badge badge-muted">{filtered.length} total</span>
          </div>
        </div>
        <div className="relative">
          <Search
            size={14}
            style={{
              position: 'absolute',
              left: 10,
              top: '50%',
              transform: 'translateY(-50%)',
              color: 'var(--color-text-muted)',
            }}
          />
          <input
            id="device-search"
            className="input"
            placeholder="Search vehicles or IMEI..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ paddingLeft: 30 }}
          />
        </div>
      </div>

      <div className="device-list">
        {filtered.length === 0 && (
          <div style={{
            padding: 'var(--space-8)',
            textAlign: 'center',
            color: 'var(--color-text-muted)',
            fontSize: 'var(--text-sm)',
          }}>
            No devices found
          </div>
        )}
        {filtered.map((device) => (
          <DeviceRow
            key={device.id}
            device={device}
            selected={device.id === selectedId}
            onClick={() => selectDevice(device.id === selectedId ? null : device.id)}
          />
        ))}
      </div>
    </div>
  )
}

interface DeviceRowProps {
  device: ReturnType<typeof useDeviceStore.getState>['devices'][string]
  selected: boolean
  onClick: () => void
}

function DeviceRow({ device, selected, onClick }: DeviceRowProps) {
  const pos = device.position
  return (
    <div
      className={`device-item${selected ? ' selected' : ''}`}
      onClick={onClick}
      id={`device-row-${device.id}`}
    >
      <div
        className={`dot ${
          pos?.sos_event ? 'dot-alert' : device.online ? 'dot-online' : 'dot-offline'
        }`}
      />
      <div className="device-item-info">
        <div className="device-item-name">{device.name}</div>
        <div className="device-item-meta">
          {pos ? `${pos.lat.toFixed(4)}, ${pos.lng.toFixed(4)}` : device.imei}
        </div>
      </div>
      {pos && device.online && (
        <div className="device-speed">
          {pos.speed} <span style={{ opacity: 0.6, fontWeight: 400 }}>km/h</span>
        </div>
      )}
      {!device.online && (
        <span className="badge badge-muted">offline</span>
      )}
    </div>
  )
}
