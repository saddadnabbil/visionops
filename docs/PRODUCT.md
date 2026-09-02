# VisionOps Product Scope

## Product Statement

VisionOps turns machine-produced workplace-safety detections into accountable Incidents that a safety team can triage, resolve, audit, and deliver to approved systems. It is a safety-operations tool, not an AI-camera product and not a safety-certification system.

## Users and Outcomes

| User | Primary outcome |
| --- | --- |
| Safety Operator | Finds urgent work, acknowledges ownership, records a resolution. |
| Safety Supervisor | Understands risk patterns and response performance without changing live work. |
| Administrator | Configures people, cameras, ingest credentials, and notification destinations. |
| Viewer | Has trustworthy, read-only operational visibility. |
| AI Camera Service | Reliably submits an idempotent Detection for a registered Camera. |

## MVP Boundary: Implemented

- Tenant-scoped users and four roles.
- Password login and signed bearer claims for the local demo.
- API-key ingestion of Detection events, 64 KiB body limit, and per-key local rate limiting.
- Five-minute correlation from Detections to Incidents.
- Incident list/detail, acknowledgement, resolution note, and audit activity.
- Camera registry, heartbeat-derived health, operations metrics, SSE updates.
- Transactional outbox, signed webhook delivery, retries, dead-letter, and replay.
- Seeded synthetic data and failure simulation for local scenario verification.
- Responsive role-specific application shell: actual email/password login, session profile, role landing, navigation, read-only or mutable incident detail, and Admin configuration screens.

## Explicitly Outside the MVP

- Video storage, streaming, recording playback, face recognition, or training a vision model.
- SMS/email/push delivery provider; the demo uses a webhook receiver.
- SSO, password reset, MFA, SCIM, and production session management.
- Escalation policy engine, assignments, comments, attachments, and legal hold.
- Native mobile apps, offline mode, and safety compliance certification.

## Future Scope, in Dependency Order

1. Incident assignment, severity policy, escalation timer, and supervisor review queue.
2. Camera lifecycle management, configuration validation, and media-evidence references.
3. External notification connectors with per-subscription delivery policy.
4. Production identity (OIDC/SAML/MFA), key rotation/revocation, and audit export.

## Product Rules

- A Detection is evidence. An Incident is the human-operable case.
- An Incident is only correlated with open or acknowledged Incidents for the same organization, camera, and safety rule within five minutes.
- Resolution is terminal in the current model. A later Detection opens a new Incident rather than reopening a resolved one.
- Every mutable resource is organization scoped. Cross-tenant access is forbidden.
- A successful ingest response means persistence and outbox creation completed; it does not mean a webhook was received.

## Definition of Done for a New Capability

1. It has an owner role and denied-role behavior.
2. Its loading, empty, failure, and success states are designed in `DESIGN.md` terms.
3. Its API/data contract and audit behavior are documented.
4. It has automated tests at the appropriate boundary.
5. Its operational failure mode and rollback/recovery action are in `OPERATIONS.md`.
