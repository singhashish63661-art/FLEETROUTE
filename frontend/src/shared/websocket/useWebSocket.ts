import { useEffect, useRef, useCallback } from 'react'
import { useAuthStore } from '../store/authStore'
import { useDeviceStore, DevicePosition } from '../store/deviceStore'

const WS_BASE_URL = import.meta.env.VITE_WS_URL || ''

// Exponential backoff constants
const INITIAL_DELAY = 1_000
const MAX_DELAY = 30_000
const BACKOFF_MULTIPLIER = 2

export function useWebSocket() {
  const ws = useRef<WebSocket | null>(null)
  const reconnectDelay = useRef(INITIAL_DELAY)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isUnmounted = useRef(false)

  const accessToken = useAuthStore((s) => s.accessToken)
  const updatePosition = useDeviceStore((s) => s.updatePosition)

  const buildWSURL = useCallback((token: string) => {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const hostOrigin = `${proto}//${window.location.host}`
    const rawBase = WS_BASE_URL.trim()

    const normalized = (() => {
      if (!rawBase) return new URL(hostOrigin)
      if (rawBase.startsWith('ws://') || rawBase.startsWith('wss://')) {
        return new URL(rawBase)
      }
      if (rawBase.startsWith('http://') || rawBase.startsWith('https://')) {
        const u = new URL(rawBase)
        u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
        return u
      }
      return new URL(rawBase, hostOrigin)
    })()

    const path = normalized.pathname.replace(/\/+$/, '')
    normalized.pathname = path.endsWith('/ws/v1/live')
      ? path
      : path
        ? `${path}/ws/v1/live`
        : '/ws/v1/live'
    normalized.searchParams.set('token', token)
    return normalized.toString()
  }, [])

  const connect = useCallback(() => {
    if (!accessToken || isUnmounted.current) return

    const url = buildWSURL(accessToken)
    const socket = new WebSocket(url)
    ws.current = socket

    socket.onopen = () => {
      console.log('[WS] connected')
      reconnectDelay.current = INITIAL_DELAY
    }

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data) as { type: string } & DevicePosition
        if (msg.type === 'position') {
          updatePosition(msg)
        }
      } catch {
        // ignore malformed messages
      }
    }

    socket.onclose = (e) => {
      if (isUnmounted.current) return
      console.log(`[WS] closed (${e.code}), reconnecting in ${reconnectDelay.current}ms`)
      reconnectTimer.current = setTimeout(() => {
        if (!isUnmounted.current) connect()
      }, reconnectDelay.current)
      reconnectDelay.current = Math.min(
        reconnectDelay.current * BACKOFF_MULTIPLIER,
        MAX_DELAY,
      )
    }

    socket.onerror = () => {
      socket.close()
    }
  }, [accessToken, updatePosition, buildWSURL])

  // Subscribe to specific devices
  const subscribe = useCallback((deviceIds: string[]) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({ action: 'subscribe', devices: deviceIds }))
    }
  }, [])

  useEffect(() => {
    isUnmounted.current = false
    connect()
    return () => {
      isUnmounted.current = true
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current)
      ws.current?.close()
    }
  }, [connect])

  return { subscribe }
}
