# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o rtb-generator .

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a single test
go test -run TestName ./...

# Run server locally
./rtb-generator -server -port=8080 -scheduler-interval=1m -out-dir=./output

# Docker
docker compose up --build
```

MMDB integration tests (`TestIP_InitialGeoFromMMDB`) are skipped automatically when `data/GeoLite2-City.mmdb` is absent.

## Architecture

Single Go package (`main`). No subdirectories. All files compile together.

### File responsibilities

| File | Purpose |
|---|---|
| `models.go` | OpenRTB 2.5 structs (`BidRequest`, `Device`, `Geo`, `Imp`, etc.) |
| `generator.go` | Random bid request generation (`GenerateRandomBidRequestWithConfig`, `generateGeo`, `generateGeoNear`, `generateGeoInBBox`, `generateRequestForTask`) |
| `task.go` | `Task` struct, `TaskStore` (thread-safe JSON-file-backed store), `CriteriaType` constants, `GeoJSONGeometry` with bbox extraction |
| `scheduler.go` | `Scheduler` — ticker loop that calls `generateForDeviceTask` (ip/bbox) or ifa walking per active task, writes JSONL output files |
| `server.go` | HTTP handlers for CRUD task API; resolves initial geo at task creation via MMDB or `generateGeo()` |
| `geocoder.go` | `ReverseGeocoder` — Nominatim reverse geocoding with coordinate-keyed cache and 1 req/s rate limit |
| `main.go` | Flag parsing, CLI one-shot generation mode, `runServer` wiring |

### Data flow

**CLI mode**: flags → `GenerateRandomBidRequestWithConfig` → stdout. No tasks, no scheduler, no persistent state. `-mmdb` flag is parsed but unused in CLI mode.

**Server mode**:
1. `POST /tasks` → `Server.createTask` resolves initial `LastGeo` (MMDB lookup for `ip`, random city for `ifa`, nil for `bbox`) and persists to `TaskStore` (JSON file).
2. `Scheduler.Start()` ticks every `-scheduler-interval`:
   - `ip` / `bbox` tasks → `generateForDeviceTask`: persistent `Devices map[string]*Geo` pool, `count` slots distributed across devices (max `count/10` per device per tick), intra-tick walk ≤1 km per appearance, bbox-constrained for bbox tasks.
   - `ifa` tasks → single-location walk: `count` requests per tick, geo moves ≤2 km between consecutive requests, final position saved as `LastGeo`.
   - Each geo is enriched via `ReverseGeocoder.Enrich()` before writing.
   - Output: `{out-dir}/task_{correlation_id}_{unix_timestamp}.jsonl`

### Key design constraints

- **Geo state persistence**: `Task.LastGeo` and `Task.Devices` are persisted in `tasks.json` and survive restarts. The scheduler mutates task structs directly and calls `store.Add(task)` to persist after each tick.
- **Device pool for ip/bbox**: Initialised on the first scheduler tick. For `ip`: devices spread within 1 km of `LastGeo`. For `bbox`: devices at random points within the polygon's bounding box.
- **`generateRequestForTask`**: Accepts a `deviceIFA string` parameter — non-empty overrides the random IFA for `ip` and `bbox` tasks. `baseGeo` is always passed as `nil` from the scheduler (geo is set directly on `req.Device.Geo` after generation).
- **Nominatim rate limit**: `ReverseGeocoder` enforces ≥1 s between HTTP calls using a `lastCall` timestamp under mutex. Cache key is `"%.2f,%.2f"` (≈1 km resolution).
- **`TaskStore` thread safety**: `sync.RWMutex` guards all reads and writes; `save()` must be called with write lock held.
