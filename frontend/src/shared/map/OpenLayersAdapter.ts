import Map from 'ol/Map'
import View from 'ol/View'
import TileLayer from 'ol/layer/Tile'
import VectorLayer from 'ol/layer/Vector'
import VectorSource from 'ol/source/Vector'
import OSM from 'ol/source/OSM'
import Feature from 'ol/Feature'
import Point from 'ol/geom/Point'
import LineString from 'ol/geom/LineString'
import { fromLonLat } from 'ol/proj'
import { Style, Fill, Stroke, Circle as CircleStyle, Text } from 'ol/style'
import Overlay from 'ol/Overlay'
import type { MapAdapter, MarkerOptions } from './MapAdapter'

/**
 * OpenLayers implementation of MapAdapter.
 * Uses OSM tiles — no token required.
 * Swap to Mapbox/Google by implementing MapAdapter with their SDK.
 */
export class OpenLayersAdapter implements MapAdapter {
  private map: Map | null = null
  private markerSource = new VectorSource()
  private pathSource = new VectorSource()
  private popupElement: HTMLElement | null = null
  private popupOverlay: Overlay | null = null

  init(container: HTMLElement): void {
    // Create popup element
    this.popupElement = document.createElement('div')
    this.popupElement.className = 'device-popup'
    this.popupElement.style.display = 'none'
    container.appendChild(this.popupElement)

    this.popupOverlay = new Overlay({
      element: this.popupElement,
      positioning: 'bottom-center',
      stopEvent: false,
      offset: [0, -16],
    })

    this.map = new Map({
      target: container,
      layers: [
        // Base tile layer — CartoDB dark matter for a sleek dark look
        new TileLayer({
          source: new OSM({
            url: 'https://{a-c}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png',
            attributions: '© <a href="https://carto.com">CARTO</a>',
          }),
          className: 'base-tile',
        }),
        // Path layer
        new VectorLayer({
          source: this.pathSource,
          zIndex: 1,
          style: (feature) =>
            new Style({
              stroke: new Stroke({
                color: feature.get('color') || '#00D4FF',
                width: 3,
                lineDash: [6, 3],
              }),
            }),
        }),
        // Marker layer
        new VectorLayer({
          source: this.markerSource,
          zIndex: 10,
          style: (feature) => this.markerStyle(feature as Feature),
        }),
      ],
      overlays: [this.popupOverlay!],
      view: new View({
        center: fromLonLat([78.9629, 20.5937]), // Center of India
        zoom: 5,
      }),
      controls: [],
    })

    // Click handler for popups
    this.map.on('click', (evt) => {
      const feature = this.map!.forEachFeatureAtPixel(evt.pixel, (f) => f)
      if (feature && feature.get('type') === 'device') {
        const geom = feature.getGeometry() as Point
        this.popupOverlay!.setPosition(geom.getCoordinates())
        this.showPopup(
          feature.get('label') || feature.get('device_id'),
          feature.get('speed'),
          feature.get('heading'),
          feature.get('ignition'),
        )
        feature.get('onClick')?.()
      } else {
        this.popupOverlay!.setPosition(undefined)
        if (this.popupElement) this.popupElement.style.display = 'none'
      }
    })

    // Pointer cursor on hover
    this.map.on('pointermove', (evt) => {
      const hit = this.map!.hasFeatureAtPixel(evt.pixel)
      container.style.cursor = hit ? 'pointer' : ''
    })
  }

  updateMarker(deviceId: string, opts: MarkerOptions): void {
    let feature = this.markerSource.getFeatureById(deviceId) as Feature | null

    if (!feature) {
      feature = new Feature({ type: 'device', device_id: deviceId })
      feature.setId(deviceId)
      this.markerSource.addFeature(feature)
    }

    feature.setGeometry(new Point(fromLonLat([opts.lng, opts.lat])))
    feature.setProperties({
      type: 'device',
      device_id: deviceId,
      heading: opts.heading ?? 0,
      speed: opts.speed ?? 0,
      ignition: opts.ignition ?? false,
      online: opts.online ?? true,
      sos: opts.sos ?? false,
      label: opts.label ?? deviceId,
      onClick: opts.onClick,
    })
  }

