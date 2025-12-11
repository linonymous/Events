package handlers

import (
	"embed"
	"html/template"
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

		tmpl, _ := template.ParseFS(content, "templates/home.html")
		tmpl.Execute(w, data)
	}
}
