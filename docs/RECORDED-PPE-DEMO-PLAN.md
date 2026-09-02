# Recorded PPE Demo plan

## Decision

Use one licensed industrial clip in two honest demo states. The worker begins
without a hard hat and then puts one on, while already wearing a high-vis vest.
This provides a consistent `missing_hard_hat` and `ppe_compliant` presentation
without using public CCTV, filming a real workplace, or claiming that a model
has inferred a result when it has not.

The downloaded master and licence record are listed in
[`LICENSE-SOURCES.md`](../LICENSE-SOURCES.md). The research decision and
rejected personal-use-only candidates are in
[`FOOTAGE-RESEARCH.md`](FOOTAGE-RESEARCH.md).

## What "AI" means in each phase

| Phase | Input decision maker | What VisionOps may claim |
| --- | --- | --- |
| Recorded scenario (next) | A reviewed timestamp manifest | `Recorded scenario`; no vision inference occurred. |
| Model pilot (future) | A versioned PPE detector plus rule evaluator | `Model-generated detection`, with model/version/confidence retained. |
| Production integration (future) | Approved detector on an authorised stream | Same as model pilot, plus source/retention controls and monitoring. |

The existing Go API, incident correlation, RBAC, audit events, durable webhook,
and live dashboard are real product behaviour in every phase. The *detector*
is the part that changes.

## Recorded-scenario implementation plan

### Phase A — curate and prepare

- [x] Obtain a licensed master clip and record provenance.
- [x] Establish two visual states: before the hard hat is on, then after it is
  on. The high-vis vest is present in both states.
- [x] Create local derived previews,
  `missing-hard-hat-recorded-scenario.mp4` and
  `hard-hat-compliant-recorded-scenario.mp4`; leave the master intact. Do not
  commit footage to a public repository without a release decision.
- [ ] Add a poster image that avoids presenting the person's face as an
  identity. The application must never name or recognize this person.

### Phase B — scenario contract

Add a manifest beside the existing adapter fixture. It is explicit about being
scripted:

```json
{
  "mode": "recorded_scenario",
  "source_asset": "mixkit-worker-hard-hat-transition-1440-720.mp4",
  "source_url": "https://mixkit.co/free-stock-video/man-puts-on-hard-hat-1440/",
  "events": [
    {
      "after_seconds": 2,
      "rule": "missing_ppe",
      "ppe_item": "hard_hat",
      "severity": "high",
      "disclosure": "SIMULATED DETECTOR / RECORDED SCENARIO"
    },
    {
      "after_seconds": 7,
      "rule": "ppe_compliant",
      "disclosure": "RECORDED SCENARIO"
    }
  ]
}
```

`ppe_compliant` should be a UI-only state initially, not an ingest rule that
creates an incident. The missing-hard-hat event continues to use the existing
validated `missing_ppe` ingestion rule, so all incident behaviour stays real.

### Phase C — adapter and API (implemented)

1. Extend `camera-adapter` with an opt-in `recorded_scenario` mode. It loads the
   manifest, discovers the registered Camera, maintains the existing heartbeat,
   and emits only the validated `missing_ppe` Detection at its declared time.
2. Add provenance to safe Detection metadata: `source_mode`, `scenario_id`,
   `asset_id`, and `disclosure`. Do not send face data, raw frames, or video
   URLs through the ingestion API.
3. Preserve idempotency: producer event IDs are deterministic per scenario run
   but unique per new run.
4. Keep the current fixture mode as the fast, video-free test path.

### Phase D — dashboard (implemented)

1. Add a `Recorded Demo` camera card and a local HTML5 video preview.
   Use the wide construction-site clip for an anonymous work-area view; use the
   hard-hat transition clip only where the demo needs a clearly visible PPE
   state change.
2. Show a persistent mono label: `RECORDED SCENARIO — NOT LIVE CCTV`.
3. Overlay the current state with text, not colour alone: `Missing hard hat`
   before correction and `Hard hat present` afterwards.
4. When the adapter emits an event, link the preview state to the real incident
   queue and its immutable audit timeline.
5. Keep face identification out of scope. The incident identifies a camera and
   anonymous track/reference only; an Operator may record a verified worker
   name manually only if a real organisation's policy permits it.

### Phase E — quality gate (verified for the recorded path)

- Unit tests: manifest parsing, required disclosure, allowed PPE item, event
  scheduling, and idempotency.
- Integration: recorded adapter can discover only its tenant Camera and creates
  one missing-PPE incident plus outbox delivery.
- Browser E2E: video disclosure visible, event reaches queue, Operator
  acknowledges/resolves, Viewer cannot mutate.
- Responsive review: 1440px, 768px, and 390px; video never autoplays with
  sound, controls are keyboard accessible, and the scenario label remains
  visible.

Verification on 2026-09-02: `go test ./...` passed; the Docker
`recorded-demo` profile discovered the tenant Camera, accepted its heartbeat,
and accepted the `recorded_scenario` missing-PPE event; Playwright passed 12
desktop/mobile checks, including the visible disclosure and video source.

## Future real-AI boundary — intentionally not implemented yet

```text
authorised recorded video or RTSP/HLS stream
  -> frame sampler
  -> PPE inference service (versioned model)
  -> policy evaluator: hard-hat / vest required for this camera zone
  -> Go camera-adapter validation client
  -> existing VisionOps ingest API
```

Entry requirements: written stream-owner permission, documented zone PPE
policy, a held-out labelled evaluation set, measured precision/recall per PPE
item, false-positive review flow, retention/deletion policy, and visible model
version/confidence in the incident. A detector model is not added merely to
make the product sound like AI.

## Release walkthrough

1. Start VisionOps with the recorded-demo profile.
2. Sign in as Operator and open the `Recorded Demo` camera.
3. Show the persistent recorded-scenario disclosure and the pre-hard-hat state.
4. Let the adapter create the real missing-PPE Incident.
5. Acknowledge and resolve it with an operator note.
6. Show the subsequent compliant visual state; explain that it indicates
   correction in the scripted demo and does not auto-close the incident.
