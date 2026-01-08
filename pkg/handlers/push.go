package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/linonymous/Events/pkg/models"
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

// ClearAllSubscriptionsHandler removes all push subscriptions for a user (requires CRON_SECRET)
func (c *Controller) ClearAllSubscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify cron secret
	cronSecret := os.Getenv("CRON_SECRET")
	if cronSecret == "" {
		http.Error(w, "CRON_SECRET not configured", http.StatusInternalServerError)
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+cronSecret {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user_id from query parameter
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id query parameter required", http.StatusBadRequest)
		return
	}

	var userID int
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil || userID == 0 {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	deleted, err := c.Store.DeleteAllPushSubscriptionsByUser(userID)
	if err != nil {
		log.Printf("Error clearing push subscriptions for user %d: %v", userID, err)
		http.Error(w, "Failed to clear subscriptions", http.StatusInternalServerError)
		return
	}

	log.Printf("[CLEAR] Cleared %d subscription(s) for user %d", deleted, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": deleted,
		"user_id": userID,
		"status":  "cleared",
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

	log.Printf("[TEST] Test notification requested for user %d", userID)

	subs, err := c.Store.GetPushSubscriptionsByUser(userID)
	if err != nil {
		log.Printf("[TEST] Error getting subscriptions: %v", err)
		http.Error(w, "Error getting subscriptions", http.StatusInternalServerError)
		return
	}
	if len(subs) == 0 {
		log.Printf("[TEST] No subscriptions found for user %d", userID)
		http.Error(w, "No subscriptions found", http.StatusNotFound)
		return
	}

	log.Printf("[TEST] Found %d subscription(s) for user %d", len(subs), userID)

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
	failed := 0
	errors := []string{}

	for _, sub := range subs {
		subscription := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		log.Printf("[TEST] Sending to endpoint: %s...", sub.Endpoint[:min(80, len(sub.Endpoint))])

		resp, err := webpush.SendNotification(payload, subscription, &webpush.Options{
			Subscriber:      vapidEmail,
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			TTL:             30,
		})

		if err != nil {
			failed++
			errMsg := fmt.Sprintf("Error: %v", err)
			log.Printf("[TEST] %s", errMsg)
			errors = append(errors, errMsg)
			// If subscription is invalid, remove it
			if resp != nil {
				log.Printf("[TEST] Push service response status: %d", resp.StatusCode)
				if resp.StatusCode == 410 || resp.StatusCode == 404 {
					log.Printf("[TEST] Removing invalid subscription (status %d)", resp.StatusCode)
					c.Store.DeletePushSubscription(sub.Endpoint)
					errors = append(errors, "Subscription was invalid and has been removed. Please re-enable notifications.")
				}
			}
			continue
		}
		defer resp.Body.Close()

		// Check for non-2xx status codes (push service rejected the notification)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			failed++
			errMsg := fmt.Sprintf("Push service rejected notification. Status: %d", resp.StatusCode)
			log.Printf("[TEST] %s", errMsg)
			errors = append(errors, errMsg)
			// 403, 404, 410 indicate the subscription is invalid
			if resp.StatusCode == 410 || resp.StatusCode == 404 || resp.StatusCode == 403 {
				log.Printf("[TEST] Removing invalid subscription (status %d)", resp.StatusCode)
				c.Store.DeletePushSubscription(sub.Endpoint)
				errors = append(errors, "Subscription was invalid and has been removed. Please re-enable notifications.")
			}
			continue
		}

		log.Printf("[TEST] Push service accepted notification. Status: %d", resp.StatusCode)
		sent++
	}

	log.Printf("[TEST] Completed. Sent: %d, Failed: %d", sent, failed)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sent":   sent,
		"failed": failed,
		"total":  len(subs),
		"errors": errors,
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
	failed := 0
	skippedNoSubs := 0
	matched := 0

	log.Printf("[CRON] Starting notification check. Total events: %d, Total subscriptions: %d", len(events), len(allSubs))
	log.Printf("[CRON] Today's date (user timezone): %s", today.Format("2006-01-02"))

	for _, event := range events {
		checked++

		var targetDate time.Time
		var daysUntil int

		if event.Recurring {
			// For recurring events (birthdays/anniversaries), calculate next yearly occurrence
			targetDate = getNextYearlyOccurrence(event.EventDate, today, c.location)
		} else {
			// For one-time events, use the actual event date
			targetDate = time.Date(event.EventDate.Year(), event.EventDate.Month(), event.EventDate.Day(), 0, 0, 0, 0, c.location)
		}

		daysUntil = int(targetDate.Sub(today).Hours() / 24)

		// Notify if event is today or tomorrow
		if daysUntil < 0 || daysUntil > 1 {
			continue
		}

		matched++
		log.Printf("[CRON] Event matched for notification: ID=%d, Title=%s, TargetDate=%s, DaysUntil=%d, Recurring=%v",
			event.ID, event.Title, targetDate.Format("2006-01-02"), daysUntil, event.Recurring)

		userSubs := subsByUser[event.UserID]
		if len(userSubs) == 0 {
			skippedNoSubs++
			log.Printf("[CRON] No subscriptions found for user %d (event: %s)", event.UserID, event.Title)
			continue
		}
		log.Printf("[CRON] Found %d subscription(s) for user %d", len(userSubs), event.UserID)

		var body string
		if event.Recurring {
			// Calculate age (years since original date) for recurring events
			age := today.Year() - event.EventDate.Year()
			if today.Month() < event.EventDate.Month() ||
				(today.Month() == event.EventDate.Month() && today.Day() < event.EventDate.Day()) {
				age--
			}

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
		} else {
			// One-time event message
			if daysUntil == 0 {
				body = fmt.Sprintf("%s is today!", event.Title)
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

			// Log subscription endpoint domain for debugging
			log.Printf("[CRON] Sending to endpoint: %s...", sub.Endpoint[:min(80, len(sub.Endpoint))])

			resp, err := webpush.SendNotification(payload, subscription, &webpush.Options{
				Subscriber:      vapidEmail,
				VAPIDPublicKey:  vapidPublicKey,
				VAPIDPrivateKey: vapidPrivateKey,
				TTL:             3600,
			})

			if err != nil {
				failed++
				log.Printf("[CRON] Error sending notification for event %d: %v", event.ID, err)
				if resp != nil {
					log.Printf("[CRON] Push service response status: %d", resp.StatusCode)
					if resp.StatusCode == 410 || resp.StatusCode == 404 {
						log.Printf("[CRON] Removing invalid subscription (status %d)", resp.StatusCode)
						c.Store.DeletePushSubscription(sub.Endpoint)
					}
				}
				continue
			}
			defer resp.Body.Close()

			// Check for non-2xx status codes (push service rejected the notification)
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				failed++
				log.Printf("[CRON] Push service rejected notification. Status: %d", resp.StatusCode)
				// 403, 404, 410 indicate the subscription is invalid
				if resp.StatusCode == 410 || resp.StatusCode == 404 || resp.StatusCode == 403 {
					log.Printf("[CRON] Removing invalid subscription (status %d)", resp.StatusCode)
					c.Store.DeletePushSubscription(sub.Endpoint)
				}
				continue
			}

			log.Printf("[CRON] Push service accepted notification. Status: %d", resp.StatusCode)
			sent++
		}
	}

	log.Printf("[CRON] Completed. Checked: %d, Matched: %d, Sent: %d, Failed: %d, SkippedNoSubs: %d",
		checked, matched, sent, failed, skippedNoSubs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"checked":        checked,
		"matched":        matched,
		"sent":           sent,
		"failed":         failed,
		"skippedNoSubs":  skippedNoSubs,
		"date":           today.Format("2006-01-02"),
		"totalEvents":    len(events),
		"totalSubs":      len(allSubs),
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
