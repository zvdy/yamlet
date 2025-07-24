package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Created string `json:"created"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Database string `json:"database"`
	SSL      bool   `json:"ssl"`
	MaxConns int    `json:"max_connections"`
}

var (
	users = []User{
		{ID: 1, Name: "John Doe", Email: "john@example.com", Created: "2025-01-01"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Created: "2025-01-02"},
		{ID: 3, Name: "Bob Wilson", Email: "bob@example.com", Created: "2025-01-03"},
	}

	dbConfig DatabaseConfig
)

func main() {
	port := getEnv("PORT", "3000")

	// Load database configuration from environment variables
	// These would typically be fetched from Yamlet
	loadDatabaseConfig()

	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Database info
	r.HandleFunc("/db/info", dbInfoHandler).Methods("GET")

	// User endpoints
	r.HandleFunc("/users", getUsersHandler).Methods("GET")
	r.HandleFunc("/users/{id}", getUserHandler).Methods("GET")

	// Config endpoint
	r.HandleFunc("/config", configHandler).Methods("GET")

	log.Printf("Mock Database Service starting on port %s", port)
	log.Printf("Database config: %+v", dbConfig)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func loadDatabaseConfig() {
	dbConfig = DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		Database: getEnv("DB_NAME", "mockdb"),
		SSL:      getEnv("DB_SSL", "false") == "true",
		MaxConns: parseIntOrDefault(getEnv("DB_MAX_CONNECTIONS", "10"), 10),
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "mock-database",
	})
}

func dbInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"config":      dbConfig,
		"connections": len(users), // Mock active connections
		"uptime":      "24h",
		"version":     "1.0.0",
	})
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	for _, user := range users {
		if fmt.Sprintf("%d", user.ID) == userID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
			return
		}
	}

	http.Error(w, "User not found", http.StatusNotFound)
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	config := map[string]interface{}{
		"database": dbConfig,
		"app": map[string]interface{}{
			"name":        getEnv("APP_NAME", "mock-db-service"),
			"version":     getEnv("APP_VERSION", "1.0.0"),
			"environment": getEnv("APP_ENVIRONMENT", "development"),
		},
		"features": map[string]interface{}{
			"logging_enabled": getEnv("LOGGING_ENABLED", "true") == "true",
			"metrics_enabled": getEnv("METRICS_ENABLED", "false") == "true",
			"debug_mode":      getEnv("DEBUG_MODE", "true") == "true",
		},
	}

	json.NewEncoder(w).Encode(config)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseIntOrDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	// Simple parsing - in real code you'd handle errors
	switch s {
	case "5":
		return 5
	case "10":
		return 10
	case "20":
		return 20
	case "50":
		return 50
	default:
		return defaultValue
	}
}
