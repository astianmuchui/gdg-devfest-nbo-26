# High-Throughput IoT with Go — DevFest Nairobi 2026

Demo code for *Building High-Throughput, Low-Latency IoT Systems with Go*.

## v1-naive: Receive → Store

The MQTT handler parses each reading and writes it to ScyllaDB synchronously.
Storage latency becomes ingestion latency.

```
producer ──MQTT──▶ mosquitto ──▶ ingester (handler → INSERT) ──▶ ScyllaDB
```

## Run

```sh
cp .env.example .env
docker compose up -d
go run ./cmd
go run ./cmd/producer -devices 50000 -interval 5s -conns 20
```

If a native ScyllaDB already listens on 9042, set `SCYLLA_HOST_PORT` and
`IOT_CASSANDRA_PORT` to another port (e.g. 9043) in `.env`.

| Service    | URL                                  |
|------------|--------------------------------------|
| Grafana    | http://localhost:3000 (admin/admin)  |
| Prometheus | http://localhost:9090                |
| MQTT       | tcp://localhost:1883                 |
| API        | http://localhost:8080                |

Grafana opens on the provisioned **Node Exporter Full** dashboard.
Prometheus also scrapes ScyllaDB metrics (`job="scylla"`).

## Producer flags

| Flag        | Default                | Meaning                                  |
|-------------|------------------------|------------------------------------------|
| `-devices`  | 1000                   | simulated devices                        |
| `-interval` | 5s                     | time between readings per device         |
| `-conns`    | 10                     | MQTT connections devices are spread over |
| `-qos`      | 1                      | MQTT QoS                                 |
| `-prefix`   | devices                | topic: `<prefix>/<device_id>/readings`   |
| `-duration` | 0                      | stop after this long (0 = until Ctrl+C)  |
| `-broker`   | tcp://localhost:1883   | broker URL                               |

## API

- `GET /health`
- `GET /api/v1/telemetry/latest?device_id=DEV00042`
- `GET /api/v1/telemetry/history?device_id=DEV00042&hours=1` (or `start`/`end`)
- `GET /api/v1/telemetry/count[?device_id=DEV00042]`
