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

	"github.com/IAmSurajBobade/events/internal/handlers"
	"github.com/IAmSurajBobade/events/internal/storage"
	"github.com/gorilla/mux"
)

//go:embed templates/*
var content embed.FS

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

	eventsRouter := muxRouter.PathPrefix("/events").Subrouter()

	// Public routes
	eventsRouter.HandleFunc("/login", ctrl.LoginHandler(content)).Methods(http.MethodGet, http.MethodPost)
	eventsRouter.HandleFunc("/register", ctrl.RegisterHandler(content)).Methods(http.MethodGet, http.MethodPost)
	eventsRouter.HandleFunc("/logout", ctrl.LogoutHandler)

	// Protected routes
	protectedRouter := eventsRouter.NewRoute().Subrouter()
	protectedRouter.Use(handlers.AuthMiddleware)

	protectedRouter.HandleFunc("/", ctrl.HomeHandler(content)).Methods(http.MethodGet)
	protectedRouter.HandleFunc("/lists/{list_name}", ctrl.ListHandler(content)).Methods(http.MethodGet)
	protectedRouter.HandleFunc("/lists/{list_name}/edit/{id}", ctrl.EditHandler(content)).Methods(http.MethodGet)
	protectedRouter.HandleFunc("/lists/{list_name}/delete/{id}", ctrl.DeleteHandler).Methods(http.MethodGet)
	protectedRouter.HandleFunc("/save", ctrl.SaveHandler).Methods(http.MethodPost)

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
