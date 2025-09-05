# gRPC Device Logging

Minimal gRPC service in Go that records device metadata to PostgreSQL and exposes Prometheus metrics ready for Grafana. Includes protobuf definitions, Buf-based code generation, and an example dashboard.

## Features

- gRPC server with `CreateDevice` and `GetDevices` RPCs
- PostgreSQL storage via `pgx` connection pool
- Prometheus metrics (`/metrics`) with histogram and error counters
- Grafana dashboard JSON included
- Config via YAML (ports, DB credentials, pool size)
- Protobuf definitions with Buf for linting and codegen

## Project Layout

- `cmd/server/main.go` — service bootstrap and gRPC registration
- `config/` — YAML config loader (`config.go`)
- `db/` — database connector (`db.go`)
- `device/` — domain model and DB operations (`device.go`)
- `metrics/` — Prometheus collectors and HTTP exporter (`metrics.go`)
- `proto/` — protobuf source files
- `gen/` — generated Go code from protobufs (via Buf)
- `grafana.json` — example Grafana dashboard for Prometheus metrics

## Prerequisites

- Go 1.25+
- PostgreSQL 14+ (reachable from the server)
- Buf CLI (for proto codegen): https://buf.build/docs/installation
- Optional: Prometheus and Grafana for metrics visualization

## Configuration

Copy the example and adjust values:

```bash
cp config.yaml.example config.yaml
```

Config fields (see `config/Config`):

- `debug` (bool): enable verbose logging
- `appPort` (int): gRPC server port (default example: 50051)
- `metricsPort` (int): Prometheus HTTP exporter port (default example: 9090)
- `database.user`/`password`/`host`/`database`/`maxConnections`

Note: `db.DbConnect` builds a URL like `postgres://user:pass@host:5432/db?pool_max_conns=N`. Ensure `host` includes hostname (and optional port). The function currently forces port `5432`.

## Database

Create table before running the server:

```sql
CREATE TABLE IF NOT EXISTS grpc_device (
  id SERIAL PRIMARY KEY,
  uuid TEXT NOT NULL,
  mac TEXT NOT NULL,
  firmware TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
```

## Build & Run

```bash
go mod download
go build -o bin/server ./cmd/server
./bin/server -config ./config.yaml
```

On Windows PowerShell:

```powershell
go build -o bin/server.exe .\cmd\server
./bin/server.exe -config .\config.yaml
```

The gRPC server listens on `:${appPort}`. Prometheus metrics are exposed on `http://localhost:${metricsPort}/metrics`.

## Protobufs & Code Generation

This repo uses Buf for linting and generation.

- Edit proto files in `proto/device/v1/proto.proto`.
- Generate Go code into `gen/`:

```bash
buf generate
```

Buf configs: `buf.yaml` (lint/breaking) and `buf.gen.yaml` (plugins). The Go package prefix is `github.com/yaninyzwitty/grpc-device-logging/gen`.

## API

Service: `device.v1.CloudService`

- CreateDevice

  - Request: `{ mac, firmware }`
  - Response: `{ device }` where `device` has `{ id, uuid, mac, firmware, created_at, updated_at }`

- GetDevices
  - Request: `{}`
  - Response: `{ devices: Device[] }`

Example `protocurl`/grpcurl invocation:

```bash
grpcurl -plaintext -d '{"mac":"AA-BB-CC-11-22-33","firmware":"1.0.0"}' localhost:50051 device.v1.CloudService/CreateDevice
```

```bash
grpcurl -plaintext -d '{}' localhost:50051 device.v1.CloudService/GetDevices
```

Note: `GetDevices` in `cmd/server/main.go` currently returns static sample data for demonstration.

## Metrics

Prometheus collectors (namespace `myapp`):

- `myapp_stage` (gauge)
- `myapp_request_duration_seconds{op,db}` (histogram)
- `myapp_errors_total{op,db}` (counter)

Scrape endpoint: `http://localhost:${metricsPort}/metrics`.

### Grafana Dashboard

Import `grafana.json` into Grafana. It includes panels for latency percentiles, error rates, and request volume. Ensure your Prometheus datasource is connected and scraping this service.

## Logging

When `debug: true`, the server sets slog level to debug. Errors are annotated and counted in Prometheus where applicable.

## Development Notes

- Module path: `github.com/yaninyzwitty/grpc-device-logging`
- Requires regeneration when changing protobufs (`buf generate`)
- Keep `config.yaml` outside version control or avoid committing secrets

## Troubleshooting

- Connection refused to PostgreSQL: verify `database.host`, firewall rules, and that Postgres allows remote connections.
- Unable to generate code: ensure Buf CLI is installed and `buf.yaml`/`buf.gen.yaml` exist at repo root.
- Metrics not visible: confirm `/metrics` endpoint, Prometheus scrape config, and matching ports.

## License

MIT (or update as appropriate).
