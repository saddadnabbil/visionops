# Camera Adapter Pilot

## Purpose

`camera-adapter` is a separate Go process that behaves like a future approved AI Camera Service. It discovers an eligible Camera through its ingestion API key, posts heartbeat messages, then sends a legally safe fixture Detection. It has no video, RTSP, scraping, image storage, or model-inference capability.

This is the recommended next integration phase because it validates VisionOps's true external boundary before any camera-source agreement exists.

## Run the Safe Demo

```bash
docker compose --profile camera-demo up --build
```

The adapter waits five seconds, posts a synthetic `missing_ppe` Detection for `Line A Entrance`, and continues sending a heartbeat every 15 seconds. Open the app and sign in as an Operator to acknowledge and resolve it.

Stop it with:

```bash
docker compose --profile camera-demo down
```

## Producer Contract

1. `GET /api/v1/ingest/cameras` with `X-API-Key` returns only Cameras owned by that key's organization.
2. `POST /api/v1/ingest/camera-heartbeats` establishes connectivity health for a selected Camera.
3. `POST /api/v1/ingest/detections` submits the idempotent Detection. The adapter creates a new producer event ID for each fixture run.

The adapter never receives human user credentials and never accesses VisionOps PostgreSQL directly.

## Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `VISIONOPS_API_URL` | `http://localhost:18080` | VisionOps API origin. |
| `VISIONOPS_API_KEY` | required | Tenant-scoped producer key. |
| `CAMERA_ADAPTER_CAMERA_NAME` | first available | Registered Camera to use. |
| `CAMERA_ADAPTER_SCENARIO` | fixture path | JSON scenario of synthetic rule events. |
| `CAMERA_ADAPTER_HEARTBEAT_INTERVAL` | `30s` | Connectivity heartbeat interval. |

Scenario events contain `after_seconds`, `rule`, `severity`, and optional safe metadata. Supported rules and severities are validated before any HTTP request.

## Replacing the Fixture Later

Keep this adapter boundary and replace only the source that creates a validated `Detection`:

```text
authorized RTSP/HLS or recorded video
  -> frame sampler
  -> approved detector/rule evaluator
  -> validated Detection
  -> existing Camera Adapter client
  -> VisionOps ingestion API
```

Before doing so, obtain stream-owner permission, a written purpose/retention policy, and a data-processing review. Do not scrape public CCTV portals, attempt to discover undocumented stream URLs, store public feeds, or use identifiable footage in the public demo.

## Recorded PPE Demo

`docker compose --profile recorded-demo up --build` starts a second opt-in
adapter profile using `fixtures/camera-adapter/recorded-ppe-demo.json`. The
manifest is not a detector: it supplies a reviewed timestamp and source
provenance, and the adapter still sends only the standard validated Detection.

The browser shows the accompanying worksite clip from `/assets/demo/` with the
mandatory disclosure `RECORDED SCENARIO / SIMULATED DETECTOR — NOT LIVE CCTV`.
No frame, face, crop, direct media/stream URL, or biometric identifier is sent to VisionOps.
Only `source_mode`, `source_asset`, `source_url`, `disclosure`, and safe
scenario metadata are attached to the Detection. See
[Recorded PPE Demo plan](RECORDED-PPE-DEMO-PLAN.md) and
[asset sources](../LICENSE-SOURCES.md).
