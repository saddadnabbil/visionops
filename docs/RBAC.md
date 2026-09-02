# VisionOps Access-Control Matrix

## Model

Every authenticated request carries a user, organization, role, and expiry. Resources are scoped to the caller's organization. A valid login never grants access to another organization.

| Capability | Admin | Operator | Supervisor | Viewer | Service API key |
| --- | :---: | :---: | :---: | :---: | :---: |
| View incidents, cameras, jobs, deliveries, metrics | Yes | Yes | Yes | Yes | No |
| View an incident detail/activity | Yes | Yes | Yes | Yes | No |
| Acknowledge or resolve an incident | Yes | Yes | No | No | No |
| Replay a dead job | Yes | Yes | No | No | No |
| Create/list users | Yes | No | No | No | No |
| Create/list ingest keys | Yes | No | No | No | No |
| Create/list webhook subscriptions | Yes | No | No | No | No |
| Create a camera | Yes | No | No | No | No |
| Toggle demo delivery failure | Yes | No | No | No | No |
| Ingest detection / heartbeat | No | No | No | No | Yes |

## Enforcement Rules

- Missing/invalid/expired bearer claim: `401 unauthorized`.
- Authenticated user outside the permitted role set: `403 forbidden`.
- Existing resource in another organization: respond as `404` where possible; never disclose its data.
- Invalid state transition (for example resolving an already resolved Incident): `409 invalid state or incident`.
- API keys are ingestion credentials, not human sessions; they are limited to registered Cameras in their organization.

## Role Landing Intent (Planned UI)

| Role | Default destination | Primary decision |
| --- | --- | --- |
| Admin | Organization / operations overview | Is configuration and delivery healthy? |
| Operator | Live incident queue | What needs action now? |
| Supervisor | Safety performance overview | Where is risk recurring and is response adequate? |
| Viewer | Read-only live overview | What is happening now? |

## Access Test Cases

1. A Viewer receives `403` for acknowledge, resolve, replay, user, key, webhook, and failure-mode mutations.
2. An Operator can acknowledge/resolve and replay a dead job but cannot configure the organization.
3. A Supervisor can read planned detail screens but cannot mutate any operational record.
4. An Admin can create a user/key/webhook/camera only inside their organization.
5. A service key for organization A cannot ingest an event or heartbeat for a camera from organization B.
