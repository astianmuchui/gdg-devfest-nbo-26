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

## v4-tuned: same architecture, less work per message

v4 keeps v3's shards, bounded queues and batched writers, and cuts the work per message:

- The Paho handler slices the device ID out of the topic and hashes it. No JSON, no allocation.
  Shard workers do `json.Unmarshal`.
- One reusable grouper per worker instead of a fresh map per flush.
- `IOT_WRITES_PER_SHARD` long-lived writer goroutines per shard instead of a goroutine per group.
- gocql queries are released back to the pool after `Exec`.
- `cmd/default.pgo` is a CPU profile from the standard load. `go build` uses it automatically.

### Benchmark runs

```sh
./run.sh <label> <seconds> [extra env]
./run.sh v4 40
./run.sh v4-tuned 40 "GOGC=400 GOMEMLIMIT=1GiB"
```

`run.sh` builds both binaries, runs the ingester on an isolated topic (`bench/+/readings`, port 8090),
drives it with the producer (`DEVICES`, `INTERVAL`, `CONNS` override the 50,000 / 5s / 20 defaults),
and writes `results/<label>.log` plus a steady-state summary.

The report line includes Go runtime numbers:

| Field        | Meaning                                                    |
|--------------|------------------------------------------------------------|
| `cpu_ms/1k`  | process user+sys CPU ms per 1,000 stored messages          |
| `allocs/msg` | heap objects allocated per stored message                  |
| `alloc_MB/s` | heap bytes allocated per second                            |
| `gc/s`       | GC cycles per second                                       |
| `gc_cpu`     | share of process CPU spent in GC                           |

### Profiling

pprof listens on `localhost:6060` (`IOT_PPROF_ADDR`). Under load:

```sh
curl -o cpu.pprof 'localhost:6060/debug/pprof/profile?seconds=30'
curl -o allocs.pprof 'localhost:6060/debug/pprof/allocs'
go tool pprof -top bin/ingester cpu.pprof
go tool pprof -sample_index=alloc_objects -top bin/ingester allocs.pprof
```

To refresh PGO, capture a 30 s CPU profile under the standard load and save it as `cmd/default.pgo`.

### Microbenchmarks

```sh
go test -run '^$' -bench . -benchmem ./internal/pipeline/
```

| Benchmark                              | ns/op  | B/op    | allocs/op |
|----------------------------------------|--------|---------|-----------|
| Receive, v3 parse on dispatcher        | 2,090  | 336     | 8         |
| Receive, v4 topic slice + hash         | 12.6   | 0       | 0         |
| Group 500 rows (50k fleet), fresh map  | 66,400 | 140,440 | 515       |
| Group 500 rows (50k fleet), sort+runs  | 68,200 | 1       | 0         |
| Group 500 rows (50k fleet), grouper    | 14,100 | 0       | 0         |
| Group 500 rows (50 devices), fresh map | 43,000 | 129,800 | 242       |
| Group 500 rows (50 devices), sort+runs | 50,600 | 0       | 0         |
| Group 500 rows (50 devices), grouper   | 10,900 | 1       | 0         |

### Results at 10,000 msg/s (50,000 devices every 5 s)

| Run                              | allocs/msg | alloc_MB/s | gc/s | gc_cpu | max Sys | cpu_ms/1k |
|----------------------------------|-----------:|-----------:|-----:|-------:|--------:|----------:|
| v3                               | 75.3       | 53.1       | 7.5  | 6.3%   | 34 MB   | 77.9      |
| v4 receive path (step 2)         | 75.3       | 53.2       | 13.3 | 11.2%  | 30 MB   | 92.8      |
| v4 + grouper (step 3)            | 74.2       | 50.9       | 11.7 | 10.2%  | 30 MB   | 87.4      |
| v4 + long-lived writers (step 4) | 73.1       | 50.1       | 11.8 | 10.0%  | 30 MB   | 79.5      |
| v4 + query release (step 5)      | 73.2       | 50.2       | 11.8 | 10.3%  | 26 MB   | 82.8      |
| v4, defaults (2 runs)            | 73.2–73.6  | 50.2       | 11.3–11.8 | 10.2–12.0% | 26–30 MB | 81.2–114.6 |
| v4, GOGC=400 GOMEMLIMIT=1GiB (2) | 73.0–73.1  | 50.0       | 2.2–2.3 | 2.2–2.6% | 47–51 MB | 76.8–98.4 |
| v4 + PGO, defaults (2 runs)      | 73.2       | 50.2       | 11.3–11.5 | 9.9–10.1% | 26–30 MB | 80.1–84.3 |
| v4 + PGO, GOGC=400 GOMEMLIMIT=1GiB (2) | 73.0 | 50.0     | 2.2–2.3 | 2.2–4.1% | 47–51 MB | 80.7–85.2 |

CPU per message varies by up to ±20% between identical runs on one machine that also runs ScyllaDB,
Mosquitto and the producer. Allocation and GC numbers are stable to within 1%.
