# Events

A web application for tracking recurring events like birthdays and anniversaries with push notification reminders. Built with Go and deployable to Vercel with Supabase as the database.

## Features

- Track birthdays, anniversaries, and other recurring events
- Push notifications for event reminders (today and tomorrow)
- Progressive Web App (PWA) support
- User authentication
- Mobile-responsive design
- Optional WhatsApp integration for quick messaging

## Tech Stack

- **Backend**: Go 1.24+
- **Frontend**: HTML templates with HTMX
- **Database**: PostgreSQL (Supabase)
- **Hosting**: Vercel (Serverless)
- **Push Notifications**: Web Push (VAPID)

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `SESSION_SECRET` | Yes | Secret key for session cookies (min 32 chars recommended) |
| `VAPID_PUBLIC_KEY` | Yes* | VAPID public key for web push notifications |
| `VAPID_PRIVATE_KEY` | Yes* | VAPID private key for web push notifications |
| `VAPID_EMAIL` | Yes* | Contact email for VAPID (e.g., `mailto:you@example.com`) |
| `CRON_SECRET` | Yes* | Secret for authenticating cron job requests |
| `PORT` | No | Server port for local development (default: 8100) |

*Required for push notification functionality

### Generating Environment Variables

**DATABASE_URL**

- **Local Docker**: Use the connection string from your `docker-compose.yml`
- **Supabase**: Go to Project Settings > Database > Connection string (use Transaction pooler)

**SESSION_SECRET**

Generate a secure random string (32+ characters):
```bash
# Using openssl
openssl rand -base64 32

# Using Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

**VAPID Keys (for Push Notifications)**

Generate a VAPID key pair:
```bash
# Using Node.js
npx web-push generate-vapid-keys
```

This outputs both `VAPID_PUBLIC_KEY` and `VAPID_PRIVATE_KEY`. Set `VAPID_EMAIL` to your contact email prefixed with `mailto:`.

Alternatively, use an online generator: [vapidkeys.com](https://vapidkeys.com)

**CRON_SECRET**

Generate a secure random string:
```bash
openssl rand -hex 32
```

### Running Locally

Create a `.env` file with the variables above, then:
```bash
# Load env and run
export $(cat .env | xargs) && go run cmd/app/main.go

# Or with Docker Compose
docker-compose up --build
```

---

## Supabase Setup

1. **Create a Supabase Project**
   - Go to [supabase.com](https://supabase.com) and create a new project
   - Wait for the project to be provisioned

2. **Get Database Connection String**
   - Go to **Project Settings** > **Database**
   - Copy the **Connection string** (URI format)
   - It should look like: `postgresql://postgres.[project-ref]:[password]@aws-0-[region].pooler.supabase.com:6543/postgres`

3. **Tables are Auto-Created**
   - The application automatically creates the required tables on first run:
     - `users` - User accounts
     - `events` - Event records
     - `push_subscriptions` - Push notification subscriptions

---

## Vercel Deployment

### 1. Fork/Clone the Repository

```bash
git clone https://github.com/IAmSurajBobade/Events.git
cd Events
```

### 2. Install Vercel CLI

```bash
npm install -g vercel
```

### 3. Deploy to Vercel

```bash
vercel
```

### 4. Configure Environment Variables

In the Vercel dashboard or via CLI:

```bash
vercel env add DATABASE_URL
vercel env add SESSION_SECRET
vercel env add VAPID_PUBLIC_KEY
vercel env add VAPID_PRIVATE_KEY
vercel env add VAPID_EMAIL
vercel env add CRON_SECRET
```

### 5. Generate VAPID Keys

You can generate VAPID keys using Node.js:

```bash
npx web-push generate-vapid-keys
```

