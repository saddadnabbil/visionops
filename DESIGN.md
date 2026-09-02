# VisionOps Design System

## Source and Intent

This is the local implementation reference for VisionOps. It adapts the supplied Figma marketing design direction into an operational safety application: a monochrome frame, editorial typography, pill actions, and large, deliberate pastel color blocks. It does not copy Figma product UI; operational screens remain more compact and data-focused.

## Foundations

### Color tokens

| Token | Value | Use |
| --- | --- | --- |
| `primary` | `#000000` | Ink, primary pills, marquee, selected state. |
| `canvas` | `#FFFFFF` | Default application canvas. |
| `surface-soft` | `#F4F4F4` | Secondary controls and quiet tiles. |
| `hairline` | `#D4D4D4` | Borders, table rules, inputs. |
| `block-lime` | `#D9FF00` | Live operations and attention group. |
| `block-lilac` | `#D9D2FF` | Analytics and system context. |
| `block-cream` | `#FFF8E7` | Quiet templates/empty-state context. |
| `block-mint` | `#C9F6DE` | Optional success/process context. |
| `block-pink` | `#F8C8E8` | Optional editorial context only. |
| `block-coral` | `#FFB39F` | Optional developer/integration context. |
| `block-navy` | `#171B4F` | Rare inverse escalation/callout. |
| `semantic-success` | `#16803C` | Success glyph, never a broad surface. |
| `accent-magenta` | `#FF4FA3` | One-off promo only; not an operational status. |

Status always requires a textual label and shape/icon in addition to color.

### Typography

- Sans: `Inter, SF Pro Display, system-ui, helvetica`; use variable weights 320/330/340/480/540/700 when available.
- Mono: `JetBrains Mono, SF Mono, Menlo, monospace`; only for labels, status, timestamps, taxonomy.
- Enable kerning.

| Token | Desktop | Mobile | Use |
| --- | --- | --- | --- |
| `display-xl` | 86px/1.0/-1.72px/340 | 48px/1.0 | Overview/marketing hero only. |
| `display-lg` | 64px/1.1/-0.96px/340 | 40px/1.1 | Large color-block title. |
| `page-title` | 40-56px/1.05/340 | 32-40px | Operational landing title. |
| `headline` | 26px/1.35/540 | 24px | Component headings. |
| `subhead` | 26px/1.35/340 | 20px | Editorial block body. |
| `body` | 18px/1.45/320 | 16px | Default readable text. |
| `body-sm` | 16px/1.45/330 | 16px | Rows/cards. |
| `eyebrow` | 12-18px/1.3/400 | 12px | Uppercase mono labels. |

### Spacing and shape

- Base unit: 8px. Use 4, 8, 12, 16, 24, 32, 48, and 96px.
- Desktop content maximum: 1280-1440px; 32px gutters. Mobile: 24px gutters.
- Large color-block: 48px padding, 24px radius; mobile becomes full-bleed, square outer edge, 24px padding.
- Input/list item: 8px radius. Text action: 50px pill. Icon action: fully round.
- Avoid square buttons. Use hairlines before shadows; shadow only for modal/popover layers.

## Component Standards

### Buttons

- Primary: black surface, white text, 20px/480 type, 10px 20px padding, pill shape.
- Secondary: soft/white surface, black text, pill shape.
- Button press feedback: `transform: scale(0.97)` over 120-160ms ease-out.
- One primary action maximum within a visible section; demote competing actions to secondary/text.

### Operational components

