# rtbgen

A Go CLI tool and HTTP service for generating random [OpenRTB 2.5](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf) bid requests. Useful for load testing ad exchanges, DSPs, and SSPs.

## Features

- Generates fully compliant OpenRTB 2.5 bid requests
- Supports banner, video, audio, and native impression types
- Timestamps randomized within a configurable sliding window
- HTTP service for managing long-running generation tasks
- Persistent task storage (JSON file)
- Background scheduler generating JSONL output for active tasks
- Three geo criteria modes: IP address, IFA, or GeoJSON bounding box
- IP-based geo lookup via MaxMind GeoIP2 MMDB
- Geo anchor persisted per task — location stays consistent across scheduler ticks and server restarts
- Generated locations scattered within 1 km radius of the anchor (IP/IFA) or randomly within the polygon (bbox)
- Docker and Docker Compose support

## Installation

```bash
git clone <repo>
cd rtbgen
go build -o rtb-generator .
```

Requires Go 1.22+.

## CLI Usage

```bash
./rtb-generator [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-type` | `random` | Request type: `site`, `app`, or `random` |
| `-imp` | `banner` | Impression type: `banner`, `video`, `audio`, `native` |
| `-count` | `1` | Number of requests to generate |
| `-test` | `false` | Set the test flag on generated requests |
| `-pretty` | `true` | Pretty-print JSON output |
| `-compact` | `false` | Compact JSON output (overrides `-pretty`) |
| `-bbox` | | Bounding box filter: `maxlat,maxlon,minlat,minlon` |
| `-scheduler-interval` | `5m` | Time window for randomized timestamps |
| `-examples` | `false` | Print example requests and exit |

### Examples

```bash
# Single random request
./rtb-generator

# 10 site banner requests, compact output
./rtb-generator -type=site -imp=banner -count=10 -compact

# App video requests with bounding box (New York area)
./rtb-generator -type=app -imp=video -count=5 -bbox="40.9,-73.7,40.5,-74.1"

# Use a 10-minute timestamp window
./rtb-generator -count=20 -scheduler-interval=10m
```

Each generated request includes an `ext.timestamp` field — a Unix millisecond timestamp randomized within `[now - scheduler-interval, now]`.

### CLI vs HTTP service

| Feature | CLI | HTTP service |
|---|---|---|
| Output | stdout (pretty or compact JSON) | JSONL files in `-out-dir` |
| Task system | none — one-shot generation | persistent tasks with scheduler |
| IFA | random per request | persistent device pool (`ip`/`bbox`) or fixed (`ifa`) |
| IP address | random | pinned to task's `ip_address` (`ip` criteria) |
| Geo | random or within `-bbox` | walking path persisted across ticks |
| `-bbox` format | `maxlat,maxlon,minlat,minlon` | GeoJSON `Polygon` in request body |
| `-mmdb` | parsed but unused | used to resolve IP coordinates at task creation |
| `ext` fields | `timestamp` only | `task_id`, `correlation_id`, `timestamp` |

## HTTP Service

Start the server:

```bash
./rtb-generator -server [flags]
```

### Server Flags

| Flag | Default | Description |
|---|---|---|
| `-server` | `false` | Run as HTTP server |
| `-port` | `8080` | Listening port |
| `-tasks-file` | `tasks.json` | Path to task persistence file |
| `-out-dir` | `output` | Directory for generated JSONL files |
| `-scheduler-interval` | `5m` | How often active tasks generate requests |
| `-mmdb` | | Path to MaxMind GeoIP2 City MMDB file (optional) |

```bash
./rtb-generator -server -port=8080 -scheduler-interval=1m -out-dir=./output -mmdb=./GeoLite2-City.mmdb
```

### MaxMind MMDB

When `-mmdb` is provided, IP-criteria tasks look up the IP address coordinates from the MMDB on the first scheduler tick. The resolved coordinates become the starting point (`last_geo`) for location generation.

If the MMDB is not configured, or the IP is not found (e.g. private/reserved addresses), the service falls back to a random city location.

Download a free database from [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data).

### API

#### Create task

```
POST /tasks
```

```json
{
  "correlation_id": "campaign-abc",
  "start_time": "2026-03-24T10:00:00Z",
  "end_time":   "2026-03-24T18:00:00Z",
  "criteria_type": "bbox",
  "geometry": {
    "type": "Polygon",
    "coordinates": [[
      [-74.1, 40.5],
      [-73.7, 40.5],
      [-73.7, 40.9],
      [-74.1, 40.9],
      [-74.1, 40.5]
    ]]
  },
  "count": 100
}
```

