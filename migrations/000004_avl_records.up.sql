-- 000004: AVL records hypertable (TimescaleDB)
CREATE TABLE avl_records (
    -- Identity
    device_id         TEXT NOT NULL,
    tenant_id         TEXT NOT NULL,

    -- Timing
    timestamp         TIMESTAMPTZ NOT NULL,
    received_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Position
    lat               DOUBLE PRECISION NOT NULL,
    lng               DOUBLE PRECISION NOT NULL,
    altitude          INT NOT NULL DEFAULT 0,
    speed             SMALLINT NOT NULL DEFAULT 0,
    heading           SMALLINT NOT NULL DEFAULT 0,
    satellites        SMALLINT NOT NULL DEFAULT 0,
    valid             BOOLEAN NOT NULL DEFAULT FALSE,

    -- Status flags
    priority          SMALLINT NOT NULL DEFAULT 0,
    raw_codec         SMALLINT NOT NULL DEFAULT 8,

    -- Named telemetry (pre-decoded for query performance)
    ignition          BOOLEAN NOT NULL DEFAULT FALSE,
    movement          BOOLEAN NOT NULL DEFAULT FALSE,
    external_voltage  FLOAT,    -- V
    battery_voltage   FLOAT,    -- V
    battery_level     SMALLINT, -- %
    gnss_status       SMALLINT,
    gsm_signal        SMALLINT,
    engine_rpm        INT,
    fuel_level        SMALLINT, -- %
    temperature_1     FLOAT,    -- °C
    user_id           TEXT,     -- iButton / RFID
    sos_event         BOOLEAN NOT NULL DEFAULT FALSE,
    can_speed         SMALLINT,

    -- Raw IO dump (preserved for rule engine and custom queries)
    io_data           JSONB,

    -- Composite primary key includes timestamp for hypertable partitioning
    PRIMARY KEY (device_id, timestamp)
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('avl_records', 'timestamp',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Enable compression after 7 days
ALTER TABLE avl_records SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_id, tenant_id',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('avl_records', INTERVAL '7 days');

-- Retention: drop data older than 1 year (configurable)
SELECT add_retention_policy('avl_records', INTERVAL '365 days');

-- ── Indexes for common query patterns ────────────────────────────────────────
-- Note: TimescaleDB manages time-based chunk indexes automatically.
CREATE INDEX idx_avl_device_time  ON avl_records (device_id, timestamp DESC);
CREATE INDEX idx_avl_tenant_time  ON avl_records (tenant_id, timestamp DESC);
CREATE INDEX idx_avl_position     ON avl_records USING GIST (
    ST_SetSRID(ST_MakePoint(lng, lat), 4326)
) WHERE valid = TRUE;  -- spatial index for geo queries
CREATE INDEX idx_avl_sos          ON avl_records (tenant_id, timestamp DESC) WHERE sos_event = TRUE;
CREATE INDEX idx_avl_ignition_off ON avl_records (device_id, timestamp) WHERE ignition = FALSE;
