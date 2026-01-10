# Events API Documentation

Base URL: `https://events-iota-eight.vercel.app`

## Authentication

The API uses session-based authentication. After login, include the session cookie in subsequent requests.

---

## cURL Commands

### Public Endpoints

#### 1. Register
```bash
curl -X POST 'https://events-iota-eight.vercel.app/events/register' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'username=testuser&password=testpass'
```

#### 2. Login
```bash
curl -X POST 'https://events-iota-eight.vercel.app/events/login' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'username=testuser&password=testpass' \
  -c cookies.txt
```

#### 3. Logout
```bash
curl 'https://events-iota-eight.vercel.app/events/logout' \
  -b cookies.txt
```

#### 4. Get VAPID Public Key
```bash
curl 'https://events-iota-eight.vercel.app/events/api/vapid-key'
```

### Protected Endpoints (require session cookie)

#### 5. Get Home Page
```bash
curl 'https://events-iota-eight.vercel.app/events/' \
  -b cookies.txt
```

#### 6. Get Event List
```bash
curl 'https://events-iota-eight.vercel.app/events/lists/birthdays' \
  -b cookies.txt
```

#### 7. Get Edit Event Page
```bash
curl 'https://events-iota-eight.vercel.app/events/lists/birthdays/edit/1' \
  -b cookies.txt
```

#### 8. Delete Event
```bash
curl 'https://events-iota-eight.vercel.app/events/lists/birthdays/delete/1' \
  -b cookies.txt
```

#### 9. Save Event (Create)
```bash
curl -X POST 'https://events-iota-eight.vercel.app/events/save' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'title=John Birthday&event_date=1990-05-15&recurring=true&list_name=birthdays' \
  -b cookies.txt
```

#### 10. Save Event (Update)
```bash
curl -X POST 'https://events-iota-eight.vercel.app/events/save' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'id=1&title=John Birthday Updated&event_date=1990-05-15&recurring=true&list_name=birthdays' \
  -b cookies.txt
```

#### 11. Get Push Notification Status
```bash
curl 'https://events-iota-eight.vercel.app/events/api/push/status' \
  -b cookies.txt
```

#### 12. Subscribe to Push Notifications
```bash
curl -X POST 'https://events-iota-eight.vercel.app/events/api/push/subscribe' \
  -H 'Content-Type: application/json' \
  -d '{
    "endpoint": "https://fcm.googleapis.com/fcm/send/xxx",
    "keys": {
      "p256dh": "your-p256dh-key",
      "auth": "your-auth-key"
    }
  }' \
  -b cookies.txt
```

#### 13. Unsubscribe from Push Notifications
```bash
curl -X POST 'https://events-iota-eight.vercel.app/events/api/push/unsubscribe' \
  -H 'Content-Type: application/json' \
  -d '{"endpoint": "https://fcm.googleapis.com/fcm/send/xxx"}' \
  -b cookies.txt
```

#### 14. Test Push Notification
```bash
curl -X POST 'https://events-iota-eight.vercel.app/events/api/push/test' \
  -b cookies.txt
```

### Admin/Cron Endpoints (require CRON_SECRET)

#### 15. Send Reminders (Cron Job)
```bash
curl 'https://events-iota-eight.vercel.app/events/api/cron/send-reminders' \
  -H 'Authorization: Bearer YOUR_CRON_SECRET'
```

#### 16. Clear All Subscriptions for User
```bash
curl -X POST 'https://events-iota-eight.vercel.app/events/api/push/clear?user_id=1' \
  -H 'Authorization: Bearer YOUR_CRON_SECRET'
```