Or use an online generator like [vapidkeys.com](https://vapidkeys.com)

### 6. Cron Job Configuration

The `vercel.json` includes a cron job for sending daily reminders:

```json
{
  "crons": [
    {
      "path": "/events/api/cron/send-reminders",
      "schedule": "30 0 * * *"
    }
  ]
}
```

This runs daily at **00:30 UTC (6:00 AM IST)**. Adjust the schedule as needed.

**Important**: Vercel Cron requires the `CRON_SECRET` to be set. The cron request must include the header:
```
Authorization: Bearer <CRON_SECRET>
```

Vercel automatically adds this header when calling cron endpoints.

---

## Local Development

### Option 1: Docker Compose (Recommended)

1. **Start the services**
   ```bash
   docker-compose up --build
   ```

2. **Access the app**
   - App: http://localhost:8100/events/
   - Database: localhost:5432

3. **Set environment variables** (create `.env` file or export):
   ```bash
   export SESSION_SECRET="your-session-secret-here"
   export VAPID_PUBLIC_KEY="your-vapid-public-key"
   export VAPID_PRIVATE_KEY="your-vapid-private-key"
   export VAPID_EMAIL="mailto:your@email.com"
   export CRON_SECRET="your-cron-secret"
   ```

### Option 2: Manual Setup

1. **Install Go 1.24+**
   ```bash
   # macOS
   brew install go

   # Or download from https://golang.org/dl/
   ```

2. **Start PostgreSQL**
   ```bash
   # Using Docker
   docker run -d --name events-db \
     -e POSTGRES_USER=user \
     -e POSTGRES_PASSWORD=password \
     -e POSTGRES_DB=events \
     -p 5432:5432 \
     postgres:18-alpine
   ```

3. **Set environment variables**
   ```bash
   export DATABASE_URL="postgres://user:password@localhost:5432/events?sslmode=disable"
   export SESSION_SECRET="your-session-secret-here"
   export PORT="8100"
   ```

4. **Run the application**
   ```bash
   go run cmd/app/main.go
   ```

5. **Access the app**
   - http://localhost:8100/events/

---

## Project Structure

```
Events/
├── api/
│   ├── index.go          # Vercel serverless entry point
│   ├── templates/         # HTML templates
│   └── static/            # PWA assets (manifest, icons, service worker)
├── cmd/
│   └── app/
│       └── main.go        # Local development entry point
├── pkg/
│   ├── handlers/          # HTTP handlers
│   ├── models/            # Data models
│   └── storage/           # Database layer
├── vercel.json            # Vercel configuration
├── docker-compose.yml     # Docker Compose for local dev
├── Dockerfile             # Docker build configuration
└── go.mod                 # Go module definition
```

---

## API Endpoints

### Public Routes
| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/events/login` | User login |
| GET/POST | `/events/register` | User registration |
| GET | `/events/logout` | User logout |
| GET | `/events/api/vapid-key` | Get VAPID public key |
| GET | `/events/api/cron/send-reminders` | Trigger reminder notifications (requires auth) |

### Protected Routes (requires login)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/events/` | Home page with event lists |
| GET | `/events/lists/{list_name}` | View events in a list |
| GET | `/events/lists/{list_name}/edit/{id}` | Edit an event |
| GET | `/events/lists/{list_name}/delete/{id}` | Delete an event |
| POST | `/events/save` | Save/update an event |
| POST | `/events/api/push/subscribe` | Subscribe to push notifications |
| POST | `/events/api/push/unsubscribe` | Unsubscribe from push |
| GET | `/events/api/push/status` | Get push subscription status |
| POST | `/events/api/push/test` | Send test notification |

---

## Known Limitations

### Timezone

| Limitation | Details |
|------------|---------|
| **Hardcoded timezone** | The application uses `Asia/Kolkata` (IST, UTC+5:30) timezone exclusively |
| **No per-user timezone** | All users share the same timezone setting |
| **Fixed notification time** | Cron runs at 6:00 AM IST for all users |
| **Database queries** | SQL date comparisons use hardcoded `AT TIME ZONE 'Asia/Kolkata'` |

**Files affected:**
- `api/index.go` - `time.LoadLocation("Asia/Kolkata")`
- `cmd/app/main.go` - `time.LoadLocation("Asia/Kolkata")`
- `pkg/storage/postgres.go` - SQL queries with `AT TIME ZONE 'Asia/Kolkata'`

### Authentication

| Limitation | Details |
|------------|---------|
| **Basic auth only** | Username/password authentication only |
| **No OAuth** | No Google, GitHub, or social login support |
| **No password reset** | Users cannot reset forgotten passwords |
| **No email verification** | Accounts are active immediately upon registration |

### Push Notifications

| Limitation | Details |
|------------|---------|
| **HTTPS required** | Push notifications only work over HTTPS (browser requirement) |
| **Browser support** | Requires browsers with Web Push API and Service Worker support |
| **No email fallback** | Only push notifications, no email reminders |
| **Reminder window** | Only notifies for events happening today or tomorrow (no advance reminders like 1 week before) |

### Database

| Limitation | Details |
|------------|---------|
| **PostgreSQL only** | No support for SQLite, MySQL, or other databases |
| **No migrations** | Schema changes require manual intervention |

### General

| Limitation | Details |
|------------|---------|
| **No data export** | Cannot export events to CSV/iCal |
| **No calendar sync** | No Google Calendar or iCal integration |
| **No recurring patterns** | Only yearly recurring events (birthdays/anniversaries), no weekly/monthly patterns |
| **Single list type** | Events are organized by simple text-based lists |

---

## Troubleshooting

### Database Connection Issues
- Ensure `DATABASE_URL` includes `sslmode=require` for Supabase
- Check that the database password doesn't contain special characters that need URL encoding

### Push Notifications Not Working
- Verify all VAPID environment variables are set
- Check browser console for service worker errors
- Ensure the site is served over HTTPS (required for push)

### Vercel Build Errors
- Ensure Go version in `go.mod` is supported by Vercel
- Check that all imports use `github.com/IAmSurajBobade/Events` path

---

## Manual Deployment (alwaysdata)

``` bash
# build
GOOS=linux GOARCH=amd64 go build -o main ./cmd/app/main.go
# copy templates folder
scp -r ./cmd/app/templates <username>@ssh-<username>.alwaysdata.net:./app/templates/
# copy to alwaysdata
scp ./main <username>@ssh-<username>.alwaysdata.net:./app/
# restart the app
```

---

## License

MIT
