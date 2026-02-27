# KickWatch — Daily Kickstarter Monitor

iOS app + Go backend for discovering, tracking, and getting daily digests of Kickstarter crowdfunding campaigns.

---

## Problem

Kickstarter launches hundreds of campaigns daily. There is no good mobile tool to monitor specific categories or keywords and get alerted when relevant campaigns launch or cross funding thresholds. The official Kickstarter app is creator-focused and has no watchlist or alert system for backers.

---

## Solution

KickWatch is a backer-first iOS app that:
- Aggregates Kickstarter campaigns via a Go backend that calls Kickstarter's internal GraphQL API nightly
- Lets users browse by category, search, and save campaigns to a watchlist
- Sends daily APNs push digests for user-defined keyword/category alerts

---

## Architecture Decisions (Resolved)

| Question | Decision |
|---|---|
| Repo | New standalone repo at `KickWatch/` (not in SnapAction) |
| Backend infra | New ECS service (separate task definition + service from SnapAction) |
| Auth | Anonymous — device-keyed via UUID stored in Keychain, no Apple Sign-In |
| Data source | Kickstarter internal GraphQL API (primary) + REST discover endpoint (fallback) |

---

## Architecture

```
[iOS App] ←→ [Go Backend (new ECS service)] ←→ [Kickstarter /graph GraphQL]
                          ↕                           ↕ (fallback)
                   [Postgres (RDS)]         [Kickstarter /discover REST]
                          ↕
                   [APNs (Apple)]
```

---

## Kickstarter API Investigation

### Primary: Internal GraphQL API

Kickstarter's own iOS app (open-sourced at [kickstarter/ios-oss](https://github.com/kickstarter/ios-oss)) uses Apollo GraphQL. All queries are publicly readable in their repo.

**Endpoint**: `POST https://www.kickstarter.com/graph`

**Required headers**:
```
Content-Type: application/json
x-csrf-token: <token from page meta>
Cookie: _ksr_session=<anonymous session cookie>
User-Agent: Mozilla/5.0 ...
```

**Session bootstrap** (no login required):
1. `GET https://www.kickstarter.com` → extract `_ksr_session` cookie + CSRF token from `<meta name="csrf-token">` tag
2. Reuse both for all subsequent GraphQL POSTs
3. Refresh every 12 hours or on 403 response

**Key queries** (sourced directly from kickstarter/ios-oss `graphql/` folder):

#### Search / Discover
```graphql
query Search(
  $term: String,
  $sort: ProjectSort,       # MAGIC | NEWEST | END_DATE | MOST_BACKED | MOST_FUNDED
  $categoryId: String,      # "16" for Technology, "12" for Games, etc.
  $state: PublicProjectState, # LIVE | SUCCESSFUL | FAILED | CANCELED
  $raised: RaisedBuckets,   # NONE | BETWEEN_0_AND_20 | BETWEEN_20_AND_100 | BETWEEN_100_AND_1000 | ABOVE_1000
  $pledged: PledgedBuckets,
  $goal: GoalBuckets,
  $showProjectsWeLove: Boolean,  # staff picks
  $first: Int,              # page size (max 24)
  $cursor: String           # cursor-based pagination
) {
  projects(
    term: $term, sort: $sort, categoryId: $categoryId, state: $state,
    raised: $raised, pledged: $pledged, goal: $goal,
    staffPicks: $showProjectsWeLove, after: $cursor, first: $first
  ) {
    nodes { ...ProjectCardFragment }
    totalCount
    pageInfo { endCursor hasNextPage }
  }
}
```

#### ProjectCardFragment (fields returned per campaign)
```graphql
fragment ProjectCardFragment on Project {
  pid          # numeric ID
  name
  state        # "live" | "successful" | "failed" | "canceled"
  isLaunched
  deadlineAt
  percentFunded
  url          # https://www.kickstarter.com/projects/creator/slug
  image { url(width: 1024) }
  goal    { amount currency symbol }
  pledged { amount currency symbol }
  # creator info comes from ProjectPamphletMainCellPropertiesFragment
}
```

#### Fetch Project Detail by Slug
```graphql
query FetchProjectBySlug($slug: String!) {
  project(slug: $slug) {
    ...ProjectFragment   # full detail including rewards, creator, story
    backing { id }
  }
}
```

#### Fetch All Root Categories
```graphql
query FetchRootCategories {
  rootCategories {
    id
    name
    totalProjectCount
    subcategories {
      nodes { id name parentId totalProjectCount }
      totalCount
    }
  }
}
```

**Full category list** (from `FetchRootCategories`):
Art, Comics, Crafts, Dance, Design, Fashion, Film & Video, Food, Games, Journalism, Music, Photography, Publishing, Technology, Theater

### Fallback: REST Discover Endpoint

No auth required. Used as backup if GraphQL session acquisition fails.

