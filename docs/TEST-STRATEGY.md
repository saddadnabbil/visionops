# VisionOps Test Strategy

## Quality Pyramid

| Layer | Purpose | Current / target examples |
| --- | --- | --- |
| Unit | Fast deterministic business and boundary checks | claim signing/expiry, role gate, state transition helpers |
| Integration | Real PostgreSQL transaction and worker behavior | cross-tenant rejection, durable outbox delivery |
| Contract | Prevent producer/client drift | OpenAPI examples, detection schema, webhook signature payload |
| E2E | Role journey in a running app | Implemented Playwright desktop/mobile suite for each role landing, Operator acknowledge/resolve, Viewer read-only detail, and Admin configuration/create flows |
| Non-functional | Protect reliability and usability | race, load, a11y, responsive, recovery drills |

## Required Automated Gates

```bash
go test ./...
go test -race ./...
go test -tags=integration ./internal/visionops
docker compose up --build -d
curl --fail http://localhost:18080/health
npm ci
npx playwright install chromium
npm run test:e2e
```

CI must fail on any gate. Integration tests require the compose PostgreSQL service.

## Scenario Matrix

| Scenario | Expected result |
| --- | --- |
| Same event submitted twice | first `202`; subsequent request `200 duplicate`; one Detection/Incident effect only. |
| Repeated same camera/rule inside five minutes | one Incident with increased occurrence count. |
| Camera belongs to another tenant | ingest rejected; no tenant information leaks. |
| Operator resolves Incident | status changes once; resolution activity has actor/note. |
| Viewer attempts mutation | `403`; no record changes. |
| Webhook endpoint fails | job retries, records deliveries, ends in `dead` after five attempts. |
| Dead job replayed | returns to pending, later finishes if endpoint recovers. |
| SSE disconnect | UI identifies paused updates and falls back to periodic refresh. |
| Mobile role screen | no horizontal data table; all actions reach 44px target. |
| Fixture Camera Adapter | API-key discovery returns tenant-only Camera; adapter heartbeat and synthetic Detection are accepted. |

## Future Test Work

- Add table-driven tests for every incident state and role/resource combination.
- Validate request schemas and expand OpenAPI response definitions.
- Add visual baselines and keyboard-only browser E2E journeys for each role.
- Add accessibility assertions: landmark names, focus order, dialog trap, live notifications, contrast.
- Load test ingestion and concurrent worker claims; set a measured throughput SLO before scale-out.

## Test Data Rules

- Tests and public demos use synthetic locations/events only.
- Never commit raw API keys, real credentials, camera streams, images, or employee data.
- A test that changes webhook failure mode must reset it in cleanup.
