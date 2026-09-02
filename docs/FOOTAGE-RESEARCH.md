# Licensed industrial footage research

Research date: 2026-09-02. This note identifies **display footage** for the
VisionOps recorded scenario. It is not training data and it must not be presented
as evidence produced by a real PPE model.

## Recommended pair

### 1. PPE-compliant baseline

- **Candidate:** [Pexels — Engineer and a worker looking at the
  camera](https://www.pexels.com/video/engineer-and-a-worker-looking-at-the-camera-8964295/)
  (Mikael Blomkvist).
- **Why:** The publisher's page labels it free to use and explicitly describes
  two construction workers wearing hard hats and safety vests. This is a clear
  baseline clip for a `ppe_compliant` / all-clear camera state.
- **License:** Pexels says photos and videos may be downloaded and used for
  free, including online/app use and commercial use; attribution is not
  required. It also permits modification. See the official
  [Pexels License](https://www.pexels.com/license/).

### 2. No-hard-hat scenario: do not source a public reusable clip yet

- **Finding:** No candidate was found whose first-party page both clearly
  establishes *no hard hat* and gives a reuse licence suitable for a public
  product use. A seemingly relevant [Mixkit construction-worker
  clip](https://mixkit.co/free-stock-video/construction-man-working-on-a-hot-day-47218/)
  is Restricted License / personal use only, so it is rejected for this use.
- **Decision:** Use the licensed compliant Pexels footage as recorded camera
  playback, then trigger a clearly labelled `RECORDED SCENARIO` event such as
  `missing_hard_hat`. It demonstrates the incident workflow without falsely
  claiming that the video or an AI model established a violation.
- **Future replacement condition:** Replace the scripted event only after an
  authorised clip is manually reviewed and its individual licence is recorded.
  Do not use a clip merely because its search tags omit “hard hat”.

## Alternate sources and exclusions

- [Mixkit construction collection](https://mixkit.co/free-stock-video/construction/)
  has free construction footage, including workers in hard hats/high-vis
  clothing. Every individual item must be checked: Mixkit has both a **Free
  License** (commercial use) and a **Restricted License** (personal use only).
  The official [Mixkit video page](https://mixkit.co/free-stock-video/) states
  both conditions and that downloads have no watermark.
- Do **not** use [Mixkit's industrial-corridor worker
  clip](https://mixkit.co/free-stock-video/worker-walking-in-industrial-machine-corridor-23378/)
  for public product use: its item page explicitly identifies the free 720p
  download as Restricted License / personal use only.
- Do not use arbitrary YouTube uploads. A YouTube video is out of scope unless
  its creator separately provides a direct, unambiguous reuse licence and a
  permitted download path.

## Download and retention procedure

1. Use the provider's **browser download** button, then retain the original
   filename unmodified under a non-public local demo-assets directory.
2. At download time record the item URL, creator, download date, selected
   resolution and a screenshot/PDF of the licence page in `LICENSE-SOURCES.md`.
3. Do not commit footage to the public repository unless the repository's
   release policy explicitly allows it. Distribute a short setup instruction
   instead.
4. No confident automated download method is recommended. The source pages
   expose an interactive free-download control; Pexels' official guidance also
   prohibits unauthorised bulk/systematic collection and using its API/content
   to build ML/AI datasets without explicit permission. Manual single-asset
   download preserves provenance and avoids treating footage as a dataset.

## Safe product claim

Use the footage only as a recorded camera preview. The adapter must label the
event `SIMULATED DETECTOR` / `RECORDED SCENARIO` until an actual authorised
vision model is integrated. For the second clip, a timestamp manifest can
trigger `missing_hard_hat` only after the manually verified no-hard-hat interval
starts.

## First-party sources

- [Pexels License](https://www.pexels.com/license/)
- [Pexels help: license of photos and videos](https://help.pexels.com/hc/en-us/articles/360042295174-What-is-the-license-of-the-photos-and-videos-on-Pexels)
- [Pexels help: terms and conditions](https://help.pexels.com/hc/en-us/articles/900005880463-What-are-the-Terms-and-Conditions)
- [Mixkit free stock-video licence overview](https://mixkit.co/free-stock-video/)

## Local curation outcome

One item was downloaded as a single, attributable demo asset:
[`../LICENSE-SOURCES.md`](../LICENSE-SOURCES.md) records its source URL, date,
format, and SHA-256. It is a 12.25-second Mixkit **Free License** clip in
which a high-vis worker begins without a hard hat and puts one on. It is
suitable for a transparent before/after *recorded scenario*, not as evidence
of a real model detection.

## Wide framing candidates

These are **recorded-preview** candidates with more of the work area in frame;
they are not yet approved as model-training data or evidence of an AI result.
Inspect the downloaded cut and record its provenance before adding it to the
demo.

1. **Recommended — indoor warehouse, compliant baseline:**
   [Pexels — Workers with safety helmets in warehouse](https://www.pexels.com/video/workers-with-safety-helmets-in-warehouse-10817415/).
   The item describes helmeted workers walking through an industrial warehouse;
   its tags include *Back View*, *Indoors*, *Large Space*, *Warehouse*, and
   *Hard Hats*. This is the closest fit to a fixed indoor CCTV view while
   avoiding face-led framing. The item is marked “Free to use”; apply the
   [Pexels License](https://www.pexels.com/license/) restrictions (no implied
   endorsement, offensive portrayal, or unaltered resale/redistribution).

2. **Strong construction alternative — group and site context:**
   [Mixkit — Construction workers at a house under construction](https://mixkit.co/free-stock-video/construction-workers-at-a-house-under-construction-1459/).
   Its description says two workers inspect plans while others work in the
   background, giving a multi-person work-area composition rather than a face
   close-up. The item page lists 1920×1080 and the Mixkit Stock Video **Free
   License** for commercial or personal use; re-check that individual licence
   state at download time because Mixkit also carries restricted clips.

3. **Wide overview only — useful as an establishing camera:**
   [Pexels — People working in a construction site](https://www.pexels.com/video/people-working-in-a-construction-site-3815641/).
   The item explicitly tags the view *Long Shot* and *Wide Shot*, with workers,
   hard hats, and heavy machinery visible. It is good for a camera-selector
   preview, but workers may be too small for credible PPE inference; keep the
   outcome a recorded scenario unless a model is evaluated on the original
   resolution. It is marked “Free to use” under the same Pexels License.

**Ranking decision:** start with candidate 1 for the in-app camera feed. Keep
candidate 2 as a fallback if a horizontal, busier construction scene reads
better in the dashboard. Candidate 3 is presentation context, not the primary
PPE-detection feed. None establishes a genuine no-hard-hat violation; continue
to model that state as a clearly labelled `RECORDED SCENARIO` until an
authorised, manually reviewed clip or a real vision model is available.