```
GET https://www.kickstarter.com/discover/advanced.json
  ?category_id=16
  &sort=newest          # magic | newest | end_date | most_backed | most_funded
  &page=1
  &per_page=20
```

Returns flat JSON with same core fields. Less filter capability than GraphQL but zero auth overhead.

### Chosen Strategy

| Use Case | Endpoint |
|---|---|
| Nightly category crawl | REST `/discover/advanced.json` (no auth, stable) |
| Search with keyword | GraphQL `/graph` (richer filters, cursor pagination) |
| Campaign detail | GraphQL `/graph` by slug |
| Category list | GraphQL `FetchRootCategories` (cached daily) |

The nightly crawl uses REST to avoid session management complexity. GraphQL is used on-demand from the iOS app via the backend proxy.

---

## Tech Stack

| Layer | Technology |
|---|---|
| iOS | SwiftUI, SwiftData, iOS 17+ |
| Backend | Go 1.24, Gin framework |
| Database | Postgres (AWS RDS, new DB or new schema in existing RDS) |
| Infrastructure | New ECS Fargate service + task definition |
| Notifications | APNs direct HTTP/2 (no Firebase) |
| Scheduling | `robfig/cron` v3 embedded in backend process |
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
- Segmented control: **Trending** (`MAGIC`) / **New** (`NEWEST`) / **Ending Soon** (`END_DATE`)
- Horizontal scrollable category filter chips (loaded from `/api/categories`)
- `CampaignListView` → cursor-paginated list of `CampaignRowView`
- Pull-to-refresh

#### CampaignRowView
- Thumbnail (cached), title, creator name
- Funding progress bar + `%` funded + days-left label
- Backer count
- Watchlist heart button (tap to toggle, SwiftData local)

#### CampaignDetailView
- Hero image, title, blurb
- Funding ring (% funded), goal amount, pledged amount, backer count, deadline countdown
- Reward tiers list (if returned from GraphQL ProjectFragment)
- "Back this project" → `openURL` to `campaign.url`
- Share sheet
- Add/Remove Watchlist button

#### WatchlistView
- Saved campaigns from SwiftData, sorted by days left
- Status badge: `Live` / `Funded` / `Ended` / `Failed`
- Swipe-to-remove
- Empty state CTA → Discover

#### AlertsView
- List of keyword alerts
- Tap alert → show matched campaigns from last digest run
- "New Alert" sheet: keyword (required), category (optional), min-% funded (optional)
- Enable/disable toggle per alert

#### SettingsView
- Notification opt-in/out
- About / app version

---

## Data Models

### SwiftData (local — watchlist + cache)

```swift
@Model
class Campaign {
    @Attribute(.unique) var pid: String   // Kickstarter numeric project ID
    var name: String
    var blurb: String
    var photoURL: String
    var goalAmount: Double
    var goalCurrency: String
    var pledgedAmount: Double
    var deadline: Date
    var state: String           // "live" | "successful" | "failed" | "canceled"
    var categoryName: String
    var categoryID: String      // Kickstarter category ID string
    var projectURL: String
    var creatorName: String
    var percentFunded: Double
    var isWatched: Bool
    var lastFetchedAt: Date
}

@Model
class WatchlistAlert {
    @Attribute(.unique) var id: String    // client-generated UUID
    var keyword: String
    var categoryID: String?               // nil = all categories
    var minPercentFunded: Double          // 0 = no filter
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

### Postgres (backend — campaigns cache + alert routing)

```sql
campaigns (
    pid              TEXT PRIMARY KEY,   -- Kickstarter numeric project ID
    name             TEXT NOT NULL,
    blurb            TEXT,
    photo_url        TEXT,
    goal_amount      NUMERIC,
    goal_currency    TEXT,
    pledged_amount   NUMERIC,
    deadline         TIMESTAMPTZ,
    state            TEXT,
    category_id      TEXT,
    category_name    TEXT,
    project_url      TEXT,
    creator_name     TEXT,
    percent_funded   NUMERIC,
    slug             TEXT,               -- for GraphQL detail fetch
    first_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
)

categories (
    id     TEXT PRIMARY KEY,
    name   TEXT NOT NULL,
    parent_id TEXT
)

devices (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_token TEXT UNIQUE NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT now()
)

alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID REFERENCES devices(id) ON DELETE CASCADE,
    keyword         TEXT NOT NULL,
    category_id     TEXT,               -- NULL = all categories
    min_percent     NUMERIC DEFAULT 0,
    is_enabled      BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT now(),
    last_matched_at TIMESTAMPTZ
)

