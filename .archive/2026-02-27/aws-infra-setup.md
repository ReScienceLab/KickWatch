---
date: 2026-02-27
title: KickWatch AWS Infrastructure Setup
category: infrastructure
tags: [aws, ecs, ecr, rds, iam, oidc, secrets-manager, github-actions]
related: [2026-02-27/mvp-implementation.md]
---

# KickWatch AWS Infrastructure Setup

## Account & Region
- Account ID: `739654145647`
- Region: `us-east-2`
- IAM user: `snapaction-admin` (shared with SnapAction)

## Resources Created

### ECR Repositories
- `kickwatch-api` — prod
- `kickwatch-api-dev` — dev

### IAM Roles
- `kickwatch-deploy-role` — GitHub Actions OIDC deploy role
  - Trust: `repo:ReScienceLab/KickWatch:*`
  - Policy: `kickwatch-deploy-policy` (ECR push, ECS deploy, iam:PassRole, secrets read)
- `kickwatch-task-role` — ECS container role (no extra permissions in v1)
- `ecsTaskExecutionRole` — existing shared role, added `kickwatch-secrets-access` inline policy

### OIDC Provider
- Reused existing: `arn:aws:iam::739654145647:oidc-provider/token.actions.githubusercontent.com`

### ECS Clusters
- `kickwatch-cluster` (prod, containerInsights=enabled)
- `kickwatch-cluster-dev` (dev, containerInsights=enabled)

### ECS Services
- `kickwatch-cluster-dev/kickwatch-api-dev-service` (desired=0, task def :2)
- `kickwatch-cluster/kickwatch-api-service` (desired=0, task def :1)

### ECS Task Definitions
- `kickwatch-api-dev:2` — dev, GIN_MODE=debug, APNS_ENV=sandbox
- `kickwatch-api:1` — prod, GIN_MODE=release, APNS_ENV=production
- Networking: awsvpc, subnets: `subnet-03c3f58cea867dac7`, `subnet-0eaf3dc3284bf18d9`, `subnet-0d6addfa05326637e`
- SG: `sg-09a8956d7d1e3274e` (default VPC SG)
- assignPublicIp: ENABLED

### RDS
- `kickwatch-db-dev` — postgres 16.8, db.t3.micro, 20GB
  - Endpoint: `kickwatch-db-dev.c164w44w2oh3.us-east-2.rds.amazonaws.com`
  - DB name: `kickwatch_dev`
  - User: `kickwatch`
  - Password: stored in `/tmp/kw_dbpw_dev.txt` locally → set in Secrets Manager
  - Publicly accessible: YES (needed for ECS task; protected by SG)
  - SG: `sg-0f27ad8fd043ce974` (snapaction-rds-sg, allows 5432 from default SG)
- Prod RDS: **not yet created** — create when ready to deploy prod

### CloudWatch Log Groups
- `/ecs/kickwatch-api` (30 day retention)
- `/ecs/kickwatch-api-dev` (14 day retention)

### Secrets Manager
All in `us-east-2`:
| Secret | Value |
|--------|-------|
| `kickwatch-dev/database-url` | Real URL pointing to kickwatch-db-dev |
| `kickwatch-dev/apns-key-id` | `FILL_IN_APNS_KEY_ID` ← needs real value |
| `kickwatch-dev/apns-team-id` | `FILL_IN_APNS_TEAM_ID` ← needs real value |
| `kickwatch-dev/apns-bundle-id` | `com.kickwatch.app` |
| `kickwatch/database-url` | `PLACEHOLDER` ← fill when prod RDS created |
| `kickwatch/apns-key-id` | `FILL_IN_APNS_KEY_ID` ← needs real value |
| `kickwatch/apns-team-id` | `FILL_IN_APNS_TEAM_ID` ← needs real value |
| `kickwatch/apns-bundle-id` | `com.kickwatch.app` |

### GitHub Secrets (ReScienceLab/KickWatch)
- `AWS_DEPLOY_ROLE_ARN` = `arn:aws:iam::739654145647:role/kickwatch-deploy-role`
- `AWS_ACCOUNT_ID` = `739654145647`

## GitHub Actions Workflow
- `test-backend.yml` — triggers on `backend/**` changes, go vet + test + build
- `deploy-backend.yml` — OIDC auth, `develop`→dev deploy, `main`→prod deploy, Dockerfile.ci
- PR #2 open: `feature/ci-oidc → develop`

## What Needs Manual Action
1. Fill APNs secrets: `APNS_KEY_ID`, `APNS_TEAM_ID` — from Apple Developer Portal
2. Upload `.p8` APNs key file — needs ECS secrets mount or embed in Secrets Manager
3. Create prod RDS (`kickwatch-db`) when ready to go to production
4. Set `DEVELOPMENT_TEAM` in `ios/project.yml` for Xcode builds
5. Set ECS service desired_count to 1 once first image is pushed to ECR

## Gotchas
- VPN (Quantumult X) causes RDS DNS to resolve to 198.18.x.x — same issue as SnapAction
  - Can't connect to RDS via psql from local machine when VPN active
  - Fix: disable VPN, OR use ECS Exec from a running container
- `kickwatch-db-dev` shares SG `sg-0f27ad8fd043ce974` with SnapAction RDS
  - SG already had port 5432 intra-SG rule (default SG → default SG)
- snapaction-db-dev was temporarily set to `publicly-accessible` during setup → reverted
- ECS services start at `desired_count=0` — CI deploy sets count to 1 on first deploy

## Key Commands
```bash
# Update a secret
aws secretsmanager put-secret-value \
  --secret-id kickwatch-dev/apns-key-id \
  --region us-east-2 --secret-string "YOUR_KEY_ID"

# Tail ECS logs
aws logs tail /ecs/kickwatch-api-dev --region us-east-2 --follow

# Force new deployment
aws ecs update-service --cluster kickwatch-cluster-dev \
  --service kickwatch-api-dev-service --force-new-deployment --region us-east-2
```
