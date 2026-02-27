<coding_guidelines>
# KickWatch

Daily Kickstarter campaign monitor. iOS app + Go backend deployed on AWS ECS.

## Core Commands

### iOS
- Generate Xcode project: `cd ios && xcodegen generate`
- Build: `xcodebuild -project ios/KickWatch.xcodeproj -scheme KickWatch build`

### Backend
- Run dev server: `cd backend && go run ./cmd/api`
- Run tests: `cd backend && go test ./...`
- Build binary: `cd backend && go build -o api ./cmd/api`
- Deploy: see `.github/workflows/deploy-backend.yml`

## Project Layout

```
├── ios/                        → SwiftUI app (iOS 17+, XcodeGen)
│   ├── KickWatch/Sources/
│   │   ├── App/                → App entry, ContentView
│   │   ├── Models/             → SwiftData models
│   │   ├── Views/              → UI views
│   │   ├── ViewModels/         → View models
│   │   └── Services/           → APIClient, NotificationService
│   ├── Assets.xcassets/
│   ├── project.yml             → XcodeGen config (source of truth)
│   └── Package.swift           → SPM dependencies
└── backend/                    → Go + Gin API server
    ├── cmd/api/                → Entry point
    └── internal/
        ├── handler/            → HTTP handlers (campaigns, alerts, devices)
        ├── service/            → kickstarter_rest, kickstarter_graph, cron, apns
        ├── model/              → GORM models
        ├── config/             → Configuration
        └── middleware/         → Logging, error handling
```

## Development Patterns

### iOS
- SwiftUI + SwiftData, deployment target iOS 17.0
- `.xcodeproj` is gitignored; always regenerate with `xcodegen generate`
- Info.plist properties must be defined in `project.yml` `info.properties`
- Assets.xcassets listed under `sources:` in project.yml (not `resources:`)

### Backend
- Go 1.24, Gin framework
- Two Kickstarter data sources:
  - REST `/discover/advanced.json` — no auth, used for nightly crawl
  - GraphQL `POST /graph` — anonymous session (CSRF token + cookie), used for search + detail
- Session bootstrap: GET kickstarter.com → extract `_ksr_session` cookie + `<meta name="csrf-token">` → reuse for GraphQL
- Refresh session every 12h or on 403
- Tests: `go test ./...`

## Kickstarter GraphQL
- Endpoint: `POST https://www.kickstarter.com/graph`
- Key queries: `Search`, `FetchProjectBySlug`, `FetchRootCategories`
- Source of truth for query structure: https://github.com/kickstarter/ios-oss/tree/main/graphql
- `Search` sort enum: `MAGIC | NEWEST | END_DATE | MOST_BACKED | MOST_FUNDED`
- `state` enum: `LIVE | SUCCESSFUL | FAILED | CANCELED`
- Pagination: cursor-based (`first` + `after` cursor from `pageInfo.endCursor`)

## Git Workflow

Git Flow branching:
- `main` — production, never push directly
- `develop` — integration branch
- `feature/<name>` — branch from develop, PR → develop
- `fix/<name>` — branch from develop, PR → develop
- `hotfix/<name>` — branch from main, merge to main + develop

### Commit Convention
- `feat:` / `fix:` / `docs:` / `test:` / `refactor:` / `chore:` / `security:`
- Reference issues: `feat(#12): add keyword alerts`
- No AI-generated signatures in commit messages

### Rules
- Never merge directly into `main`
- Never force-push `main` or `develop`
- Delete merged branches immediately
- Use `git worktree` for feature branches

## Security
- Never commit `.env` files or API keys
- Backend secrets via AWS Secrets Manager / ECS task definition environment
- No Kickstarter user credentials stored anywhere

## Gotchas
- XcodeGen regenerates Info.plist — put all plist keys in `project.yml`
- `Assets.xcassets` must be under `sources:` not `resources:` in project.yml
- Adding new SwiftData `@Model` default entries may require bumping `schemaVersion`
- Kickstarter CSRF token must be refreshed; do not cache indefinitely
- ECS task env vars added manually are wiped on next CI deploy — always add to deploy workflow

## Archived Knowledge
Before debugging or repeating past work, consult .archive/MEMORY.md
</coding_guidelines>
