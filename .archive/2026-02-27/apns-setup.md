---
date: 2026-02-27
title: APNs Key Setup and CI/CD Integration
category: infrastructure
tags: [apns, ios, push-notifications, secrets-manager, ecs, github-actions]
related: [2026-02-27/aws-infra-setup.md]
---

# APNs Key Setup and CI/CD Integration

## Apple Developer Portal

- **Key Name**: KickWatch APNs
- **Key ID**: `GUFRSCY8ZV`
- **Team ID**: `7Q28CBP3S5` (same as SnapAction)
- **Bundle ID**: `com.rescience.kickwatch`
- **Environment**: Sandbox & Production (covers both dev and prod with one key)
- **Key Restriction**: Team Scoped (All Topics)
- **File**: `AuthKey_GUFRSCY8ZV.p8` — downloaded to `/Users/yilin/Downloads/`

## Secrets Manager (us-east-2)

All 4 APNs secrets set for both dev and prod prefixes:

| Secret | Value |
|--------|-------|
| `kickwatch-dev/apns-key-id` | `GUFRSCY8ZV` |
| `kickwatch-dev/apns-team-id` | `7Q28CBP3S5` |
| `kickwatch-dev/apns-bundle-id` | `com.rescience.kickwatch` |
| `kickwatch-dev/apns-key` | Full `.p8` PEM content |
| `kickwatch/apns-key-id` | `GUFRSCY8ZV` |
| `kickwatch/apns-team-id` | `7Q28CBP3S5` |
| `kickwatch/apns-bundle-id` | `com.rescience.kickwatch` |
| `kickwatch/apns-key` | Full `.p8` PEM content |

## Commands Used

```bash
KEY_ID="GUFRSCY8ZV"
REGION=us-east-2

# Key ID
aws secretsmanager put-secret-value \
  --secret-id kickwatch-dev/apns-key-id --region $REGION --secret-string "$KEY_ID"

# .p8 content
aws secretsmanager put-secret-value \
  --secret-id kickwatch-dev/apns-key --region $REGION \
  --secret-string "$(cat ~/Downloads/AuthKey_GUFRSCY8ZV.p8)"
```

## Backend Change: File Path → Env Var

`internal/service/apns.go` updated to read key from `APNS_KEY` env var first, falling back to `APNS_KEY_PATH` file. Avoids need to mount `.p8` file into ECS container.

`internal/config/config.go` added `APNSKey string` field reading `APNS_KEY`.

## CI Workflow Change

`deploy-backend.yml` — removed `APNS_KEY_PATH` env var, added `APNS_KEY` secret injected from Secrets Manager ARN.

## iOS Changes

- `project.yml`: `DEVELOPMENT_TEAM: 7Q28CBP3S5`, `PRODUCT_BUNDLE_IDENTIFIER: com.rescience.kickwatch`
- `KickWatch.entitlements`: `aps-environment = development`

## Gotchas

- APNs key environment set to **Sandbox & Production** — one key works for both; do NOT create separate keys
- Bundle ID must match exactly what's registered in Apple Developer Portal
- `APNS_KEY` env var content is the raw PEM string including `-----BEGIN PRIVATE KEY-----` header/footer
- ECS task execution role needs `secretsmanager:GetSecretValue` for `kickwatch*` ARNs (already added)
