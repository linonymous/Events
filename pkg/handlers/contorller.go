package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/IAmSurajBobade/Events/pkg/models"
	"github.com/IAmSurajBobade/Events/pkg/storage"
	"github.com/gorilla/mux"
)

type Controller struct {
	mutex     *sync.Mutex
	userCount int
	location  *time.Location
	Store     storage.Storage
}

func NewController(location *time.Location, store storage.Storage) *Controller {

	return &Controller{
		mutex:     &sync.Mutex{},
		userCount: 0,
		location:  location,
		Store:     store,
	}
}

func (c *Controller) ListHandler(content embed.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := GetUserID(r)
		listName := vars["list_name"]

		events, err := c.Store.GetEvents(userID, listName)
		if err != nil {
			http.Error(w, "Error retrieving events", http.StatusInternalServerError)
			return
		}

		log.Printf("%s %s %s %v", r.Method, r.RequestURI, r.RemoteAddr, r.UserAgent())

		currTime := c.now()

		for i := range events {
			events[i].EventDate = events[i].EventDate.In(c.location)
			events[i].EventDateStr = events[i].EventDate.Format("02/Jan/2006")

			// Calculate age in calendar days
			d1 := time.Date(events[i].EventDate.Year(), events[i].EventDate.Month(), events[i].EventDate.Day(), 0, 0, 0, 0, c.location)
			d2 := time.Date(currTime.Year(), currTime.Month(), currTime.Day(), 0, 0, 0, 0, c.location)
			events[i].AgeInDays = int(d2.Sub(d1).Hours() / 24)

			y, m, d := diff(events[i].EventDate, currTime)
			events[i].AgeInYears = setAgeInYears(y, m, d)
		}

		data := map[string]interface{}{
			"UserID":   userID,
			"ListName": listName,
			"Events":   events,
		}

		tmpl, err := template.ParseFS(content, "templates/user/list.html")
		if err != nil {
			log.Printf("Error parsing template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, data)
	}
}

func setAgeInYears(y, m, d int) string {
	parts := []string{}
	if y == 1 {
		parts = append(parts, "1 year")
	} else if y > 1 {
		parts = append(parts, fmt.Sprintf("%d years", y))
	}
	if m == 1 {
		parts = append(parts, "1 month")
	} else if m > 1 {
		parts = append(parts, fmt.Sprintf("%d months", m))
	}
	if d == 1 {
		parts = append(parts, "1 day")
	} else if d > 1 {
		parts = append(parts, fmt.Sprintf("%d days", d))
	}

	if len(parts) == 0 {
		return "0 days"
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += " " + parts[i]
	}
	return result
}

func (c *Controller) EditHandler(content embed.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventID, _ := strconv.Atoi(vars["id"])
		userID := GetUserID(r)
		listName := vars["list_name"]

		event, err := c.Store.GetEventByID(eventID, userID)
		if err != nil {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}

		data := map[string]interface{}{
			"ID":           event.ID,
			"Title":        event.Title,
			"EventDateStr": event.EventDate.In(c.location).Format("2006-01-02"),
			"Recurring":    event.Recurring,
			"MobileNumber": event.MobileNumber,
			"UserID":       userID,
			"ListName":     listName,
		}

		tmpl, err := template.ParseFS(content, "templates/user/edit_event.html")
		if err != nil {
			log.Printf("Error parsing template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, data)
	}
}

func (c *Controller) DeleteHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	eventID, _ := strconv.Atoi(vars["id"])
	userID := GetUserID(r)
	listName := vars["list_name"]

	if err := c.Store.DeleteEvent(userID, eventID); err != nil {
		http.Error(w, "Error deleting event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/events/lists/%s", listName))
}

func (c *Controller) SaveHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	title := r.FormValue("title")
	dateStr := r.FormValue("event_date")
	eventDate, err := time.ParseInLocation("2006-01-02", dateStr, c.location)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	// Parse recurring field - defaults to true (yearly) if not specified
	recurring := r.FormValue("recurring") != "false"
	mobileNumber := r.FormValue("mobile_number")

	event := models.Event{
		ID:           id,
		Title:        title,
		EventDate:    eventDate,
		Recurring:    recurring,
		MobileNumber: mobileNumber,
	}

	userID := GetUserID(r)
	listName := r.FormValue("list_name")

	if event.ID == 0 {
		err = c.Store.SaveEvent(userID, listName, &event)
	} else {
		err = c.Store.UpdateEvent(userID, &event)
	}

	if err != nil {
		http.Error(w, "Error saving event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/events/lists/%s", listName))
}

func diff(a, b time.Time) (year, month, day int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour := int(h2 - h1)
	min := int(m2 - m1)
	sec := int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		month--
		// Get days in the month before b
		t := time.Date(y2, M2, 0, 0, 0, 0, 0, b.Location())
		day += t.Day()
	}
	if month < 0 {
		month += 12
		year--
	}

	return
}
