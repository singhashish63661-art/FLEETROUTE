-- 000006: Trips and Alerts/Rules
CREATE TABLE trips (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id       TEXT NOT NULL,
    vehicle_id      UUID REFERENCES vehicles(id),
    driver_id       UUID REFERENCES drivers(id),

    -- Trip bounds
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,

    -- Start/end positions
    start_lat       DOUBLE PRECISION,
    start_lng       DOUBLE PRECISION,
    start_address   TEXT,
    end_lat         DOUBLE PRECISION,
    end_lng         DOUBLE PRECISION,
    end_address     TEXT,

    -- Metrics
    distance_m      BIGINT NOT NULL DEFAULT 0,    -- metres
    duration_s      INT NOT NULL DEFAULT 0,        -- seconds
    max_speed       SMALLINT NOT NULL DEFAULT 0,   -- km/h
    avg_speed       FLOAT NOT NULL DEFAULT 0,
    idle_time_s     INT NOT NULL DEFAULT 0,
    fuel_used_l     FLOAT,
    driver_score    FLOAT,

    -- Harsh events
    harsh_accel     INT NOT NULL DEFAULT 0,
    harsh_brake     INT NOT NULL DEFAULT 0,
    harsh_corner    INT NOT NULL DEFAULT 0,
    overspeed_count INT NOT NULL DEFAULT 0,

    -- Path (GeoJSON LineString — stored for playback)
    path            GEOMETRY(LINESTRING, 4326),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trips_tenant    ON trips (tenant_id, started_at DESC);
CREATE INDEX idx_trips_device    ON trips (device_id, started_at DESC);
CREATE INDEX idx_trips_vehicle   ON trips (vehicle_id, started_at DESC);

-- ── Alert Rules ───────────────────────────────────────────────────────────────
CREATE TABLE alert_rules (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    template_id  TEXT,  -- reference to built-in template
    conditions   JSONB NOT NULL,   -- AND/OR condition tree
    actions      JSONB NOT NULL,   -- action definitions
    cooldown_s   INT NOT NULL DEFAULT 300,  -- min seconds between re-triggers
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rules_tenant ON alert_rules (tenant_id);

-- ── Alerts (triggered events) ─────────────────────────────────────────────────
CREATE TABLE alerts (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id         TEXT NOT NULL,
    rule_id           UUID REFERENCES alert_rules(id),
    alert_type        TEXT NOT NULL,
    severity          TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('info','warning','critical')),
    message           TEXT NOT NULL,
    lat               DOUBLE PRECISION,
    lng               DOUBLE PRECISION,
    speed             SMALLINT,
    io_snapshot       JSONB,
    triggered_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at   TIMESTAMPTZ,
    acknowledged_by   UUID REFERENCES users(id)
);

CREATE INDEX idx_alerts_tenant       ON alerts (tenant_id, triggered_at DESC);
CREATE INDEX idx_alerts_device       ON alerts (device_id, triggered_at DESC);
CREATE INDEX idx_alerts_unacknowledged ON alerts (tenant_id) WHERE acknowledged_at IS NULL;
