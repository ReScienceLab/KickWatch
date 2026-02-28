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

We use **Git Flow** for version control.

### Branching Strategy

- `main` — Production branch, always deployable, tagged releases
- `develop` — Integration branch for features
- `feature/<name>` — New features (branch from develop, PR → develop)
- `fix/<name>` — Bug fixes (branch from develop, PR → develop)
- `hotfix/<name>` — Urgent production fixes (branch from main, merge to main + develop)
- `release/<version>` — Release prep (branch from develop, merge to main + develop, tag)

### Rules

- Feature/fix branches PR directly to `develop` — no chaining PRs
- **Never merge directly into `main`** — `main` is only updated via PR (release or hotfix branches)
- To sync: only sync `develop` (pull/merge PRs). Do NOT fast-forward `main` from `develop`.
- Merged branches must be deleted immediately
- Never force-push `main` or `develop`
- Use `git worktree` for feature branches to avoid disrupting the main working directory

### Commit Convention

- `feat:` — New features
- `fix:` — Bug fixes
- `docs:` — Documentation
- `test:` — Test additions/changes
- `refactor:` — Code refactoring
- `chore:` — Maintenance
- `security:` — Security fixes

Reference issues: `feat(#12): add keyword alerts`

**Important**: Do not add any watermark or AI-generated signatures to commit messages.

### Issue Management

When creating new issues:
1. **Add type labels** to categorize the issue:
   - `bug` - Something isn't working
   - `feature` - New feature request
   - `enhancement` - Improvement to existing feature
   - `documentation` - Documentation improvements
   - `refactor` - Code refactoring needed
   - `test` - Test-related issues
   - `chore` - Maintenance tasks

2. **Add tag labels** for organization:
   - `priority:high` / `priority:medium` / `priority:low` - Priority level
   - `good first issue` - Good for newcomers
   - `help wanted` - Extra attention needed
   - Area-specific tags: `ios`, `backend`, `api`, `ci/cd`, etc.

3. **Write clear descriptions**:
   - For bugs: Include reproduction steps, expected vs actual behavior
   - For features: Describe the use case and desired outcome
   - Reference related issues or PRs if applicable

### PR Requirements

1. All tests must pass (`cd backend && go test ./...`)
2. Feature branches merge to `develop` via PR
3. Hotfix branches merge to both `main` and `develop`
4. Releases: `develop` → `main` via PR
5. If a PR corresponds to an issue, reference the issue number in the PR description to auto-link them (e.g., `#123`)
6. To auto-close the issue on merge, use closing keywords (e.g., `Fixes #123`, `Closes #123`, or `Resolves #123`)

### Versioning

Semantic versioning: `vMAJOR.MINOR.PATCH`
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

### Release Checklist

1. Create `release/vX.Y.Z` branch from develop
2. Bump version in `ios/project.yml` (`CFBundleShortVersionString` + `CFBundleVersion`)
3. Update `CHANGELOG.md` with the release notes
4. Run all tests: `cd backend && go test ./...`
5. Merge release branch to `main` via PR
6. Tag: `git tag vX.Y.Z && git push origin vX.Y.Z`
7. Merge release branch back to `develop`
8. Archive and upload to TestFlight
9. Delete release branch

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
