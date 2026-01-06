package storage

import "github.com/IAmSurajBobade/sb-dashboard/internal/models"

type Storage interface {
	CreateTables() error
	CreateUser(username, passwordHash string) error
	GetUser(username string) (*models.User, error)
	SaveEvent(userID int, listName string, event *models.Event) error
	UpdateEvent(userID int, event *models.Event) error
	DeleteEvent(userID int, eventID int) error
	GetEvents(userID int, listName string) ([]models.Event, error)
	GetEventByID(eventID int, userID int) (*models.Event, error)
	GetLists(userID int) ([]string, error)

	// Push subscription methods
	SavePushSubscription(userID int, endpoint, p256dh, auth string) error
	DeletePushSubscription(endpoint string) error
	GetPushSubscription(userID int, endpoint string) (*models.PushSubscription, error)
	GetPushSubscriptionsByUser(userID int) ([]models.PushSubscription, error)
	GetAllPushSubscriptions() ([]models.PushSubscription, error)

	// Events for notifications
	GetAllEventsForNotifications() ([]models.EventWithUser, error)
}
