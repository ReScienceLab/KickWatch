---
date: 2026-02-27
title: KickWatch MVP Implementation
category: feature
tags: [go, gin, gorm, swiftui, swiftdata, kickstarter, graphql, apns, cron, xcodegen]
---

# KickWatch MVP Implementation

## What Was Built

Full MVP from scratch across 22+ atomic commits on `develop` branch.

### Backend (Go/Gin) — `backend/`

**Module**: `github.com/kickwatch/backend`

**Key packages added**:
```
github.com/gin-gonic/gin
github.com/joho/godotenv
github.com/google/uuid
github.com/golang-jwt/jwt/v5
github.com/robfig/cron/v3
gorm.io/gorm
gorm.io/driver/postgres
```

**Structure**:
- `internal/config/` — env var loader
- `internal/model/` — GORM models: Campaign, Category, Device, Alert, AlertMatch
- `internal/db/` — AutoMigrate on startup
- `internal/middleware/` — CORS, Logger
- `internal/service/kickstarter_rest.go` — REST /discover/advanced.json (no auth, nightly crawl)
- `internal/service/kickstarter_graph.go` — GraphQL /graph with session bootstrap (CSRF + _ksr_session cookie), 12h refresh, 403 retry
- `internal/service/apns.go` — APNs HTTP/2 with JWT signing (golang-jwt ES256)
- `internal/service/cron.go` — nightly 02:00 UTC crawl, 15 categories × 10 pages, upsert + alert matching + APNs push
- `internal/handler/` — campaigns, search, categories, devices, alerts CRUD

**API routes**:
```
GET  /api/health
GET  /api/campaigns?sort=trending|newest|ending&category_id=&cursor=&limit=
GET  /api/campaigns/search?q=&category_id=&cursor=
GET  /api/campaigns/:pid
GET  /api/categories
POST /api/devices/register
POST /api/alerts
GET  /api/alerts?device_id=
PATCH /api/alerts/:id
DELETE /api/alerts/:id
GET  /api/alerts/:id/matches
```

**Gotcha**: `restProject` struct had duplicate json tag `"urls"` on two fields — caused `go vet` failure. Fixed by removing the unused `URL string` field.

### iOS (SwiftUI/SwiftData) — `ios/`

**project.yml** key settings:
```yaml
bundleIdPrefix: com.kickwatch
deploymentTarget: iOS: "17.0"
PRODUCT_BUNDLE_IDENTIFIER: com.kickwatch.app
DEVELOPMENT_TEAM: ""   # fill in before building
```

**SwiftData models**: Campaign, WatchlistAlert, RecentSearch

**Services**:
- `APIClient` — actor, base URL switches DEBUG/Release, supports GET/POST/PATCH/DELETE
- `KeychainHelper` — identical pattern to SnapAction
- `NotificationService` — @MainActor ObservableObject, registers APNs token via APIClient, stores device_id in Keychain
- `ImageCache` — actor-based URL→Image cache with RemoteImage SwiftUI view

**ViewModels**: `@Observable` (iOS 17 pattern, not ObservableObject)
- `DiscoverViewModel` — sort + category filter + cursor pagination
- `AlertsViewModel` — full CRUD

**Views**: DiscoverView, CampaignRowView, CampaignDetailView (funding ring), WatchlistView, AlertsView, AlertMatchesView, SearchView, SettingsView, CategoryChip

**App entry**: `KickWatchApp` with `@UIApplicationDelegateAdaptor(AppDelegate.self)` for APNs token registration

### CI/CD
- `.github/workflows/test-backend.yml` — triggered on `backend/**` changes, runs `go build`, `go test`, `go vet`
- `.github/workflows/deploy-backend.yml` — triggered on `main` push, builds Docker image, pushes to ECR, deploys to ECS

## Git Workflow

- Worktree created at `.worktrees/develop` for `develop` branch
- `.worktrees/` added to `.gitignore` before creation
- All work on `develop`, never touched `main` directly
- Published repo: https://github.com/ReScienceLab/KickWatch

## .gitignore Fix

Original pattern `*.env` blocked `.env.example`. Changed to:
```
.env
.env.local
.env.production
!.env.example
```

## App Icon

- Generated via Gemini (nanobanana skill): Kickstarter K + newspaper/daily digest metaphor, Notion-style flat design, green (#05CE78)
- Best result: `logo-07.png` → user provided `o.png` as final source
- Processed: crop → remove_bg (remove.bg API) → vectorize (Recraft API) → SVG
- Centered in white background SVG: `final-centered.svg`
  - Transform: `translate(1000,1000) scale(0.82) translate(-1044.5,-1086)` (content bbox: x 106–1983, y 285–1887)
- All 14 PNG sizes generated with `rsvg-convert` (homebrew):
  ```bash
  rsvg-convert -w $SIZE -h $SIZE input.svg -o AppIcon-${SIZE}x${SIZE}.png
  ```
  Sizes: 20, 29, 40, 58, 60, 76, 80, 87, 120, 152, 167, 180, 1024

## Commands Reference

```bash
# Backend
cd backend && go run ./cmd/api
cd backend && go test ./...
cd backend && go build ./... && go vet ./...

# iOS
cd ios && xcodegen generate
xcodebuild -project ios/KickWatch.xcodeproj -scheme KickWatch build

# Worktree
git worktree add .worktrees/develop -b develop
cd .worktrees/develop

# Icon generation
rsvg-convert -w 1024 -h 1024 final-centered.svg -o AppIcon-1024x1024.png
```
