# Deployment runbook

This runbook is for the current Dockerized demo. It is not a claim that the
application is production-ready or safety-certified.

## Local release check

```bash
go vet ./...
go test ./...
go test -race ./...
npm ci
npx playwright install chromium
docker compose --profile camera-demo up --build -d
curl --fail http://localhost:18080/health
npm run test:e2e
docker compose down -v
```

The `camera-demo` profile is a fixture producer. For the recorded scenario,
use `docker compose --profile recorded-demo up --build`; see
[`../DEMO-ASSETS.md`](../DEMO-ASSETS.md) before adding any local footage.

## Environment

Copy `.env.example` to `.env` for local changes. Do not commit `.env`.

| Variable | Required in production | Purpose |
| --- | --- | --- |
| `APP_ENV` | Yes | Set to `production` outside local development. |
| `DATABASE_URL` | Yes | PostgreSQL connection string. Use managed TLS in production. |
| `JWT_SECRET` | Yes | Signs demo bearer tokens. Must be unique and at least 32 characters when `APP_ENV=production`. |
| `INGEST_API_KEY` | Yes | Bootstrap key for the fixture adapter. Must be unique and at least 20 characters when `APP_ENV=production`. |
| `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` | Compose only | Local database bootstrap values. |

The API refuses to start in `APP_ENV=production` with its known demo values.
Use the deployment platform's secret manager; never put a production value in a
workflow file, image, or GitHub repository.

## Deployment sequence

1. Build an immutable image from a tagged Git commit.
2. Apply backward-compatible migrations with a tested database backup.
3. Inject production secrets through the target platform.
4. Start the API and verify `GET /health` returns `200`.
5. Run the role-based smoke flow and verify one webhook delivery.
6. Monitor errors, dead jobs, and camera-heartbeat health before widening use.

## Rollback

Roll back the API image only when its migrations are backward compatible. For
schema changes, use expand/migrate/contract and test restore separately. Do
not replay failed webhooks until their receiver has recovered.
