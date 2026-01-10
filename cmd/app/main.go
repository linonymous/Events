package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"syscall"
	"time"

	"github.com/IAmSurajBobade/Events/pkg/handlers"
	"github.com/IAmSurajBobade/Events/pkg/storage"
	"github.com/gorilla/mux"
)

//go:embed templates/* static/*
var content embed.FS

//go:embed static/*
var staticFiles embed.FS

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8100"
	}

	muxRouter := mux.NewRouter()
	location := time.UTC
	if l, err := time.LoadLocation("Asia/Kolkata"); err == nil {
		location = l
	}

	// Initialize Postgres Storage
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	pgStore, err := storage.NewPostgresStorage(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := pgStore.CreateTables(); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	ctrl := handlers.NewController(location, pgStore)

	// Custom 404 handler
	muxRouter.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// redirect to home page
		fmt.Printf("404: %s %s\n", r.Method, r.URL.Path)
		http.Redirect(w, r, "/events/", http.StatusSeeOther)
	})
	muxRouter.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	muxRouter.Use(recoveryMiddleware)
	// muxRouter.Use(loggingMiddleware)

	// Serve static files
	muxRouter.PathPrefix("/static/").Handler(http.FileServer(http.FS(content)))

	eventsRouter := muxRouter.PathPrefix("/events").Subrouter()

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
	eventsRouter.HandleFunc("/api/push/clear", ctrl.ClearAllSubscriptionsHandler).Methods(http.MethodPost)

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

	// Create a channel to listen for termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		fmt.Println("Server started on port:", port)
		if err := http.ListenAndServe(":"+port, muxRouter); err != nil {
			fmt.Println("Server stopped with error:", err)
		}
	}()

	// Wait for a termination signal
	sig := <-sigChan
	fmt.Println("Received signal:", sig)

	fmt.Println("Server stopped gracefully")
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
