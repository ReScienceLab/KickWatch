# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Backend: Go/Gin API server with Kickstarter REST + GraphQL clients
- Backend: GORM models for campaigns, categories, devices, alerts, alert_matches
- Backend: Nightly cron crawler (02:00 UTC) across all Kickstarter root categories
- Backend: Alert matching engine with APNs push notification delivery
- Backend: REST API — campaigns, search, categories, devices, alerts CRUD
- Backend: Dockerfile and `.env.example`
- iOS: SwiftData models — Campaign, WatchlistAlert, RecentSearch
- iOS: APIClient (actor), KeychainHelper, NotificationService, ImageCache
- iOS: DiscoverView with sort segmented control and horizontal category chips
- iOS: CampaignRowView with funding progress bar, watchlist heart toggle
- iOS: CampaignDetailView with funding ring, stat boxes, share sheet, back link
- iOS: WatchlistView with SwiftData query, swipe-to-remove, status badges
- iOS: AlertsView with CRUD, enable/disable toggle, alert matches view
- iOS: SearchView with cursor pagination
- iOS: SettingsView with notification opt-in and app version
- CI: GitHub Actions workflows for backend tests and ECS deployment
