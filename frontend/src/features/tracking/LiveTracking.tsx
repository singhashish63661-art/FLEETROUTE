import { useEffect, useRef } from 'react'
import { useDeviceStore } from '../../shared/store/deviceStore'
import { OpenLayersAdapter } from '../../shared/map/OpenLayersAdapter'
import DevicePanel from './DevicePanel'
import MapStatsBar from './MapStatsBar'

export default function LiveTracking() {
  const mapContainerRef = useRef<HTMLDivElement>(null)
  const adapterRef = useRef<OpenLayersAdapter | null>(null)
  const devices = useDeviceStore((s) => s.devices)
  const selectedId = useDeviceStore((s) => s.selectedDeviceId)
  const selectDevice = useDeviceStore((s) => s.selectDevice)

  // Initialize map
  useEffect(() => {
    if (!mapContainerRef.current) return
    adapterRef.current = new OpenLayersAdapter()
    adapterRef.current.init(mapContainerRef.current)
    return () => {
      adapterRef.current?.destroy()
      adapterRef.current = null
    }
  }, [])

  // Update markers whenever device positions change
  useEffect(() => {
    const adapter = adapterRef.current
    if (!adapter) return
    Object.values(devices).forEach((device) => {
      if (!device.position) return
      const pos = device.position
      adapter.updateMarker(device.id, {
        lat: pos.lat,
        lng: pos.lng,
        heading: pos.heading,
        speed: pos.speed,
        ignition: pos.ignition,
        online: device.online,
        sos: pos.sos_event,
        label: device.name,
        onClick: () => selectDevice(device.id),
      })
    })
  }, [devices, selectDevice])

  // Fly to selected device
  useEffect(() => {
    if (!selectedId || !adapterRef.current) return
    const device = devices[selectedId]
    if (device?.position) {
      adapterRef.current.flyTo(device.position.lat, device.position.lng, 14)
    }
  }, [selectedId, devices])

  return (
    <div className="page flex" style={{ flexDirection: 'row' }}>
      {/* Device Panel */}
      <DevicePanel />

      {/* Map */}
      <div className="map-container flex-1 relative">
        <div ref={mapContainerRef} className="h-full w-full" />
        <MapStatsBar />
      </div>
    </div>
  )
}
