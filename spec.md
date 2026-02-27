# KickWatch — Daily Kickstarter Monitor

iOS app + Go backend for discovering, tracking, and getting daily digests of Kickstarter crowdfunding campaigns.

---

## Problem

Kickstarter launches hundreds of campaigns daily. There is no good mobile tool to monitor specific categories or keywords and get alerted when relevant campaigns launch or cross funding thresholds. The official Kickstarter app is creator-focused and has no watchlist or alert system for backers.

---

## Solution

KickWatch is a backer-first iOS app that:
- Aggregates Kickstarter campaigns via a Go backend that scrapes Kickstarter's undocumented JSON endpoints daily
- Lets users browse by category, search, and save campaigns to a watchlist
- Sends daily APNs push digests for user-defined keyword/category alerts

---

## Architecture

```
[iOS App] ←→ [Go Backend (ECS)] ←→ [Kickstarter JSON endpoints]
                    ↕
              [Postgres (RDS)]
                    ↕
              [APNs (Apple)]
```

### Data sourcing

Kickstarter exposes unauthenticated JSON endpoints used by their web frontend:

- **Discover**: `https://www.kickstarter.com/discover/advanced.json?category_id=<id>&sort=<sort>&page=<n>`
- **Search**: `https://www.kickstarter.com/projects/search.json?term=<q>&page=<n>`

Sort options: `magic` (trending), `newest`, `end_date`, `most_backed`, `most_funded`

Supported category IDs (subset):
| ID | Name |
|---|---|
| 1 | Art |
| 7 | Design |
| 10 | Food |
| 11 | Film & Video |
| 12 | Games |
| 14 | Music |
| 16 | Technology |
| 18 | Publishing |

A Go `cron` job runs nightly (02:00 UTC) to refresh all categories and diff against previous snapshots. Results are stored in Postgres and served to the iOS app via REST API.

---

## Tech Stack

| Layer | Technology |
|---|---|
| iOS | SwiftUI, SwiftData, iOS 17+ |
| Backend | Go 1.24, Gin framework |
| Database | Postgres (AWS RDS) |
| Infrastructure | AWS ECS (Fargate), ECR |
| Notifications | APNs (direct HTTP/2, no Firebase) |
| Scheduling | Go `robfig/cron` v3 |
| CI/CD | GitHub Actions |

---

## iOS App

### Tab Structure

```
ContentView (TabView)
├── [Discover]  DiscoverView
├── [Watchlist] WatchlistView
├── [Alerts]    AlertsView
└── [Settings]  SettingsView
```

### Screens

#### DiscoverView
- Segmented control: **Trending** / **New** / **Ending Soon**
- Horizontal scrollable category filter chips
- `CampaignListView` → paginated list of `CampaignRowView`
- Pull-to-refresh

#### CampaignRowView
- Thumbnail (cached), title, creator name
- Funding progress bar with `%` and days-left label
- Backer count
- Watchlist heart button (tap to toggle, no navigation)

#### CampaignDetailView
- Hero image, title, blurb
- Funding ring (% funded), goal, pledged, backers, deadline
- Reward tiers (if available)
- "Back this project" → `openURL` to Kickstarter
- Share sheet
- Add/Remove Watchlist button

#### WatchlistView
- List of saved campaigns sorted by days left
- Status badge: `Live` / `Funded` / `Ended` / `Failed`
- Swipe-to-remove
- Empty state CTA → Discover

#### AlertsView
- List of user keyword alerts
- Tap alert → show matched campaigns from last digest
- "New Alert" sheet: keyword, optional category, optional min-funding-%
- Enable/disable toggle per alert

#### SettingsView
- Notification preferences (digest time, enable/disable)
- App version, feedback link

---

## Data Models

### SwiftData (local cache + watchlist)

```swift
@Model
class Campaign {
    @Attribute(.unique) var id: String
    var name: String
    var blurb: String
    var photoURL: String
    var goal: Double
    var pledged: Double
    var backersCount: Int
    var deadline: Date
    var state: String           // "live" | "successful" | "failed" | "canceled" | "suspended"
    var categoryName: String
    var categoryID: Int
    var projectURL: String
    var creatorName: String
    var percentFunded: Double   // pledged / goal * 100
    var isWatched: Bool
    var lastFetchedAt: Date
}

@Model
class WatchlistAlert {
    @Attribute(.unique) var id: String
    var name: String
    var keyword: String
    var categoryID: Int?        // nil = all categories
    var minPercentFunded: Double
    var isEnabled: Bool
    var createdAt: Date
    var lastMatchedAt: Date?
}

@Model
class RecentSearch {
    var query: String
    var searchedAt: Date
}
```

### Postgres (backend)