alert_matches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id    UUID REFERENCES alerts(id) ON DELETE CASCADE,
    campaign_pid TEXT REFERENCES campaigns(pid),
    matched_at  TIMESTAMPTZ DEFAULT now()
)
```

---

## Backend REST API

### Campaigns

```
GET /api/campaigns
  ?sort=trending|newest|ending    default: trending
  ?category_id=<string>           optional (Kickstarter category ID)
  ?cursor=<string>                optional (opaque cursor for pagination)
  ?limit=<int>                    default: 20, max: 50
  → { campaigns: [...], next_cursor: string|null, total: int }

GET /api/campaigns/search
  ?q=<string>                     required
  ?category_id=<string>           optional
  ?cursor=<string>                optional
  → { campaigns: [...], next_cursor: string|null }

GET /api/campaigns/:pid
  → Campaign object (detail, fetched via GraphQL by slug if needed)

GET /api/categories
  → [{ id: string, name: string, parent_id: string|null }]
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
  body: { is_enabled?, keyword?, category_id?, min_percent? }
  → Alert object

DELETE /api/alerts/:id

GET /api/alerts/:id/matches
  ?since=<ISO8601>                optional, defaults to last 24h
  → [Campaign]
```

---

## Nightly Cron Job

**Schedule**: 02:00 UTC daily

**Steps**:
1. For each root category (~15), call REST `/discover/advanced.json?sort=newest&page=1..10` — captures ~200 newest campaigns per category
2. Upsert into `campaigns`; set `first_seen_at` only on INSERT
3. Query campaigns where `first_seen_at > now() - interval '25 hours'` (new since last run)
4. For each enabled `alert`:
   - SQL: `WHERE name ILIKE '%keyword%' AND (category_id = alert.category_id OR alert.category_id IS NULL) AND percent_funded >= alert.min_percent`
   - If matches found: insert `alert_matches`, send APNs push
5. Update `alerts.last_matched_at`

**APNs push payload**:
```json
{
  "aps": {
    "alert": {
      "title": "5 new \"mechanical keyboard\" campaigns",
      "body": "Tap to see today's matches in KickWatch"
    },
    "badge": 1,
    "sound": "default"
  },
  "alert_id": "<uuid>",
  "match_count": 5
}
```

---

## Go Backend Structure

```
backend/
├── cmd/api/main.go
└── internal/
    ├── config/         env vars, APNs cert path, DB URL
    ├── handler/
    │   ├── campaigns.go
    │   ├── alerts.go
    │   └── devices.go
    ├── model/          GORM models matching schema above
    ├── service/
    │   ├── kickstarter_rest.go    REST discover client (no auth)
    │   ├── kickstarter_graph.go   GraphQL client (session bootstrap)
    │   ├── cron.go                nightly crawl + alert matching
    │   └── apns.go                APNs HTTP/2 sender
    └── middleware/     request logging, error handling
```

**Key env vars**:
```
DATABASE_URL
APNS_KEY_ID
APNS_TEAM_ID
APNS_BUNDLE_ID          com.yourname.kickwatch
APNS_KEY_PATH           /secrets/apns.p8
APNS_ENV                production|sandbox
PORT                    8080
```

---

## Project Layout

```
KickWatch/
├── spec.md
├── README.md
├── .github/workflows/
│   ├── deploy-backend.yml
│   └── test-backend.yml
├── ios/
│   ├── project.yml
│   ├── Package.swift
│   └── KickWatch/Sources/
│       ├── App/
│       ├── Models/
│       ├── Views/
│       ├── ViewModels/
│       └── Services/
│           ├── APIClient.swift         our backend
│           └── NotificationService.swift
└── backend/
    ├── cmd/api/
    └── internal/
```

---

## Development Commands

```bash
# Backend
cd backend && go run ./cmd/api
cd backend && go test ./...

# iOS
cd ios && xcodegen generate
xcodebuild -project ios/KickWatch.xcodeproj -scheme KickWatch build
```

---

## MVP Scope (v1)

- [ ] Backend: REST nightly crawler + Postgres upsert
- [ ] Backend: GraphQL proxy for search + detail
- [ ] Backend: Alert matching + APNs push
- [ ] iOS: Discover feed (Trending/New/Ending) + category filter
- [ ] iOS: Campaign detail + local watchlist (SwiftData)
- [ ] iOS: Keyword alerts CRUD + digest view
- [ ] iOS: Search

**Out of v1**: cross-device sync, funding-threshold crossed alerts, creator follow, campaign history chart, subscription gating, Indiegogo support.

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| KS blocks GraphQL from datacenter IPs | Use REST-only fallback; rotate User-Agent; add residential proxy if needed |
| CSRF token / session expires | Session refresh logic on every cron run; retry on 403 |
| KS GraphQL schema changes | Schema is public in kickstarter/ios-oss; monitor their repo for changes |
| APNs invalid device tokens | Remove device + cascade-delete alerts on 410 response from APNs |
| High crawl volume triggers rate limit | Stagger requests with 500ms delay; limit to 10 pages/category |
| Kickstarter ToS | Personal/non-commercial use; read-only public data; no credential reuse |
