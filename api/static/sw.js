// Events App Service Worker
const CACHE_NAME = 'events-app-v1';

// Install event
self.addEventListener('install', (event) => {
  self.skipWaiting();
});

// Activate event
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((cacheName) => cacheName !== CACHE_NAME)
          .map((cacheName) => caches.delete(cacheName))
      );
    })
  );
  self.clients.claim();
});

// Push event - handle incoming push notifications
self.addEventListener('push', (event) => {
  if (!event.data) return;

  let data;
  try {
    data = event.data.json();
  } catch {
    data = {
      title: 'Events',
      body: event.data.text(),
    };
  }

  const options = {
    body: data.body || 'You have an event reminder',
    icon: '/events/static/icon-192.svg',
    badge: '/events/static/icon-192.svg',
    vibrate: [100, 50, 100],
    tag: data.tag || 'event-notification',
    renotify: true,
    requireInteraction: true,
    data: {
      url: data.url || '/events/',
    },
  };

  event.waitUntil(
    self.registration.showNotification(data.title || 'Event Reminder', options)
  );
});

// Notification click event
self.addEventListener('notificationclick', (event) => {
  event.notification.close();

  const urlToOpen = event.notification.data?.url || '/events/';

  // Check if this is an external URL (like WhatsApp)
  const isExternalUrl = urlToOpen.startsWith('http://') || urlToOpen.startsWith('https://');

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      // For external URLs (like WhatsApp), always open in a new window
      if (isExternalUrl) {
        if (clients.openWindow) {
          return clients.openWindow(urlToOpen);
        }
        return;
      }

      // For internal URLs, try to focus and navigate an existing window
      for (const client of clientList) {
        if (client.url.includes('/events') && 'focus' in client) {
          // Navigate to the URL and focus
          client.navigate(urlToOpen);
          return client.focus();
        }
      }

      // No existing window, open a new one
      if (clients.openWindow) {
        return clients.openWindow(urlToOpen);
      }
    })
  );
});

// Push subscription change event
self.addEventListener('pushsubscriptionchange', (event) => {
  event.waitUntil(
    self.registration.pushManager.subscribe(event.oldSubscription.options)
      .then((subscription) => {
        return fetch('/events/api/push/subscribe', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(subscription.toJSON()),
        });
      })
  );
});
