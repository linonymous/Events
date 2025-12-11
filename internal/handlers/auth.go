package handlers

import (
	"embed"
	"html/template"
	"net/http"


	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))

func (c *Controller) LoginHandler(content embed.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl, _ := template.ParseFS(content, "templates/login.html")
			tmpl.Execute(w, nil)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		user, err := c.Store.GetUser(username)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		session, _ := store.Get(r, "session-name")
		session.Values["user_id"] = user.ID
		session.Values["username"] = user.Username
		session.Save(r, w)

		http.Redirect(w, r, "/events/", http.StatusSeeOther)
	}
}

func (c *Controller) RegisterHandler(content embed.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl, _ := template.ParseFS(content, "templates/register.html")
			tmpl.Execute(w, nil)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = c.Store.CreateUser(username, string(hashedPassword))
		if err != nil {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}

		http.Redirect(w, r, "/events/login", http.StatusSeeOther)
	}
}

func (c *Controller) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values["user_id"] = nil
	session.Values["username"] = nil
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/events/login", http.StatusSeeOther)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session-name")
		if auth, ok := session.Values["user_id"].(int); !ok || auth == 0 {
			http.Redirect(w, r, "/events/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserID(r *http.Request) int {
    session, _ := store.Get(r, "session-name")
    if userID, ok := session.Values["user_id"].(int); ok {
        return userID
    }
    return 0
}
