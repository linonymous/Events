package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/IAmSurajBobade/sb-dashboard/internal/models"
	webpush "github.com/SherClockHolmes/webpush-go"
)

// PushHandler handles push subscription management
func (c *Controller) SubscribePushHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserID(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input models.PushSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Endpoint == "" || input.Keys.P256dh == "" || input.Keys.Auth == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if err := c.Store.SavePushSubscription(userID, input.Endpoint, input.Keys.P256dh, input.Keys.Auth); err != nil {
		log.Printf("Error saving push subscription: %v", err)
		http.Error(w, "Failed to save subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "subscribed"})
}

func (c *Controller) UnsubscribePushHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserID(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := c.Store.DeletePushSubscription(input.Endpoint); err != nil {
		log.Printf("Error deleting push subscription: %v", err)
		http.Error(w, "Failed to delete subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unsubscribed"})
}

func (c *Controller) GetPushStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subs, err := c.Store.GetPushSubscriptionsByUser(userID)
	if err != nil {
		log.Printf("Error getting push subscriptions: %v", err)
		http.Error(w, "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscribed": len(subs) > 0,
		"count":      len(subs),
	})
}

// TestPushHandler sends a test notification to the current user
func (c *Controller) TestPushHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserID(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subs, err := c.Store.GetPushSubscriptionsByUser(userID)
	if err != nil || len(subs) == 0 {
		http.Error(w, "No subscriptions found", http.StatusNotFound)
		return
	}

	vapidPublicKey := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey := os.Getenv("VAPID_PRIVATE_KEY")
	vapidEmail := os.Getenv("VAPID_EMAIL")

	if vapidPublicKey == "" || vapidPrivateKey == "" {
		http.Error(w, "VAPID keys not configured", http.StatusInternalServerError)
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"title": "Test Notification",
		"body":  "Push notifications are working!",
		"url":   "/events/",
	})

	sent := 0
	for _, sub := range subs {
		subscription := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification(payload, subscription, &webpush.Options{
			Subscriber:      vapidEmail,
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			TTL:             30,
		})

		if err != nil {
			log.Printf("Error sending push notification: %v", err)
			// If subscription is invalid, remove it
			if resp != nil && (resp.StatusCode == 410 || resp.StatusCode == 404) {
				c.Store.DeletePushSubscription(sub.Endpoint)
			}
			continue
		}
		defer resp.Body.Close()
		sent++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sent":  sent,
		"total": len(subs),
	})
}

// SendRemindersHandler is called by cron to send event reminders
func (c *Controller) SendRemindersHandler(w http.ResponseWriter, r *http.Request) {
	// Verify cron secret
	cronSecret := os.Getenv("CRON_SECRET")
	if cronSecret != "" {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+cronSecret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	vapidPublicKey := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey := os.Getenv("VAPID_PRIVATE_KEY")
	vapidEmail := os.Getenv("VAPID_EMAIL")

	if vapidPublicKey == "" || vapidPrivateKey == "" {
		http.Error(w, "VAPID keys not configured", http.StatusInternalServerError)
		return
	}

	events, err := c.Store.GetAllEventsForNotifications()
	if err != nil {
		log.Printf("Error getting events: %v", err)
		http.Error(w, "Failed to get events", http.StatusInternalServerError)
		return
	}

	allSubs, err := c.Store.GetAllPushSubscriptions()
	if err != nil {
		log.Printf("Error getting subscriptions: %v", err)
		http.Error(w, "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}

	// Group subscriptions by user
	subsByUser := make(map[int][]models.PushSubscription)
	for _, sub := range allSubs {
		subsByUser[sub.UserID] = append(subsByUser[sub.UserID], sub)
	}

	now := c.now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, c.location)

	sent := 0
	checked := 0

	for _, event := range events {
		checked++

		// For recurring events (birthdays/anniversaries), calculate next occurrence
		// Treat all events as yearly recurring by default
		nextOccurrence := getNextYearlyOccurrence(event.EventDate, today, c.location)
		daysUntil := int(nextOccurrence.Sub(today).Hours() / 24)

		// Notify if event is today or tomorrow
		if daysUntil < 0 || daysUntil > 1 {
			continue
		}

		userSubs := subsByUser[event.UserID]
		if len(userSubs) == 0 {
			continue
		}

		// Calculate age (years since original date)
		age := today.Year() - event.EventDate.Year()
		if today.Month() < event.EventDate.Month() ||
			(today.Month() == event.EventDate.Month() && today.Day() < event.EventDate.Day()) {
			age--
		}

		var body string
		if daysUntil == 0 {
			if age > 0 {
				body = fmt.Sprintf("%s is today! (%d years)", event.Title, age)
			} else {
				body = fmt.Sprintf("%s is today!", event.Title)
			}
		} else {
			if age >= 0 {
				body = fmt.Sprintf("%s is tomorrow! (%d years)", event.Title, age+1)
			} else {
				body = fmt.Sprintf("%s is tomorrow!", event.Title)
			}
		}

		payload, _ := json.Marshal(map[string]string{
			"title": "Event Reminder",
			"body":  body,
			"url":   "/events/lists/" + event.ListName,
			"tag":   fmt.Sprintf("event-%d", event.ID),
		})

		for _, sub := range userSubs {
			subscription := &webpush.Subscription{
				Endpoint: sub.Endpoint,
				Keys: webpush.Keys{
					P256dh: sub.P256dh,
					Auth:   sub.Auth,
				},
			}

			resp, err := webpush.SendNotification(payload, subscription, &webpush.Options{
				Subscriber:      vapidEmail,
				VAPIDPublicKey:  vapidPublicKey,
				VAPIDPrivateKey: vapidPrivateKey,
				TTL:             3600,
			})

			if err != nil {
				log.Printf("Error sending notification for event %d: %v", event.ID, err)
				if resp != nil && (resp.StatusCode == 410 || resp.StatusCode == 404) {
					c.Store.DeletePushSubscription(sub.Endpoint)
				}
				continue
			}
			defer resp.Body.Close()
			sent++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"checked": checked,
		"sent":    sent,
		"date":    today.Format("2006-01-02"),
	})
}

// VAPIDPublicKeyHandler returns the VAPID public key for client-side subscription
func (c *Controller) VAPIDPublicKeyHandler(w http.ResponseWriter, r *http.Request) {
	vapidPublicKey := os.Getenv("VAPID_PUBLIC_KEY")
	if vapidPublicKey == "" {
		http.Error(w, "VAPID key not configured", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"publicKey": vapidPublicKey,
	})
}

// getNextYearlyOccurrence calculates when this event next occurs (for birthdays/anniversaries)
// For example, if event is 21/04/1996 and today is 15/04/2024, returns 21/04/2024
// If event is 21/04/1996 and today is 25/04/2024, returns 21/04/2025
func getNextYearlyOccurrence(eventDate, today time.Time, loc *time.Location) time.Time {
	// Get the month and day of the original event
	month := eventDate.Month()
	day := eventDate.Day()

	// Create this year's occurrence
	thisYear := time.Date(today.Year(), month, day, 0, 0, 0, 0, loc)

	// If this year's occurrence is today or in the future, return it
	if !thisYear.Before(today) {
		return thisYear
	}

	// Otherwise, return next year's occurrence
	return time.Date(today.Year()+1, month, day, 0, 0, 0, 0, loc)
}
