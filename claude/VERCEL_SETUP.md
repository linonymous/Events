# Vercel Deployment Setup

This document explains how to deploy the Events app to Vercel.

## Prerequisites

- Vercel account (free tier works)
- GitHub repository with the code
- PostgreSQL database (Supabase free tier recommended)
- VAPID keys for push notifications

## Quick Start

### 1. Generate VAPID Keys

```bash
npx web-push generate-vapid-keys
```

This outputs:
```
Public Key: BM...
Private Key: ...
```

Save both keys - you'll need them for Vercel environment variables.

### 2. Set Up Database (Supabase)

1. Create account at [supabase.com](https://supabase.com)
2. Create new project
3. Go to Settings > Database
4. Copy the connection string (URI format):
   ```
   postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres
   ```

### 3. Deploy to Vercel

#### Option A: Vercel Dashboard

1. Go to [vercel.com](https://vercel.com)
2. Click "Add New" > "Project"
3. Import your GitHub repository
4. Configure:
   - Framework Preset: Other
   - Root Directory: ./
5. Add Environment Variables (see below)
6. Click "Deploy"

#### Option B: Vercel CLI

```bash
npm i -g vercel
vercel login
vercel
```

### 4. Environment Variables

Add these in Vercel Dashboard > Settings > Environment Variables:

| Variable | Value | Description |
|----------|-------|-------------|
| `DATABASE_URL` | `postgresql://...` | Supabase connection string |
| `VAPID_PUBLIC_KEY` | `BM...` | Generated public key |
| `VAPID_PRIVATE_KEY` | `...` | Generated private key |
| `VAPID_EMAIL` | `mailto:you@example.com` | Contact email for push service |
| `CRON_SECRET` | `random-string-here` | Secret for cron authentication |

## Configuration Files

### vercel.json

```json
{
  "$schema": "https://openapi.vercel.sh/vercel.json",
  "version": 2,
  "builds": [
    {
      "src": "cmd/app/main.go",
      "use": "@vercel/go"
    }
  ],
  "routes": [
    {
      "src": "/(.*)",
      "dest": "cmd/app/main.go"
    }
  ],
  "crons": [
    {
      "path": "/events/api/cron/send-reminders",
      "schedule": "0 9 * * *"
    }
  ],
  "env": {
    "DATABASE_URL": "@database-url",
    "VAPID_PUBLIC_KEY": "@vapid-public-key",
    "VAPID_PRIVATE_KEY": "@vapid-private-key",
    "VAPID_EMAIL": "@vapid-email",
    "CRON_SECRET": "@cron-secret"
  }
}
```

### Build Configuration

- **Source**: `cmd/app/main.go`
- **Builder**: `@vercel/go` (Go 1.x serverless)
- **Output**: Single serverless function handling all routes

## Cron Jobs

The app uses Vercel Cron to send daily reminders:

| Schedule | Path | Description |
|----------|------|-------------|
| `0 9 * * *` | `/events/api/cron/send-reminders` | Daily at 9 AM UTC |

### Cron Security

The `CRON_SECRET` environment variable protects the cron endpoint:

```go
auth := r.Header.Get("Authorization")
if auth != "Bearer "+cronSecret {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

Vercel automatically adds the correct Authorization header when triggering crons.

### Adjusting Schedule

Modify the `schedule` in `vercel.json`:
- `0 9 * * *` - 9 AM UTC daily
- `0 7 * * *` - 7 AM UTC daily
- `0 9,18 * * *` - 9 AM and 6 PM UTC daily

## Project Structure for Vercel

```
Events/
├── cmd/
│   └── app/
│       ├── main.go          # Entry point (Vercel builds this)
│       ├── static/          # Embedded static files
│       │   ├── manifest.json
│       │   ├── sw.js
│       │   └── icons...
│       └── templates/       # Embedded HTML templates
├── internal/
│   ├── handlers/           # HTTP handlers
│   ├── models/             # Data models
│   └── storage/            # Database layer
├── go.mod
├── go.sum
└── vercel.json             # Vercel configuration
```

## Deployment Workflow

### Automatic Deployments

1. Push to `main` branch triggers production deployment
2. Push to other branches creates preview deployments
3. Pull requests get preview URLs for testing

### Manual Deployment

```bash
vercel --prod  # Deploy to production
vercel         # Deploy preview
```

## Monitoring

### Function Logs

1. Vercel Dashboard > Project > Logs
2. Filter by function or time range
3. View cron execution logs

### Cron Monitoring

1. Vercel Dashboard > Project > Cron Jobs
2. View execution history
3. Check success/failure status

## Troubleshooting

### Build Fails

```
Error: unable to find module
```
- Ensure `go.mod` exists at repository root
- Check Go version compatibility (1.21+)
- Run `go mod tidy` locally and commit

### Database Connection

```
Error: dial tcp: connection refused
```
- Verify DATABASE_URL format
- Check Supabase is not paused (free tier pauses after inactivity)
- Ensure IP is not blocked (Supabase settings)

### Cron Not Running

- Check Vercel cron logs
- Verify schedule syntax
- Ensure CRON_SECRET matches environment variable
- Free tier: crons limited to daily execution

### Static Files 404

```
Error: 404 on /events/static/manifest.json
```
- Files must be in `cmd/app/static/` directory
- Check `//go:embed static/*` directive in main.go
- Verify route configuration in main.go

## Cost Optimization

### Vercel Free Tier Limits

- 100 GB bandwidth/month
- 100 GB-hours serverless execution
- Cron jobs: 1 per day max on free tier

### Supabase Free Tier Limits

- 500 MB database storage
- 2 GB bandwidth/month
- Pauses after 1 week inactivity

### Staying Free

- Events app uses minimal resources
- Single daily cron stays within limits
- Small user base fits free tiers
