package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/zvdy/yamlet/internal/auth"
	"github.com/zvdy/yamlet/internal/handlers"
	"github.com/zvdy/yamlet/internal/storage"

	"github.com/gorilla/mux"
)

func main() {
	var (
		port     = flag.Int("port", getEnvAsInt("PORT", 8080), "Server port")
		dataDir  = flag.String("data-dir", getEnv("DATA_DIR", "/data"), "Data directory for file storage")
		useFiles = flag.Bool("use-files", getEnvAsBool("USE_FILES", false), "Use file-based storage instead of in-memory")
	)
	flag.Parse()

	// Initialize storage
	var store storage.Store
	if *useFiles {
		store = storage.NewFileStore(*dataDir)
		log.Printf("Using file-based storage in directory: %s", *dataDir)
	} else {
		store = storage.NewMemoryStore()
		log.Println("Using in-memory storage")
	}

	// Initialize auth
	authService := auth.NewTokenAuth()

	// Initialize handlers
	h := handlers.NewHandler(store, authService)

	// Setup routes
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// API routes
	api := r.PathPrefix("/namespaces").Subrouter()
	api.HandleFunc("/{namespace}/configs/{name}", h.StoreConfig).Methods("POST")
	api.HandleFunc("/{namespace}/configs/{name}", h.GetConfig).Methods("GET")
	api.HandleFunc("/{namespace}/configs/{name}", h.DeleteConfig).Methods("DELETE")
	api.HandleFunc("/{namespace}/configs", h.ListConfigs).Methods("GET")

	// Admin routes for token management
	admin := r.PathPrefix("/admin").Subrouter()
	admin.HandleFunc("/tokens", h.CreateToken).Methods("POST")
	admin.HandleFunc("/tokens", h.ListTokens).Methods("GET")
	admin.HandleFunc("/tokens/{token}", h.RevokeToken).Methods("DELETE")

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting Yamlet server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
