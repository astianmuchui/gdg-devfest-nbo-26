CREATE KEYSPACE IF NOT EXISTS iot
  WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '1'}
  AND DURABLE_WRITES = true;

USE iot;

CREATE TABLE IF NOT EXISTS telemetry (
    device_id       TEXT,
    created_at      TIMESTAMP,
    device_ts       BIGINT,
    uptime_s        BIGINT,
    temp_c          DOUBLE,
    humidity_pct    DOUBLE,
    air_quality_ppm DOUBLE,
    air_quality_v   DOUBLE,
    PRIMARY KEY ((device_id), created_at)
) WITH CLUSTERING ORDER BY (created_at DESC);
