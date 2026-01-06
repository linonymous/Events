# Push Notifications & PWA Implementation

This document explains the push notification and PWA (Progressive Web App) implementation added to the Events app.

## Overview

The Events app now supports:
- **PWA Installation**: Users can install the app on their home screen (iOS Safari, Android Chrome, desktop browsers)
- **Web Push Notifications**: Event reminders sent via browser push notifications
- **Recurring Events**: Support for yearly recurring events (birthdays/anniversaries) vs one-time events

## Architecture

### Components

1. **Service Worker** (`cmd/app/static/sw.js`)
   - Handles incoming push notifications
   - Manages notification click events
   - Runs in the background even when app is closed

2. **PWA Manifest** (`cmd/app/static/manifest.json`)
   - Defines app metadata for installation
   - Specifies icons, colors, and display mode

3. **Push Handlers** (`internal/handlers/push.go`)
   - `SubscribePushHandler`: Saves browser push subscription
   - `UnsubscribePushHandler`: Removes subscription
   - `GetPushStatusHandler`: Checks if user is subscribed
   - `TestPushHandler`: Sends test notification
   - `SendRemindersHandler`: Cron-triggered reminder sender
   - `VAPIDPublicKeyHandler`: Returns public key for subscription

4. **Database** (`internal/storage/postgres.go`)
   - `push_subscriptions` table stores user subscriptions
   - `events.recurring` column tracks event type

## How It Works

### Push Subscription Flow

```
1. User clicks "Enable Notifications"
2. Browser requests notification permission
3. Service worker generates push subscription using VAPID public key
4. Subscription sent to /events/api/push/subscribe
5. Server stores subscription in push_subscriptions table
```

### Reminder Flow (Daily Cron at 9 AM)

```
1. Vercel triggers /events/api/cron/send-reminders
2. Server fetches all events and subscriptions
3. For each event:
   - If recurring: Calculate next yearly occurrence
   - If one-time: Use actual event date
   - If today or tomorrow: Send notification to user's subscriptions
4. Notification delivered via webpush-go library
```

### Recurring vs One-Time Events

**Recurring Events** (default):
- For birthdays, anniversaries
- Reminders sent every year on the date
- Shows "X years" in notification
- Example: Date "1996-04-21" reminds on April 21st every year

**One-Time Events**:
- For appointments, deadlines
- Reminder sent only once
- No year calculation
- Example: Date "2024-12-25" reminds only on that specific date

## iOS Safari Setup

For iOS users to receive push notifications:

1. **Use Safari browser** (Chrome/Firefox on iOS don't support Web Push)
2. **Install to Home Screen**:
   - Open the app in Safari
   - Tap Share button (square with arrow)
   - Tap "Add to Home Screen"
   - Open app from home screen icon
3. **Enable notifications**:
   - Click "Enable Notifications" button
   - Allow when prompted
4. **Requirements**:
   - iOS 16.4 or later
   - App must be opened from home screen (not Safari tab)

## Environment Variables

Required for push notifications:

```bash
# VAPID keys (generate using: npx web-push generate-vapid-keys)
VAPID_PUBLIC_KEY=BM...   # Public key shared with browser
VAPID_PRIVATE_KEY=...    # Private key for signing (keep secret!)
VAPID_EMAIL=mailto:your@email.com

# Cron authentication (optional but recommended)
CRON_SECRET=your-random-secret
```

## API Endpoints

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/events/api/vapid-key` | Returns VAPID public key |
| GET | `/events/api/cron/send-reminders` | Trigger reminder check (cron) |

### Protected Endpoints (require auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/events/api/push/subscribe` | Subscribe to push |
| POST | `/events/api/push/unsubscribe` | Unsubscribe from push |
| GET | `/events/api/push/status` | Check subscription status |
| POST | `/events/api/push/test` | Send test notification |

## Database Schema

### push_subscriptions table

```sql
CREATE TABLE push_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT UNIQUE NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### events.recurring column

```sql
ALTER TABLE events ADD COLUMN recurring BOOLEAN DEFAULT true;
```

## Notification Payload Format

```json
{
    "title": "Event Reminder",
    "body": "John's Birthday is tomorrow! (28 years)",
    "url": "/events/lists/family",
    "tag": "event-123"
}
```

## Troubleshooting

### Notifications not showing on iOS
- Ensure iOS 16.4+
- Must be installed to home screen
- Must open from home screen icon, not Safari
- Check notification permissions in Settings > Events

### Subscription fails
- Check VAPID keys are correctly set
- Ensure HTTPS is used (required for service workers)
- Check browser console for errors

### Cron not triggering
- Verify Vercel cron is configured in vercel.json
- Check CRON_SECRET matches if set
- Review Vercel function logs

### Invalid subscription cleanup
- Server automatically removes subscriptions returning 410/404
- Happens when browser clears data or user unsubscribes elsewhere
