# rtbgen

A Go CLI tool and HTTP service for generating random [OpenRTB 2.5](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf) bid requests. Useful for load testing ad exchanges, DSPs, and SSPs.

## Features

- Generates fully compliant OpenRTB 2.5 bid requests
- Supports banner, video, audio, and native impression types
- Optional geographic bounding box filtering
- Timestamps randomized within a configurable sliding window
- HTTP service for managing long-running generation tasks
- Persistent task storage (JSON file)
- Background scheduler generating JSONL output for active tasks

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

```bash
./rtb-generator -server -port=8080 -scheduler-interval=1m -out-dir=./output
```

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
  "bounding_box": {
    "max_lat": 40.9, "max_lon": -73.7,
    "min_lat": 40.5, "min_lon": -74.1
  },
  "count": 100
}
```

`criteria_type` must be one of:

| Value | Required field |
|---|---|
| `ip` | `ip_address` |
| `ifa` | `ifa` |
| `bbox` | `bounding_box` |

Returns `201 Created` with the created task object.

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

Every `-scheduler-interval`, the scheduler finds all `active` tasks and generates `count` RTB requests for each. Output is written as newline-delimited JSON to:

```
{out-dir}/task_{id}_{unix_timestamp}.jsonl
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

## Running tests

```bash
go test ./...
go test -cover ./...
```