```sql
campaigns (
    id              TEXT PRIMARY KEY,   -- Kickstarter project ID
    name            TEXT,
    blurb           TEXT,
    photo_url       TEXT,
    goal            NUMERIC,
    pledged         NUMERIC,
    backers_count   INTEGER,
    deadline        TIMESTAMPTZ,
    state           TEXT,
    category_id     INTEGER,
    category_name   TEXT,
    project_url     TEXT,
    creator_name    TEXT,
    percent_funded  NUMERIC,
    first_seen_at   TIMESTAMPTZ,
    last_updated_at TIMESTAMPTZ
)

devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_token    TEXT UNIQUE NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT now()
)

alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID REFERENCES devices(id) ON DELETE CASCADE,
    keyword         TEXT NOT NULL,
    category_id     INTEGER,            -- NULL = all categories
    min_percent     NUMERIC DEFAULT 0,
    is_enabled      BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT now(),
    last_matched_at TIMESTAMPTZ
)

alert_matches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id    UUID REFERENCES alerts(id) ON DELETE CASCADE,
    campaign_id TEXT REFERENCES campaigns(id),
    matched_at  TIMESTAMPTZ DEFAULT now()
)
```

---

## Backend API

### Campaigns

```
GET /api/campaigns
  ?sort=trending|newest|ending        default: trending
  ?category_id=<int>                  optional
  ?page=<int>                         default: 1
  ?per_page=<int>                     default: 20, max: 50
  → { campaigns: [...], total: int, page: int }

GET /api/campaigns/search
  ?q=<string>
  ?category_id=<int>                  optional
  ?page=<int>
  → { campaigns: [...], total: int }

GET /api/campaigns/:id
  → Campaign object

GET /api/categories
  → [{ id: int, name: string }]
```

### Devices & Alerts

```
POST /api/devices/register
  body: { device_token: string }
  → { device_id: uuid }

POST /api/alerts
  body: { device_id, keyword, category_id?, min_percent? }
  → Alert object

GET /api/alerts?device_id=<uuid>
  → [Alert]

PATCH /api/alerts/:id
  body: { is_enabled?: bool, keyword?, category_id?, min_percent? }
  → Alert object

DELETE /api/alerts/:id

GET /api/alerts/:id/matches
  → [Campaign] (last digest matches)
```

---

## Background Cron Job

**Schedule**: every day at 02:00 UTC

**Steps**:
1. For each category (16 total), fetch pages 1–5 from Kickstarter discover endpoint (≈ 240 campaigns/category)
2. Upsert into `campaigns` table; mark `first_seen_at` on new rows
3. Find campaigns first seen in the last 24h (`first_seen_at > now() - interval '1 day'`)
4. For each enabled alert:
   - Filter new campaigns by keyword (case-insensitive ILIKE), optional category, optional min_percent
   - If ≥ 1 match: insert `alert_matches` rows, send APNs push to the device
5. Log results

**APNs Push Payload**:
```json
{
  "aps": {
    "alert": {
      "title": "KickWatch: 5 new \"mechanical keyboard\" projects",
      "body": "Tap to see today's matches"
    },
    "badge": 1,
    "sound": "default"
  },
  "alert_id": "<uuid>",
  "match_count": 5
}
```

---

## Project Layout

```
kickwatch/
├── spec.md
├── README.md
├── .github/
│   └── workflows/
│       ├── deploy-backend.yml
│       └── test-backend.yml
├── ios/
│   ├── project.yml               ← XcodeGen config
│   ├── Package.swift             ← SPM dependencies
│   └── KickWatch/
│       └── Sources/
│           ├── App/
│           │   ├── KickWatchApp.swift
│           │   └── ContentView.swift
│           ├── Models/           ← SwiftData models
│           ├── Views/
│           │   ├── DiscoverView.swift
│           │   ├── CampaignRowView.swift
│           │   ├── CampaignDetailView.swift
│           │   ├── WatchlistView.swift
│           │   ├── AlertsView.swift
│           │   └── SettingsView.swift
│           ├── ViewModels/
│           └── Services/
│               ├── APIClient.swift
│               └── NotificationService.swift
└── backend/
    ├── cmd/
    │   └── api/
    │       └── main.go
    └── internal/
        ├── config/
        ├── handler/
        │   ├── campaigns.go
        │   ├── alerts.go
        │   └── devices.go
        ├── model/
        ├── service/
        │   ├── kickstarter.go    ← HTTP client for KS endpoints
        │   ├── cron.go           ← nightly refresh + alert matching
        │   └── apns.go           ← APNs HTTP/2 sender
        └── middleware/
```

---

## Development Commands

```bash
# Backend
cd backend && go run ./cmd/api
cd backend && go test ./...
cd backend && go build -o api ./cmd/api

# iOS
cd ios && xcodegen generate
xcodebuild -project ios/KickWatch.xcodeproj -scheme KickWatch build
```

---

## MVP Scope (v1)

- [ ] Go backend: Kickstarter scraper + Postgres upsert + REST API
- [ ] Nightly cron job with APNs push
- [ ] iOS: Discover feed (Trending/New/Ending) with category filter
- [ ] iOS: Campaign detail + watchlist save (SwiftData)
- [ ] iOS: Keyword alerts + digest view
- [ ] iOS: Search

**Out of v1**: user accounts, cross-device sync, funding threshold alerts (% crossed), creator follow, charts/history, subscription gating.

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Kickstarter changes/blocks undocumented endpoints | Backend proxy layer absorbs breakage; app shows cached data |
| Kickstarter ToS violation | App is personal/non-commercial; no resale of data |
| APNs delivery failures | Retry logic with exponential backoff; clean up invalid device tokens |
| Stale campaign data | TTL badge on campaign cards showing `last updated X ago` |
