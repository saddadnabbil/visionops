# Demo asset sources

This register records third-party assets used only for the local VisionOps
recorded demonstration. It is not an endorsement, training-data register, or
evidence that VisionOps performed model inference.

| Local asset | Source / creator | Licence at download | Downloaded | Integrity |
| --- | --- | --- | --- |
| `assets/demo/mixkit-worker-hard-hat-transition-1440-720.mp4` | [Mixkit — Man puts on hard hat](https://mixkit.co/free-stock-video/man-puts-on-hard-hat-1440/) | Mixkit Stock Video Free License; page states commercial and personal use | 2026-09-02 | SHA-256 `43f62076a27f6e551f1dd14f8e5ed8b8a8c2d7233c53caf6eee6ca0efae6d533` |
| `assets/demo/mixkit-construction-site-wide-1459-720.mp4` | [Mixkit — Construction workers at a house under construction](https://mixkit.co/free-stock-video/construction-workers-at-a-house-under-construction-1459/) | Mixkit Stock Video Free License; page states commercial and personal use | 2026-09-02 | SHA-256 `19cb686ffcb32bafbc27216ea6d4d5abd0f0aa469ed31f37e63586f024a591e2` |

The downloaded asset is 1280×720 H.264, 24 fps, and 12.25 seconds. Its source
page describes a worker in a fluorescent vest putting on a yellow hard hat.
The provider's item-level licence is the authority; retain a browser screenshot
of that page before any public deployment.

Derived local previews (no audio) retain the same source attribution:

| Local derivative | State | Duration | SHA-256 |
| --- | --- | --- | --- |
| `assets/demo/missing-hard-hat-recorded-scenario.mp4` | Worker has not yet put on the hard hat | 1.5 s | `22b1fd6fa8c43f39c71054ad6eb9c8159da78a71e60ed884ef2e397efbb60319` |
| `assets/demo/hard-hat-compliant-recorded-scenario.mp4` | Worker is wearing the hard hat and high-vis vest | 8 s | `1a756f822feb1b4ecbfce77826d80f05f053c68b2b9ef0ff4ca6bfe8821c990e` |

The wide construction-site clip is 1280×720 H.264, 24 fps, and 9 seconds. It
shows the work area and workers at a distance, so it is suitable for an
anonymous recorded-camera preview. It does not establish PPE compliance or an
actual PPE violation; retain the `RECORDED SCENARIO` disclosure.

## Use boundary

- This file is recorded playback only. It is not training data and must not be
  uploaded to a model-training service.
- The current demo must label generated events `RECORDED SCENARIO` and
  `SIMULATED DETECTOR` until an approved model actually evaluates authorised
  footage.
- Do not identify the person in the footage, derive biometric data, or publish
  a face crop.
- Re-check the provider licence before redistributing the file or publishing it
  in a public repository.
