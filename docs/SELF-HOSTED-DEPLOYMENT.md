# Self-hosted deployment guide

This guide deploys the current Go API and PostgreSQL as Docker containers.
Cloudflare handles the public edge; it does not run the current Go server.

## Prerequisites

- A server with Docker Compose and capacity for PostgreSQL plus the API.
- A domain in Cloudflare.
- A Cloudflare Tunnel when the origin has no public inbound address. A Tailscale
  `100.x.x.x` address is private to the tailnet and cannot be used as a normal
  Cloudflare proxied origin.
- A GitHub Container Registry image published from `main` or a version tag.

## 1. Publish the image

Push to `main` and wait for **Publish container image** in GitHub Actions. It
publishes `ghcr.io/saddadnabbil/visionops:main` and an immutable `sha-...` tag.
For a release, create and push a tag such as `v0.1.0` and deploy that tag.

If GitHub marks the first GHCR package private, change its package visibility
to public in GitHub Packages, or authenticate the server with a
least-privilege token that has `read:packages` before running `docker compose
pull`. Never put that token in this repository.

## 2. Prepare the server

```bash
mkdir -p ~/apps/visionops/deploy
cd ~/apps/visionops
git clone https://github.com/saddadnabbil/visionops.git .
cp deploy/.env.server.example deploy/.env
chmod 600 deploy/.env
# Edit deploy/.env and replace every placeholder with a unique value.
docker compose --env-file deploy/.env -f deploy/compose.server.yml pull
docker compose --env-file deploy/.env -f deploy/compose.server.yml up -d
curl --fail http://127.0.0.1:18081/health
```

The Compose file binds the API only to `127.0.0.1:18081`; PostgreSQL is not
publicly exposed. Do not use the demo accounts or default secrets outside a
private demonstration.

## 3. Publish through Cloudflare Tunnel

Install `cloudflared` using Cloudflare's current official package instructions,
then create a named tunnel in the Cloudflare dashboard or CLI. Map the public
hostname to `http://127.0.0.1:18081` and run it as a system service. The
Cloudflare dashboard creates the DNS route for the hostname.

If the server instead has a public IP, use the Nginx example at
[`../deploy/nginx.visionops.conf.example`](../deploy/nginx.visionops.conf.example)
and configure a proxied DNS record with Cloudflare SSL/TLS mode **Full
(strict)**. Do not expose PostgreSQL or port `18081` publicly.

## 4. Verify

1. Open the HTTPS hostname and check the landing page and `/login`.
2. Sign in with a local demo role only in a private demo environment.
3. Confirm `https://your-host/health` returns `200`.
4. Confirm SSE remains connected after several minutes; the Nginx example
   disables proxy buffering for this reason.
5. Review container logs and database volume backup before inviting users.

## Update and rollback

```bash
cd ~/apps/visionops
git fetch --tags
# Set VISIONOPS_IMAGE in deploy/.env to an immutable sha-... or vX.Y.Z tag.
docker compose --env-file deploy/.env -f deploy/compose.server.yml pull
docker compose --env-file deploy/.env -f deploy/compose.server.yml up -d
curl --fail http://127.0.0.1:18081/health
```

Rollback means restoring the prior immutable image tag, not rebuilding a
moving `main` tag. Back up PostgreSQL before schema changes; application
rollback is safe only for backward-compatible migrations.