`criteria_type` must be one of:

| Value | Required field | Geo behaviour |
|---|---|---|
| `ip` | `ip_address` | Coordinates from MaxMind MMDB lookup; requests within 1 km radius |
| `ifa` | `ifa` | Random city anchor; requests within 1 km radius |
| `bbox` | `geometry` | GeoJSON `Polygon` (coordinates as `[longitude, latitude]`); requests randomly distributed inside the polygon |

Returns `201 Created` with the created task object.

**Example: IP criteria**

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_id": "campaign-ip-001",
    "start_time": "2026-03-24T10:00:00Z",
    "end_time":   "2026-03-24T18:00:00Z",
    "criteria_type": "ip",
    "ip_address": "8.8.8.8",
    "count": 50
  }'
```

**Example: IFA criteria**

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_id": "campaign-ifa-001",
    "start_time": "2026-03-24T10:00:00Z",
    "end_time":   "2026-03-24T18:00:00Z",
    "criteria_type": "ifa",
    "ifa": "38400000-8cf0-11bd-b23e-10b96e40000d",
    "count": 50
  }'
```

**Example: GeoJSON bounding box criteria**

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_id": "campaign-geo-001",
    "start_time": "2026-03-24T10:00:00Z",
    "end_time":   "2026-03-24T18:00:00Z",
    "criteria_type": "bbox",
    "geometry": {
      "type": "Polygon",
      "coordinates": [[
        [-74.1, 40.5],
        [-73.7, 40.5],
        [-73.7, 40.9],
        [-74.1, 40.9],
        [-74.1, 40.5]
      ]]
    },
    "count": 100
  }'
