import { MapAdapter, MarkerOptions } from './MapAdapter'

// MapmyIndiaAdapter — provider adhering to India government mapping compliance
// Note: Requires mappls-web-maps script injected in index.html
export class MapMyIndiaAdapter implements MapAdapter {
  private map: any
  private markers: Map<string, any> = new Map()

  init(container: HTMLElement): void {
    // Mock MapmyIndia (mappls) initialization
    // @ts-ignore
    if (typeof mappls === 'undefined') {
      console.warn('MapmyIndia SDK not loaded. Mocking map instance.')
    }
  }

  updateMarker(deviceId: string, opts: MarkerOptions): void {
    console.debug(`[MapmyIndia] updateMarker ${deviceId}`, opts)
    this.markers.set(deviceId, opts)
  }

  removeMarker(deviceId: string): void {
    console.debug(`[MapmyIndia] removeMarker ${deviceId}`)
    this.markers.delete(deviceId)
  }

  flyTo(lat: number, lng: number, zoom?: number): void {
    console.debug(`[MapmyIndia] flyTo lat=${lat} lng=${lng} zoom=${zoom}`)
  }

  drawPath(coordinates: [number, number][], color?: string): void {
    console.debug(`[MapmyIndia] drawPath with ${coordinates.length} points`)
  }

  clearPaths(): void {
    console.debug(`[MapmyIndia] clearPaths`)
  }

  destroy(): void {
    console.debug(`[MapmyIndia] destroy`)
  }
}
