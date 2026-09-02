# Cloudflare Deployment Assessment

**Status:** recommendation for the current VisionOps architecture
**Reviewed:** 2 September 2026
**Scope:** the existing Dockerized Go HTTP API, PostgreSQL database, and browser UI.

## Decision

For a first public deployment, keep the Go API and PostgreSQL as conventional
container/database services, then put Cloudflare in front as the DNS, TLS,
WAF, and CDN layer. This is the lowest-risk path because it retains the
application's current runtime and PostgreSQL driver unchanged.

Cloudflare Containers is a viable **second** deployment option if the goal is
to run the Go API on Cloudflare itself. It is not a direct `docker compose up`:
the API container is invoked through a small Worker and a Durable Object, so it
needs a deployment adapter and production operations testing before it becomes
the default path.

## Options

| Option | Fit for VisionOps today | What changes | Recommendation |
| --- | --- | --- | --- |
| Go container on a conventional host + Cloudflare proxy | Excellent | Keep Dockerfile/API; use a managed PostgreSQL service; orange-cloud the web hostname | **Start here** |
| Cloudflare Containers + Worker | Good, but more platform work | Add a JS/TS Worker router, Container Durable Object configuration, Wrangler config, and external managed PostgreSQL | Evaluate after the public demo is stable |
| Workers with Go compiled to Wasm | Poor for this app | Rewrite the HTTP/runtime integration around the Workers APIs and validate Wasm compatibility | Do not use for the first release |
| Cloudflare Pages | Static UI only | Pages Functions run on the Workers runtime; they cannot host the existing Go server binary | Optional only after separating the frontend from the API |

## Why Containers, not a plain Worker, for Go

Cloudflare Containers supports existing container images and includes an
official Go server example on port 8080. The image must run on `linux/amd64`.
Containers are available on the Workers Paid plan and are controlled by Worker
code; a Worker receives the request and routes it to a Container-backed Durable
Object. [Containers overview](https://developers.cloudflare.com/containers/)
and [Go container quickstart](https://developers.cloudflare.com/containers/get-started/)
document this architecture.

Workers can load Go only after compiling it to WebAssembly; Go is not a
first-class Workers server runtime. Cloudflare notes that language/compiler
support varies, threading is unavailable, and WASI support is experimental
with only some syscalls implemented. That is a material mismatch for the
existing `net/http` + PostgreSQL application. [Workers languages](https://developers.cloudflare.com/workers/languages/),
[WebAssembly runtime](https://developers.cloudflare.com/workers/runtime-apis/webassembly/).

## Database and state

Do not run production PostgreSQL inside the application container. Use a
managed, regional PostgreSQL provider, take backups, and store its connection
string as a deployment secret. Cloudflare Hyperdrive can accelerate a
PostgreSQL connection **from a Worker**, but it connects to an existing
database; it is not a PostgreSQL host. [Hyperdrive overview](https://developers.cloudflare.com/hyperdrive/).

For the conventional-host route, Cloudflare's proxied A/AAAA/CNAME records put
Cloudflare between visitors and the origin, enabling TLS, caching, WAF rules,
and DDoS protections while preserving the Go service behind it. Restrict origin
ingress to Cloudflare where the hosting provider permits it. [Cloudflare proxy
status](https://developers.cloudflare.com/dns/proxy-status/).

## CI/CD implications

- **CI remains GitHub Actions:** test Go, run Playwright, build the Docker
  image, and scan for committed secrets on every pull request.
- **Conventional-host CD:** deploy an immutable image to a staging environment,
  run a health check and smoke test, then promote manually to production. Keep
  Cloudflare API credentials only in GitHub environment secrets.
- **Containers CD:** use `wrangler deploy` from the production environment.
  Cloudflare's own guidance says that non-production Workers Builds normally
  upload Worker code only—not a new container image or full preview—so staging
  needs its own Wrangler environment/Worker and a full deploy command.
  [Deploy Containers](https://developers.cloudflare.com/containers/guides/deploy/).

## Preconditions before choosing Cloudflare Containers

1. Confirm the account is on Workers Paid and that Containers is available.
2. Build the present image for `linux/amd64` and measure image size/startup.
3. Create separate staging and production Worker environments; do not point a
   feature branch deployment at the production Worker.
4. Provision managed PostgreSQL, TLS-only connectivity, backups, and a secret
   rotation path.
5. Add deployment health checks for `/health`, database migrations, logs, and
   a rollback to the prior image version.

## Suggested release sequence

1. Publish the clean GitHub repository with CI and no real secrets.
2. Deploy staging using a conventional Go container host plus Cloudflare proxy.
3. Validate login, incident workflow, SSE, migrations, and a recorded-camera
   simulation in the public environment.
4. Decide whether the operational value of Cloudflare Containers justifies the
   Worker/Durable Object adapter. If yes, build it as a separately tested
   deployment target rather than replacing the proven staging path in place.
