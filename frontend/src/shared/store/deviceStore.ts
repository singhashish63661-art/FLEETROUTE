import { create } from 'zustand'

export interface DevicePosition {
  device_id: string
  tenant_id: string
  timestamp: number
  lat: number
  lng: number
  altitude: number
  speed: number
  heading: number
  satellites: number
  valid: boolean
  ignition: boolean
  movement: boolean
  external_voltage?: number
  battery_level?: number
  gsm_signal?: number
  sos_event?: boolean
}

interface Device {
  id: string
  imei: string
  name: string
  protocol: string
  vehicle_id?: string
  online: boolean
  last_seen_at?: string
  position?: DevicePosition
}

interface DeviceStore {
  devices: Record<string, Device>
  selectedDeviceId: string | null

  setDevices: (devices: Device[]) => void
  updatePosition: (pos: DevicePosition) => void
  selectDevice: (id: string | null) => void
}

export const useDeviceStore = create<DeviceStore>((set) => ({
  devices: {},
  selectedDeviceId: null,

  setDevices: (devices) =>
    set({
      devices: Object.fromEntries(devices.map((d) => [d.id, d])),
    }),

  updatePosition: (pos) =>
    set((state) => {
      const device = state.devices[pos.device_id]
      if (!device) return state
      return {
        devices: {
          ...state.devices,
          [pos.device_id]: {
            ...device,
            online: true,
            position: pos,
          },
        },
      }
    }),

  selectDevice: (id) => set({ selectedDeviceId: id }),
}))
