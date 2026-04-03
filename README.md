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

### Geo anchor persistence

For `ip` and `ifa` tasks, `last_geo` is persisted in `tasks.json` after every scheduler tick. It holds the last generated location of that tick and becomes the starting anchor for the next tick.

On the first tick the initial anchor is resolved as follows:
- **ip** — coordinates from the MaxMind MMDB lookup for the given IP address
- **ifa** — a random city

This produces a continuous walking path: each tick generates locations within 1 km of where the previous tick ended, and any two consecutive locations across ticks are always within 2 km of each other. `last_geo` survives server restarts.

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
