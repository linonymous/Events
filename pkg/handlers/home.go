package handlers

import (
	"embed"
	"html/template"
	"log"
	"net/http"
)

func (c *Controller) HomeHandler(content embed.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.mutex.Lock()
		c.userCount++
		cnt := c.userCount
		c.mutex.Unlock()

		userID := GetUserID(r)
		lists, _ := c.Store.GetLists(userID) // Ignore error or handle log

		data := map[string]interface{}{
			"Count":      cnt,
			"TodaysDate": c.now().Format("2006-01-02"),
			"Lists":      lists,
		}

		tmpl, err := template.ParseFS(content, "templates/home.html")
		if err != nil {
			log.Printf("Error parsing template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, data)
	}
}
