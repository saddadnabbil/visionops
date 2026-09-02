# VisionOps Operations and Release Plan

## Service Objectives for the Next Production Phase

These are proposed targets, not claims about the demo: ingest acceptance p95 under 300ms, successful outbox delivery within 60 seconds under normal dependency health, and a visible camera-health transition no later than five minutes after a missed heartbeat interval.

## Operational Signals

| Signal | Investigate when | First action |
| --- | --- | --- |
| Health endpoint | non-200 | Inspect API/database reachability; stop release if persistent. |
| Open critical Incidents | count > 0 | Safety Operator triages via live queue and follows local safety procedure. |
| Dead jobs | count > 0 | Admin/Operator checks target endpoint and replays only after recovery. |
| Retry volume | sustained increase | Inspect delivery error and subscription destination. |
| Degraded/offline Camera | unexpected | Verify camera service/network; heartbeat alone does not prove video quality. |
| SSE disconnected | persistent | UI continues polling; inspect API connection/proxy configuration. |

## Incident Runbooks

### Webhook delivery failed

1. Inspect the job and its delivery errors.
2. Confirm the destination URL, signature secret, and receiver health.
3. Fix the destination; do not repeatedly replay while it is failing.
4. Replay the dead job and confirm a `done` state and a recorded successful delivery.
5. Record the outcome in the external operational process where required.

### Camera is degraded or offline

1. Confirm the camera belongs to the expected location and read its last detection/heartbeat status.
2. Check the external AI Camera Service and network path.
3. Restore the producer, then send a heartbeat; VisionOps will derive camera health.
4. Treat restored heartbeat as connectivity recovery only. Validate camera framing/model behavior outside VisionOps.

### Fixture Camera Adapter is unavailable

1. Inspect `docker compose logs camera-adapter` when running the `camera-demo` profile.
2. Confirm its API key, Camera name, and scenario file are configured.
3. The adapter retries Camera discovery while VisionOps starts; a persistent error means it cannot see a tenant-owned Camera with that key.
4. This is a demo producer only. Do not replace its fixture with a public CCTV stream; follow `CAMERA-ADAPTER.md` before attaching an authorized source.

### Ingest rejected

1. `400`: validate event fields and tenant-owned registered camera.
2. `401`: rotate/use a valid active API key.
3. `429`: producer backs off; the demo limit is 60 requests/key/minute and is local-process only.
4. `5xx`: retry the same `event_id` with bounded backoff; idempotency makes retry safe.

## Security Baseline Before Any Non-Demo Deployment

- Replace demo secret and credentials; use a managed secret store.
- Replace custom demo tokens with reviewed standards-based OIDC/session handling.
- Enforce TLS, secure headers, trusted webhook URL controls, and rate limiting shared across replicas.
- Add API-key expiry/revocation, key rotation, audit export, backup/restore testing, and access reviews.
- Complete privacy review before retaining identity, media, or location data beyond the minimum operational need.

## Release Checklist

1. Run all automated gates in `TEST-STRATEGY.md`.
2. Execute the normal event and failed-delivery/replay scenario in `RELEASE.md`.
3. Verify Admin, Operator, Supervisor, and Viewer boundaries.
4. Review 1440px, 768px, and 390px layouts against `DESIGN.md` for any UI release.
5. Apply migrations with backup/rollback procedure validated in a non-production environment.
6. Publish release notes with known limitations and rollback owner.

## Rollback Principle

Application code may roll back independently when its database migrations are backward compatible. Destructive schema changes require an expand/migrate/contract sequence and a tested restore path; do not couple them to a single irreversible deployment.
