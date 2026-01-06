package models

import "time"

type (
	Event struct {
		ID           int       `json:"id"`
		Title        string    `json:"title"`
		EventDate    time.Time `json:"event_date"`
		EventDateStr string    `json:"event_date_str"`
		AgeInDays    int       `json:"-"`
		AgeInYears   string    `json:"-"`
	}

	EventWithUser struct {
		ID        int       `json:"id"`
		UserID    int       `json:"user_id"`
		ListName  string    `json:"list_name"`
		Title     string    `json:"title"`
		EventDate time.Time `json:"event_date"`
	}

	Backup struct {
		Service string `json:"service"`
		Version string `json:"version"`
		Data    []User `json:"data"`
	}
)
