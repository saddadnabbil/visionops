# VisionOps

VisionOps turns AI-camera safety detections into actionable workplace incidents. It is a backend-first Go service focused on dependable asynchronous delivery, not computer-vision model training.

## Run it

```bash
docker compose up --build
open http://localhost:18080
```

The root URL is the VisionOps landing page. Choose **Enter workspace** to open
`/login`, then use one of the local accounts below.

Use the sign-in form with one of the local workspace accounts below. The account shortcut buttons only fill credentials; they do not impersonate a role. Click **Simulate detection** as Admin or Operator to run the complete flow. The worker writes its signed webhook delivery to the API logs.

| Role | Email | Password | Default workspace |
| --- | --- | --- | --- |
| Admin | `admin@acme.test` | `demo-password` | Operations overview and organization configuration |
| Operator | `operator@acme.test` | `demo-password` | Live incident queue |
| Supervisor | `supervisor@acme.test` | `demo-password` | Safety analytics |
| Viewer | `viewer@acme.test` | `demo-password` | Read-only operations overview |

## Architecture

```
AI simulator -> detection ingest -> PostgreSQL transaction
                                    |-> incident correlation + audit activity
                                    |-> durable outbox job -> Go worker -> signed webhook
                                                           -> SSE -> operations dashboard
```

## Camera-adapter demo

Run the separately deployed, fixture-based producer with:

```bash
docker compose --profile camera-demo up --build
```

It performs API-key Camera discovery, heartbeat, and one synthetic missing-PPE Detection. It does not access public CCTV or video. See [Camera Adapter Pilot](docs/CAMERA-ADAPTER.md) for configuration and the approved path to a future RTSP/detector integration.

## Recorded PPE demo

Run the recorded scenario with:

```bash
docker compose --profile recorded-demo up --build
```

Open **Cameras** after signing in. The worksite video is licensed recorded
footage with a persistent `RECORDED SCENARIO / SIMULATED DETECTOR` disclosure;
it is not live CCTV and no model evaluates the video. After two seconds, the
adapter submits a real `missing_ppe` event through the normal ingestion path.
Use an Operator account to acknowledge and resolve the resulting incident.
Asset provenance is in [LICENSE-SOURCES.md](LICENSE-SOURCES.md).

## API flow

The seeded camera ID is returned from `GET /api/v1/cameras` after logging in. The local API is published on port `18080`. Send a detection with:

```bash
bash scripts/simulate.sh CAMERA_ID missing_ppe
```

`POST /api/v1/ingest/detections` is authenticated with `X-API-Key`, uses `event_id` as the idempotency key, and correlates repeated camera/rule events during a five-minute window. The API supports `missing_ppe`, `restricted_area`, and `crowding` rules.

Detection bodies are capped at 64 KiB. `GET /api/v1/incidents?limit=50` supports a bounded limit from 1 to 100.
Ingestion is capped at 60 requests per API key per minute in this single-instance demo; deploy a shared rate-limit store before horizontally scaling the API.

The compact OpenAPI contract is available at [`web/openapi.yaml`](web/openapi.yaml)
and at `/openapi.yaml` while the app is running locally. Failed jobs can be
inspected through `GET /api/v1/deliveries` and replayed with
`POST /api/v1/jobs/{id}/replay` after reaching the dead-letter state.

An administrator can manage organization users through `/api/v1/users` and create ingestion keys through `/api/v1/api-keys`. A new key is returned only on creation; VisionOps stores only its SHA-256 hash.

## Reliability behavior

- A detection, audit event, and outbox job commit atomically.
- An ingestion API key can submit detections only for Cameras registered to its own organization.
- Workers claim jobs using `FOR UPDATE SKIP LOCKED`; more workers can run safely.
- Failed delivery retries with exponential backoff and becomes `dead` after five attempts.
- Each attempt is recorded in `webhook_deliveries`; alert payloads are signed with HMAC-SHA256 and timestamped.

## Checks

```bash
go test ./...
go test -race ./...
go test -tags=integration ./internal/visionops
npm ci
npx playwright install chromium
npm run test:e2e
```

The integration test requires the local PostgreSQL container from `docker compose up -d` and verifies cross-tenant camera rejection plus successful durable outbox delivery.

This demo intentionally consumes detector output only. It neither processes video streams nor makes a safety-certification claim.

## Release

See [RELEASE.md](RELEASE.md) for the clean-run release gate, public-demo
boundaries, and an operational walkthrough. GitHub Actions runs unit, race,
integration, browser E2E, container-build, and health checks on every push and
pull request.

See [DESIGN.md](DESIGN.md) for the local design-system and responsive implementation reference.

## Product delivery documentation

- [Product scope](docs/PRODUCT.md)
- [Access-control matrix](docs/RBAC.md)
- [Data and API contract](docs/DATA-API-CONTRACT.md)
- [Test strategy](docs/TEST-STRATEGY.md)
- [Operations and release plan](docs/OPERATIONS.md)
- [Camera adapter pilot](docs/CAMERA-ADAPTER.md)
- [Recorded-demo asset policy](docs/DEMO-ASSETS.md)
