# Recorded-demo assets

VisionOps is fully runnable without video. The `camera-demo` and
`recorded-demo` adapters submit their fixture events independently from the
browser preview.

The recorded video files are deliberately excluded from Git. Their providers
permit certain uses, but public redistribution must be reviewed separately for
each item. The source register and the product boundary are in
[`LICENSE-SOURCES.md`](../LICENSE-SOURCES.md).

## Optional local preview

To enable the camera-page preview in a local demo, obtain the approved asset
manually and place it at:

```text
web/assets/demo/construction-site-wide-recorded-scenario.mp4
```

Keep the source master and any derived cuts outside Git under `assets/demo/`.
Before replacing or publishing an asset, record its source URL, licence state,
download date, and SHA-256 in `LICENSE-SOURCES.md`.

Never use the preview as a claim of live CCTV, face recognition, or automated
PPE inference. The event is a labelled recorded scenario until an authorised
detector integration has been evaluated and approved.
