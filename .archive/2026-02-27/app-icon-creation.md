---
date: 2026-02-27
title: KickWatch App Icon Creation
category: design
tags: [app-icon, logo, svg, nanobanana, rsvg-convert, xcode, appiconset]
related: [2026-02-27/mvp-implementation.md]
---

# App Icon Creation

## Concept

Kickstarter K shape (rounded bubbly white K on green #05CE78) + newspaper/daily digest metaphor = daily monitor app identity. Notion-style: flat, no gradients, clean.

## Process

1. Generated 7 variations via nanobanana skill (Gemini image gen)
2. Selected `logo-07.png` — but user preferred a manually provided `o.png`
3. Pipeline on `o.png`:
   - `crop_logo.py` → `final-cropped.png` (719×719 from 776×776)
   - `remove_bg.py` → `final-nobg.png` (170KB, transparent bg via remove.bg API)
   - `vectorize.py` → `final.svg` (10KB via Recraft API)

## SVG Centering Fix

The vectorized SVG had content off-center (bbox x:106–1983, y:285–1887 in 2000×2000 viewBox). Solution — wrap in centered group with white background:

```svg
<rect width="2000" height="2000" fill="white"/>
<g transform="translate(1000,1000) scale(0.82) translate(-1044.5,-1086)">
  <!-- original paths -->
</g>
```

Content center: (1044.5, 1086). Scale 0.82 gives ~10% padding on all sides.

## PNG Generation

Used `rsvg-convert` (available via homebrew at `/opt/homebrew/bin/rsvg-convert`):

```bash
for SIZE in 20 29 40 58 60 76 80 87 120 152 167 180 1024; do
  rsvg-convert -w $SIZE -h $SIZE final-centered.svg -o AppIcon-${SIZE}x${SIZE}.png
done
cp AppIcon-1024x1024.png AppIcon.png
```

**cairosvg NOT usable** on this machine — cairo native library missing. Use `rsvg-convert` instead.

## Contents.json

Mirrors SnapAction's `AppIcon.appiconset/Contents.json` exactly (18 image entries for iPhone + iPad + ios-marketing).

## Files

All source assets in:
`.skill-archive/logo-creator/2026-02-27-kickwatch-logo/`
- `final-centered.svg` ← source of truth for icon
- `final-nobg.png`, `final-cropped.png`, `final.svg`

Final PNGs committed to:
`ios/KickWatch/Assets.xcassets/AppIcon.appiconset/`
