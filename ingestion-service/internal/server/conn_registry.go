package server

import (
	"net"
	"sync"
)

// ConnRegistry is a thread-safe map of deviceID → active net.Conn.
// Used for bidirectional command dispatch (immobilize, config push, etc.)
type ConnRegistry struct {
	mu    sync.RWMutex
	conns map[string]net.Conn
}

// NewConnRegistry creates an empty registry.
func NewConnRegistry() *ConnRegistry {
	return &ConnRegistry{conns: make(map[string]net.Conn)}
}

// Register associates a deviceID with a connection.
// If a device reconnects, the old connection is replaced.
func (r *ConnRegistry) Register(deviceID string, conn net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.conns[deviceID]; ok {
		old.Close() //nolint:errcheck
	}
	r.conns[deviceID] = conn
}

// Unregister removes a device from the registry.
func (r *ConnRegistry) Unregister(deviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, deviceID)
}

// Send writes data to a device's active connection.
// Returns an error if the device is not connected.
func (r *ConnRegistry) Send(deviceID string, data []byte) error {
	r.mu.RLock()
	conn, ok := r.conns[deviceID]
	r.mu.RUnlock()
	if !ok {
		return &DeviceNotConnectedError{DeviceID: deviceID}
	}
	_, err := conn.Write(data)
	return err
}

// Connected returns true if the device has an active connection.
func (r *ConnRegistry) Connected(deviceID string) bool {
	r.mu.RLock()
	_, ok := r.conns[deviceID]
	r.mu.RUnlock()
	return ok
}

// Count returns the number of active connections.
func (r *ConnRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.conns)
}

// DeviceNotConnectedError is returned when a command targets an offline device.
type DeviceNotConnectedError struct {
	DeviceID string
}

func (e *DeviceNotConnectedError) Error() string {
	return "device not connected: " + e.DeviceID
}
