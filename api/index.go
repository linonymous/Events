package api

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/linonymous/Events/pkg/handlers"
	"github.com/linonymous/Events/pkg/storage"
	"github.com/gorilla/mux"
)

//go:embed templates/*
var content embed.FS

//go:embed static/*
var staticFiles embed.FS

var (
	router    *mux.Router
	once      sync.Once
	initError error
)

func getRouter() (*mux.Router, error) {
	once.Do(func() {
		router = mux.NewRouter()
		location := time.UTC
		if l, err := time.LoadLocation("Asia/Kolkata"); err == nil {
			location = l
		}

		// Initialize Postgres Storage
		databaseURL := os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			initError = fmt.Errorf("DATABASE_URL environment variable is not set. Please configure it in your Vercel project settings")
			return
		}

		// Check for localhost URLs which won't work in serverless environments
		if strings.Contains(databaseURL, "localhost") || strings.Contains(databaseURL, "127.0.0.1") {
			initError = fmt.Errorf("DATABASE_URL points to localhost (%s). In serverless environments like Vercel, you need an external database. Consider using Neon, Supabase, or Railway for PostgreSQL hosting", databaseURL)
			return
		}

		pgStore, err := storage.NewPostgresStorage(databaseURL)
		if err != nil {
			initError = fmt.Errorf("failed to connect to database: %v. Please verify your DATABASE_URL is correct and the database is accessible", err)
			return
		}

		if err := pgStore.CreateTables(); err != nil {
			initError = fmt.Errorf("failed to create tables: %v", err)
			return
		}

		ctrl := handlers.NewController(location, pgStore)

		// Custom 404 handler
		router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("404: %s %s\n", r.Method, r.URL.Path)
			http.Redirect(w, r, "/events/", http.StatusSeeOther)
		})
		router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})
		router.Use(recoveryMiddleware)

		eventsRouter := router.PathPrefix("/events").Subrouter()

		// Static files (PWA manifest, service worker, icons)
		staticHandler := http.FileServer(http.FS(staticFiles))
		eventsRouter.PathPrefix("/static/").Handler(http.StripPrefix("/events/", staticHandler))

		// Service worker needs to be at root level for scope
		eventsRouter.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript")
			w.Header().Set("Service-Worker-Allowed", "/")
			data, _ := staticFiles.ReadFile("static/sw.js")
			w.Write(data)
		})

		// Public routes
		eventsRouter.HandleFunc("/login", ctrl.LoginHandler(content)).Methods(http.MethodGet, http.MethodPost)
		eventsRouter.HandleFunc("/register", ctrl.RegisterHandler(content)).Methods(http.MethodGet, http.MethodPost)
		eventsRouter.HandleFunc("/logout", ctrl.LogoutHandler)

		// Public API routes (VAPID key, cron)
		eventsRouter.HandleFunc("/api/vapid-key", ctrl.VAPIDPublicKeyHandler).Methods(http.MethodGet)
		eventsRouter.HandleFunc("/api/cron/send-reminders", ctrl.SendRemindersHandler).Methods(http.MethodGet)

		// Protected routes
		protectedRouter := eventsRouter.NewRoute().Subrouter()
		protectedRouter.Use(handlers.AuthMiddleware)

		protectedRouter.HandleFunc("/", ctrl.HomeHandler(content)).Methods(http.MethodGet)
		protectedRouter.HandleFunc("/lists/{list_name}", ctrl.ListHandler(content)).Methods(http.MethodGet)
		protectedRouter.HandleFunc("/lists/{list_name}/edit/{id}", ctrl.EditHandler(content)).Methods(http.MethodGet)
		protectedRouter.HandleFunc("/lists/{list_name}/delete/{id}", ctrl.DeleteHandler).Methods(http.MethodGet)
		protectedRouter.HandleFunc("/save", ctrl.SaveHandler).Methods(http.MethodPost)

		// Push notification API routes (protected)
		protectedRouter.HandleFunc("/api/push/subscribe", ctrl.SubscribePushHandler).Methods(http.MethodPost)
		protectedRouter.HandleFunc("/api/push/unsubscribe", ctrl.UnsubscribePushHandler).Methods(http.MethodPost)
		protectedRouter.HandleFunc("/api/push/status", ctrl.GetPushStatusHandler).Methods(http.MethodGet)
		protectedRouter.HandleFunc("/api/push/test", ctrl.TestPushHandler).Methods(http.MethodPost)
	})
	return router, initError
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				log.Printf("Recovered from panic: %v", err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Handler is the Vercel serverless function entry point
func Handler(w http.ResponseWriter, r *http.Request) {
	router, err := getRouter()
	if err != nil {
		log.Printf("Initialization error: %v", err)
		http.Error(w, fmt.Sprintf("Server configuration error: %v", err), http.StatusInternalServerError)
		return
	}
	router.ServeHTTP(w, r)
}
