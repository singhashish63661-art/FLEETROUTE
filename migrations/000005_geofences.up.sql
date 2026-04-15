-- 000005: Geofences (PostGIS)
CREATE TABLE geofences (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    shape_type  TEXT NOT NULL CHECK (shape_type IN ('circle', 'polygon', 'corridor')),
    geometry    GEOMETRY(GEOMETRY, 4326) NOT NULL,  -- PostGIS geometry
    radius_m    FLOAT,   -- for circle shapes only
    color       TEXT DEFAULT '#3B82F6',
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- Spatial index for efficient containment checks
CREATE INDEX idx_geofences_geometry ON geofences USING GIST (geometry);
CREATE INDEX idx_geofences_tenant   ON geofences (tenant_id);

ALTER TABLE geofences ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON geofences
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::UUID OR
           current_setting('app.role', TRUE) = 'super_admin');

-- ── Geofence Events (dwell tracking) ─────────────────────────────────────────
CREATE TABLE geofence_events (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    geofence_id  UUID NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    device_id    TEXT NOT NULL,
    vehicle_id   UUID REFERENCES vehicles(id),
    event_type   TEXT NOT NULL CHECK (event_type IN ('entry', 'exit')),
    triggered_at TIMESTAMPTZ NOT NULL,
    duration_s   INT,  -- dwell time in seconds (populated on exit)
    position     GEOMETRY(POINT, 4326),
    speed        SMALLINT
);

CREATE INDEX idx_geofence_events_tenant    ON geofence_events (tenant_id, triggered_at DESC);
CREATE INDEX idx_geofence_events_geofence  ON geofence_events (geofence_id, triggered_at DESC);
CREATE INDEX idx_geofence_events_device    ON geofence_events (device_id, triggered_at DESC);