  removeMarker(deviceId: string): void {
    const feature = this.markerSource.getFeatureById(deviceId)
    if (feature) this.markerSource.removeFeature(feature as Feature)
  }

  flyTo(lat: number, lng: number, zoom = 14): void {
    this.map?.getView().animate({
      center: fromLonLat([lng, lat]),
      zoom,
      duration: 600,
    })
  }

  drawPath(coordinates: [number, number][], color = '#00D4FF'): void {
    const lineCoords = coordinates.map(([lat, lng]) => fromLonLat([lng, lat]))
    const feature = new Feature(new LineString(lineCoords))
    feature.set('color', color)
    this.pathSource.addFeature(feature)
  }

  clearPaths(): void {
    this.pathSource.clear()
  }

  destroy(): void {
    this.map?.setTarget(undefined as unknown as HTMLElement)
    this.map = null
  }

  // ── Private helpers ────────────────────────────────────────────────────────

  private markerStyle(feature: Feature): Style {
    const speed = feature.get('speed') as number
    const ignition = feature.get('ignition') as boolean
    const online = feature.get('online') as boolean
    const sos = feature.get('sos') as boolean
    const heading = feature.get('heading') as number

    // Color based on state
    const color = sos
      ? '#FF4D6A'
      : !online
      ? '#4A6480'
      : ignition && speed > 0
      ? '#00D4FF'
      : ignition
      ? '#00E599'
      : '#FFB84D'

    return new Style({
      image: new CircleStyle({
        radius: 8,
        fill: new Fill({ color }),
        stroke: new Stroke({ color: 'rgba(255,255,255,0.9)', width: 2 }),
      }),
      text: new Text({
        text: ignition ? `${speed}` : '',
        font: '600 9px JetBrains Mono, monospace',
        fill: new Fill({ color: '#080D1A' }),
        offsetY: 1,
      }),
    })

    void heading // Used for future arrow rotation
  }

  private showPopup(name: string, speed: number, heading: number, ignition: boolean): void {
    if (!this.popupElement) return
    this.popupElement.style.display = 'block'
    this.popupElement.innerHTML = `
      <div class="device-popup-header">
        <div class="dot ${ignition ? 'dot-online' : 'dot-offline'}"></div>
        <div class="device-popup-name">${escapeHtml(name)}</div>
      </div>
      <div class="device-popup-grid">
        <div class="device-popup-field">
          <div class="device-popup-key">Speed</div>
          <div class="device-popup-val">${speed} km/h</div>
        </div>
        <div class="device-popup-field">
          <div class="device-popup-key">Heading</div>
          <div class="device-popup-val">${heading}°</div>
        </div>
        <div class="device-popup-field">
          <div class="device-popup-key">Ignition</div>
          <div class="device-popup-val">${ignition ? 'ON' : 'OFF'}</div>
        </div>
      </div>
    `
  }
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

/**
 * Stub adapters — swap in by passing to LiveMap
 */
export const MapboxAdapterStub = {
  // TODO: implement MapboxAdapter with Mapbox GL JS
  // Requires: VITE_MAPBOX_TOKEN env var
  // Reference: https://docs.mapbox.com/mapbox-gl-js/api/
  _note: 'Mapbox adapter — set VITE_MAPBOX_TOKEN and implement MapAdapter interface',
}

export const GoogleMapsAdapterStub = {
  // TODO: implement GoogleMapsAdapter
  // Reference: https://developers.google.com/maps/documentation/javascript
  _note: 'Google Maps adapter — requires API key and Maps JS SDK',
}

export const MapmyIndiaAdapterStub = {
  // TODO: implement MapmyIndia/Mappls adapter
  // Reference: https://about.mappls.com/api/
  _note: 'MapmyIndia (Mappls) adapter — requires API key',
}
