# Project Knowledge Archive

Archived learnings, debugging solutions, and infrastructure notes.
Search: `grep -ri "keyword" .archive/`

## Infrastructure & AWS

## Release & Deploy
- `2026-02-27/mvp-implementation.md` — Full MVP build: Go backend + iOS app, git workflow, CI/CD, repo published to ReScienceLab/KickWatch

## Debugging & Fixes
- `2026-02-27/mvp-implementation.md` — .gitignore blocked `.env.example` (fix: replace `*.env` with explicit patterns + `!.env.example`); `go vet` failed on duplicate json tag in restProject struct

## Features
- `2026-02-27/mvp-implementation.md` — Backend: Kickstarter REST+GraphQL clients, APNs, nightly cron, full REST API. iOS: SwiftData models, APIClient actor, all 4 tabs, cursor pagination
- `2026-02-27/mvp-implementation.md` — CI/CD: GitHub Actions test + ECS deploy workflows

## Design
- `2026-02-27/app-icon-creation.md` — App icon: K + newspaper concept, Notion-style. SVG centering transform, rsvg-convert for all 14 PNG sizes (cairosvg broken on this machine — use rsvg-convert)
