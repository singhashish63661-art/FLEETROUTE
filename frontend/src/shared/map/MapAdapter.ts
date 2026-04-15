// MapAdapter — provider-agnostic interface for map implementations.
// Swap providers by implementing this interface and passing to LiveMap.
export interface MapAdapter {
  /** Initialize map into the given container element */
  init(container: HTMLElement): void

  /** Update or create a device marker at the given position */
  updateMarker(deviceId: string, opts: MarkerOptions): void

  /** Remove a device marker */
  removeMarker(deviceId: string): void

  /** Fly/pan to a position */
  flyTo(lat: number, lng: number, zoom?: number): void

  /** Draw a route path */
  drawPath(coordinates: [number, number][], color?: string): void

  /** Clear all paths */
  clearPaths(): void

  /** Destroy map and clean up */
  destroy(): void
}

export interface MarkerOptions {
  lat: number
  lng: number
  heading?: number
  speed?: number
  ignition?: boolean
  online?: boolean
  sos?: boolean
  label?: string
  onClick?: () => void
}
