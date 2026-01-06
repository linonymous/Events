package storage

import (
	"database/sql"
	"fmt"


	"github.com/IAmSurajBobade/sb-dashboard/internal/models"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dataSourceName string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) CreateTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id) ON DELETE CASCADE,
			list_name TEXT NOT NULL,
			title TEXT NOT NULL,
			event_date TIMESTAMP WITH TIME ZONE NOT NULL,
			recurring BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		// Migration: Add recurring column if it doesn't exist
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='events' AND column_name='recurring') THEN
				ALTER TABLE events ADD COLUMN recurring BOOLEAN DEFAULT true;
			END IF;
		END $$`,
		`CREATE TABLE IF NOT EXISTS push_subscriptions (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id) ON DELETE CASCADE,
			endpoint TEXT UNIQUE NOT NULL,
			p256dh TEXT NOT NULL,
			auth TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("error creating table: %v", err)
		}
	}
	return nil
}

func (s *PostgresStorage) CreateUser(username, passwordHash string) error {
	_, err := s.db.Exec("INSERT INTO users (username, password_hash) VALUES ($1, $2)", username, passwordHash)
	return err
}

func (s *PostgresStorage) GetUser(username string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow("SELECT id, username, password_hash, created_at FROM users WHERE username = $1", username).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *PostgresStorage) SaveEvent(userID int, listName string, event *models.Event) error {
	err := s.db.QueryRow("INSERT INTO events (user_id, list_name, title, event_date, recurring) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		userID, listName, event.Title, event.EventDate, event.Recurring).Scan(&event.ID)
	return err
}

func (s *PostgresStorage) UpdateEvent(userID int, event *models.Event) error {
	_, err := s.db.Exec("UPDATE events SET title = $1, event_date = $2, recurring = $3 WHERE id = $4 AND user_id = $5",
		event.Title, event.EventDate, event.Recurring, event.ID, userID)
	return err
}

func (s *PostgresStorage) DeleteEvent(userID int, eventID int) error {
	_, err := s.db.Exec("DELETE FROM events WHERE id = $1 AND user_id = $2", eventID, userID)
	return err
}

func (s *PostgresStorage) GetEvents(userID int, listName string) ([]models.Event, error) {
	rows, err := s.db.Query("SELECT id, title, event_date, COALESCE(recurring, true) FROM events WHERE user_id = $1 AND list_name = $2 ORDER BY event_date ASC", userID, listName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.Title, &e.EventDate, &e.Recurring); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (s *PostgresStorage) GetEventByID(eventID int, userID int) (*models.Event, error) {
	var e models.Event
	err := s.db.QueryRow("SELECT id, title, event_date, COALESCE(recurring, true) FROM events WHERE id = $1 AND user_id = $2", eventID, userID).
		Scan(&e.ID, &e.Title, &e.EventDate, &e.Recurring)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *PostgresStorage) GetLists(userID int) ([]string, error) {
	rows, err := s.db.Query("SELECT DISTINCT list_name FROM events WHERE user_id = $1 ORDER BY list_name", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []string
	for rows.Next() {
		var listName string
		if err := rows.Scan(&listName); err != nil {
			return nil, err
		}
		lists = append(lists, listName)
	}
	return lists, nil
}

// Push subscription methods

func (s *PostgresStorage) SavePushSubscription(userID int, endpoint, p256dh, auth string) error {
	// Use upsert to handle existing subscriptions
	_, err := s.db.Exec(`
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id = $1,
			p256dh = $3,
			auth = $4
	`, userID, endpoint, p256dh, auth)
	return err
}

func (s *PostgresStorage) DeletePushSubscription(endpoint string) error {
	_, err := s.db.Exec("DELETE FROM push_subscriptions WHERE endpoint = $1", endpoint)
	return err
}

func (s *PostgresStorage) GetPushSubscription(userID int, endpoint string) (*models.PushSubscription, error) {
	var sub models.PushSubscription
	err := s.db.QueryRow(`
		SELECT id, user_id, endpoint, p256dh, auth, created_at
		FROM push_subscriptions
		WHERE user_id = $1 AND endpoint = $2
	`, userID, endpoint).Scan(&sub.ID, &sub.UserID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *PostgresStorage) GetPushSubscriptionsByUser(userID int) ([]models.PushSubscription, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, endpoint, p256dh, auth, created_at
		FROM push_subscriptions
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.PushSubscription
	for rows.Next() {
		var sub models.PushSubscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (s *PostgresStorage) GetAllPushSubscriptions() ([]models.PushSubscription, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, endpoint, p256dh, auth, created_at
		FROM push_subscriptions
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.PushSubscription
	for rows.Next() {
		var sub models.PushSubscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (s *PostgresStorage) GetAllEventsForNotifications() ([]models.EventWithUser, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, list_name, title, event_date, COALESCE(recurring, true)
		FROM events
		ORDER BY event_date ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.EventWithUser
	for rows.Next() {
		var e models.EventWithUser
		if err := rows.Scan(&e.ID, &e.UserID, &e.ListName, &e.Title, &e.EventDate, &e.Recurring); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