```

#### List tasks

```
GET /tasks
```

#### Get task

```
GET /tasks/{id}
```

#### Delete task

```
DELETE /tasks/{id}
```

Returns `204 No Content`.

### Task status

Task status is derived from the current time relative to the task's time window:

| Status | Condition |
|---|---|
| `pending` | `now < start_time` |
| `active` | `start_time ≤ now ≤ end_time` |
| `completed` | `now > end_time` |

### Scheduler

Every `-scheduler-interval`, the scheduler finds all `active` tasks and generates `count` RTB requests for each, writing them as newline-delimited JSON to:

```
{out-dir}/task_{correlation_id}_{unix_timestamp}.jsonl
```

Each request includes an `ext` object:

```json
{
  "ext": {
    "task_id": "...",
    "correlation_id": "...",
    "timestamp": 1742812345678
  }
}
```

`timestamp` is a Unix millisecond value randomized within `[now - scheduler-interval, now]`.

### Location walking

Every task type maintains a `last_geo` that is persisted in `tasks.json` and updated after each scheduler tick. This produces a continuous walking path across ticks and within each tick.

#### Initial location

| Criteria | Initial `last_geo` |
|---|---|
| `ip` | Resolved from MaxMind MMDB at task creation time |
| `ifa` | Random city, resolved at task creation time |
| `bbox` | Random point within the polygon, set on the first scheduler tick |

#### ip and bbox — per-device walking

`ip` and `bbox` tasks maintain a pool of `count` persistent devices, each with a unique IFA and its own location. The device map (`devices`) is persisted in `tasks.json`.

| Criteria | Device initialisation |
|---|---|
| `ip` | Devices spread within 1 km of the IP anchor (`last_geo`) on the first tick |
| `bbox` | Devices placed at random points within the polygon's bounding box on the first tick |

Each tick generates exactly `count` requests distributed across the device pool. A device may appear between 1 and `count/10` times per tick (randomised); slot assignments are shuffled so the same IFA does not appear consecutively. Within a tick, each consecutive appearance of the same device walks up to 1 km from its previous appearance. After the tick each device's final position is saved and becomes its starting point for the next tick.

For `bbox` tasks, if a walk step would leave the polygon's bounding box, the device resets to a fresh random point within the box.

#### ifa — single location walking

An `ifa` task generates `count` requests per tick from a single walking location (the task's fixed IFA). Within a tick, each consecutive request moves up to 2 km from the previous one. After the tick the final position is saved as `last_geo` and becomes the starting point for the next tick.

All location state survives server restarts.

### Example tick output

Each criteria type produces different patterns within a JSONL file. The examples below use `count=3` for brevity.

#### ip task

All requests share the task's IP. Devices (IFAs) are persistent across ticks and may appear more than once per tick. Geo walks ≤1 km between consecutive appearances of the same device.

```jsonl
{"id":"abc1","device":{"ip":"8.8.8.8","ifa":"3a1f2b4c-0001-0001-0001-000000000001","geo":{"lat":37.3861,"lon":-122.0839,"type":2}},"ext":{"correlation_id":"campaign-ip-001","task_id":"campaign-ip-001","timestamp":1742812341000}}
{"id":"abc2","device":{"ip":"8.8.8.8","ifa":"7e9d1a2f-0002-0002-0002-000000000002","geo":{"lat":37.3901,"lon":-122.0812,"type":2}},"ext":{"correlation_id":"campaign-ip-001","task_id":"campaign-ip-001","timestamp":1742812342000}}
{"id":"abc3","device":{"ip":"8.8.8.8","ifa":"3a1f2b4c-0001-0001-0001-000000000001","geo":{"lat":37.3865,"lon":-122.0831,"type":2}},"ext":{"correlation_id":"campaign-ip-001","task_id":"campaign-ip-001","timestamp":1742812343000}}
```

- `device.ip` is always the task IP address
- `device.ifa` repeats across ticks (persistent pool); may repeat within a tick (up to `count/10` times)
- `device.geo` walks ≤1 km between consecutive appearances of the same IFA

#### bbox task

Same per-device pool logic as `ip`. No IP field. Locations stay within the polygon's bounding box.

```jsonl
{"id":"def1","device":{"ifa":"c4d8e2a1-0003-0003-0003-000000000003","geo":{"lat":40.7214,"lon":-73.9872,"type":2}},"ext":{"correlation_id":"campaign-geo-001","task_id":"campaign-geo-001","timestamp":1742812341000}}
{"id":"def2","device":{"ifa":"9b3f1c7d-0004-0004-0004-000000000004","geo":{"lat":40.7531,"lon":-74.0124,"type":2}},"ext":{"correlation_id":"campaign-geo-001","task_id":"campaign-geo-001","timestamp":1742812342000}}
{"id":"def3","device":{"ifa":"c4d8e2a1-0003-0003-0003-000000000003","geo":{"lat":40.7221,"lon":-73.9908,"type":2}},"ext":{"correlation_id":"campaign-geo-001","task_id":"campaign-geo-001","timestamp":1742812343000}}
```

- No `device.ip`
- `device.ifa` is a persistent device UUID shared across ticks
- `device.geo` is always within the polygon's bounding box; resets to a fresh random point if a walk step would leave it

#### ifa task

All requests share the single fixed IFA from the task. Geo walks ≤2 km between consecutive requests within a tick.

```jsonl
{"id":"ghi1","device":{"ifa":"38400000-8cf0-11bd-b23e-10b96e40000d","geo":{"lat":51.5101,"lon":-0.1182,"type":2}},"ext":{"correlation_id":"campaign-ifa-001","task_id":"campaign-ifa-001","timestamp":1742812341000}}
{"id":"ghi2","device":{"ifa":"38400000-8cf0-11bd-b23e-10b96e40000d","geo":{"lat":51.5213,"lon":-0.1021,"type":2}},"ext":{"correlation_id":"campaign-ifa-001","task_id":"campaign-ifa-001","timestamp":1742812342000}}
{"id":"ghi3","device":{"ifa":"38400000-8cf0-11bd-b23e-10b96e40000d","geo":{"lat":51.5334,"lon":-0.0983,"type":2}},"ext":{"correlation_id":"campaign-ifa-001","task_id":"campaign-ifa-001","timestamp":1742812343000}}
```

- `device.ifa` is identical in every request (the value provided when creating the task)
- `device.geo` drifts ≤2 km per step; the final position in a tick becomes the starting point for the next tick

## Docker

```bash
docker compose up --build
```

Configuration is read from `.env`. Copy `.env.example` to `.env` and adjust:

```env
PORT=8081
TASKS_FILE=/app/data/tasks.json
OUT_DIR=/app/data/output
SCHEDULER_INTERVAL=5m
MMDB_PATH=/app/data/GeoLite2-City.mmdb
BBOX_MAX_LAT=
BBOX_MAX_LON=
BBOX_MIN_LAT=
BBOX_MIN_LON=
```

Place the MMDB file in `./data/` and set `MMDB_PATH` accordingly. If `MMDB_PATH` is empty, MMDB lookup is disabled and IP tasks fall back to a random city.

The `./data/` directory is mounted into the container at `/app/data` and holds tasks, output JSONL files, and optionally the MMDB file.

## Running tests

```bash
go test ./...
go test -cover ./...
```

MMDB integration tests require `data/GeoLite2-City.mmdb` to be present; they are skipped automatically when the file is absent.
