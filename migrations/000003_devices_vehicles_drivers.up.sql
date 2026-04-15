-- 000003: Devices, Vehicles, Drivers
CREATE TABLE devices (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    imei          TEXT NOT NULL,
    name          TEXT NOT NULL,
    protocol      TEXT NOT NULL DEFAULT 'teltonika',  -- teltonika, gt06, tk103, jt808, ais140
    vehicle_id    UUID,  -- FK added after vehicles table
    firmware_ver  TEXT,
    sim_iccid     TEXT,
    phone_number  TEXT,
    last_seen_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    UNIQUE(tenant_id, imei)
);

CREATE INDEX idx_devices_tenant ON devices (tenant_id);
CREATE INDEX idx_devices_imei   ON devices (imei);

ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON devices
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::UUID OR
           current_setting('app.role', TRUE) = 'super_admin');

-- ── Vehicles ──────────────────────────────────────────────────────────────────
CREATE TABLE vehicles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    registration    TEXT NOT NULL,
    make            TEXT,
    model           TEXT,
    year            INT,
    fuel_type       TEXT DEFAULT 'diesel',
    device_id       UUID REFERENCES devices(id) ON DELETE SET NULL,
    driver_id       UUID,  -- FK added after drivers table
    group_id        UUID,
    odometer_offset BIGINT NOT NULL DEFAULT 0,  -- metres
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE(tenant_id, registration)
);

CREATE INDEX idx_vehicles_tenant    ON vehicles (tenant_id);
CREATE INDEX idx_vehicles_device_id ON vehicles (device_id);

ALTER TABLE vehicles ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON vehicles
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::UUID OR
           current_setting('app.role', TRUE) = 'super_admin');

-- Add device → vehicle FK
ALTER TABLE devices ADD FOREIGN KEY (vehicle_id) REFERENCES vehicles(id) ON DELETE SET NULL;

-- ── Drivers ───────────────────────────────────────────────────────────────────
CREATE TABLE drivers (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    license_number  TEXT,
    license_expiry  DATE,
    rfid_uid        TEXT,  -- iButton or RFID for IO 238
    phone           TEXT,
    email           TEXT,
    score_current   FLOAT NOT NULL DEFAULT 100.0,  -- 0–100
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_drivers_tenant   ON drivers (tenant_id);
CREATE INDEX idx_drivers_rfid_uid ON drivers (rfid_uid);

ALTER TABLE drivers ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON drivers
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::UUID OR
           current_setting('app.role', TRUE) = 'super_admin');

-- Add driver → vehicle FK
ALTER TABLE vehicles ADD FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE SET NULL;