| Component | Required states |
| --- | --- |
| `AppShell` | desktop sidebar, tablet route strip, mobile drawer |
| `PageHeader` | title, subtitle, action slot, loading/error context |
| `StatusBadge` | incident, camera, delivery states with text + icon |
| `IncidentQueueItem` | default, selected, critical, loading, empty |
| `IncidentDetail` | drawer desktop, page mobile, read-only, mutable |
| `ResolutionForm` | default, validation, saving, failed, saved |
| `FilterBar` | inline desktop, bottom sheet mobile, removable chips |
| `DataTable` | loading, empty, error, compact; becomes list on mobile |
| `MetricBlock` | neutral, attention, inverse; name + value + time basis |
| `CameraHealthCard` | online, degraded, offline, no-heartbeat |
| `DeliveryJobRow` | pending, retrying, dead, done; confirmed replay |
| `EmptyState` | no data, no match, denied, setup-needed |

## Responsive Behavior

| Breakpoint | Layout rule |
| --- | --- |
| >= 960px | 64px top bar plus 224px role sidebar; optional 320px context rail. |
| 768-959px | Sidebar becomes horizontal route strip; secondary table metadata moves into detail. |
| < 768px | 56px top bar, drawer nav, lists replace tables, detail opens as page, sticky operator action bar. |
| < 560px | Display titles reduce, pill actions full width where needed. |

All interactive targets are at least 44px. Mobile never relies on hover. At 200% zoom, actions must remain visible and unobscured.

## Interaction, Motion, and Accessibility

- Use no animation for frequent keyboard/navigation actions.
- Drawer/dialog entry: opacity plus 0.97 scale or edge translate, 180-240ms strong ease-out. Dialogs originate center; drawers originate from their edge.
- Respect `prefers-reduced-motion` by removing transform/entrance effects.
- Preserve stale data during refresh; show small `Updating` state instead of layout jump.
- API error preserves context and provides retry. Mutation error preserves form draft and moves focus to error summary.
- SSE loss shows `Live updates paused — retrying`; critical lists poll every 30 seconds as fallback.
- Use semantic landmarks, visible focus, AA contrast, text status labels, dialog focus trap/restore, live-region result messages, and accessible chart table equivalents.

## Do and Do Not

### Do

- Use white canvas between large colored sections so each block remains intentional.
- Use mono only as taxonomy, not paragraph copy.
- Use weight rather than gray opacity to establish hierarchy.
- Reuse existing component variants before inventing a new visual pattern.
- Treat desktop, tablet, and mobile states as separate deliberate layouts.

### Do not

- Add gradients, decorative shadows, square CTAs, or arbitrary accent colors.
- Use color as the only severity/health indicator.
- Hide data after transient refresh failure.
- Make a desktop table horizontally scroll on mobile; transform it into a list/detail pattern.
- Animate repeated high-frequency work or keyboard actions.

## Design QA Gate

1. Review at 1440px, 768px, and 390px.
2. Complete an Operator incident flow with keyboard only.
3. Test each role's landing, empty, failure, and denied state.
4. Verify screen-reader labels for lists, dialogs, validation, and state changes.
5. Add a component or token here before using a new pattern in production UI.

## UI Consistency Pass — 2026-09-02

- Replace native selects with the reusable `select-menu`: a button trigger,
  listbox semantics, hidden form value, and Escape/outside-click dismissal.
  It is used for Incident filtering and Admin user-role selection.
- Keep mobile header controls on a shared 44px line. Retain the role chip until
  screens narrower than 390px, where it is intentionally hidden to prevent
  overflow rather than partially clipping the bar.
- Treat Incident detail as a bounded mobile bottom sheet: the header/close
  target stays sticky inside its own scroll container, never as an oversized
  desktop modal with an outer scrollbar.
- Use independent rounded metric blocks and a 12px mobile gap so analytics no
  longer inherits desktop-card proportions.

## Landing-to-workspace entry — 2026-09-02

- `/` is the public VisionOps landing page; `/login` is the explicit workspace
  entry. Do not make the dashboard or login form the first unexplained screen.
- The landing page follows the same monochrome frame, editorial display type,
  pill CTAs, and isolated lime/navy story panels as the product UI.
- The landing page states the recorded-demo and privacy boundary before a user
  can enter the workspace. It must never imply live CCTV, face recognition, or
  production model inference.
