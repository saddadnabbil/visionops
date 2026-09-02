# VisionOps Release Guide

## Release Gate

Run from a clean checkout:

```bash
go test ./...
go test -race ./...
docker compose --profile camera-demo up --build -d
curl --fail http://localhost:18080/health
npm ci
npx playwright install chromium
npm run test:e2e
```

Open `http://localhost:18080`. Sign in using the documented demo account for the role being demonstrated. The `camera-demo` profile delivers a safe fixture Detection after startup; verify it as an Operator. Then turn on `Simulate webhook failure` as Admin in Delivery Operations, create another detection, observe retry/dead-letter, turn the failure mode off, and replay the dead job as Admin or Operator.

Stop the local release environment with:

```bash
docker compose down -v
```

## Public Demo Boundary

- The public demo uses seeded synthetic data and the simulator only.
- Do not use real video, real employee identity, or personal data.
- The deployment is not a safety-certified system.
- Set a strong `JWT_SECRET` and remove demo credentials before exposing a non-demo environment.

## Demo Roles

All demo accounts use `demo-password`.

| Role | Email | Expected capability |
| --- | --- | --- |
| Admin | `admin@acme.test` | Full configuration and incident actions. |
| Operator | `operator@acme.test` | Acknowledge/resolve incidents and replay dead jobs. |
| Supervisor | `supervisor@acme.test` | Read-only incident, camera, delivery, and metrics review. |
| Viewer | `viewer@acme.test` | Read-only operational visibility. |

## Demo Walkthrough

1. Show the live incident dashboard and create a missing-PPE event.
2. Open its detail panel, acknowledge, add a resolution note, and resolve it.
3. Show camera status/heartbeat and supervisor metrics.
4. Enable failure simulation; show retry, dead-letter, and successful replay.
5. End on `/openapi.yaml` and the reliability architecture in `README.md`.
