package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	App         string `json:"app"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Database    struct {
		Host string `json:"host"`
		Port string `json:"port"`
		Name string `json:"name"`
	} `json:"database"`
}

type AppInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Config      map[string]string `json:"config"`
	Database    map[string]string `json:"database"`
	StartTime   time.Time         `json:"start_time"`
}

func main() {
	// This app demonstrates how to use configuration from Yamlet
	log.Println("🚀 Starting Sample Application")

	// Read configuration from environment variables
	// (These would be set by the yamlet-fetch-config.sh script)
	config := Config{
		App:         getEnv("APP", "sample-app"),
		Version:     getEnv("VERSION", "1.0.0"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
	config.Database.Host = getEnv("DATABASE_HOST", "localhost")
	config.Database.Port = getEnv("DATABASE_PORT", "5432")
	config.Database.Name = getEnv("DATABASE_NAME", "sample_db")

	log.Printf("App: %s v%s", config.App, config.Version)
	log.Printf("Environment: %s", config.Environment)
	log.Printf("Database: %s:%s/%s", config.Database.Host, config.Database.Port, config.Database.Name)

	// Create app info
	appInfo := AppInfo{
		Name:        config.App,
		Version:     config.Version,
		Environment: config.Environment,
		Config: map[string]string{
			"app":         config.App,
			"version":     config.Version,
			"environment": config.Environment,
		},
		Database: map[string]string{
			"host": config.Database.Host,
			"port": config.Database.Port,
			"name": config.Database.Name,
		},
		StartTime: time.Now(),
	}

	// Setup HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Hello from Sample Application!",
			"app":     appInfo,
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"app":    config.App,
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appInfo)
	})

	http.HandleFunc("/database", func(w http.ResponseWriter, r *http.Request) {
		// Simulate database connection using config from Yamlet
		dbURL := fmt.Sprintf("http://%s:%s", config.Database.Host, config.Database.Port)

		resp, err := http.Get(dbURL + "/info")
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Database connection failed",
				"db_host": config.Database.Host,
				"db_port": config.Database.Port,
			})
			return
		}
		defer resp.Body.Close()

		var dbInfo map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&dbInfo)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":     "Connected to database",
			"config_from": "yamlet",
			"database":    dbInfo,
			"connection": map[string]string{
				"host": config.Database.Host,
				"port": config.Database.Port,
				"name": config.Database.Name,
			},
		})
	})

	port := getEnv("PORT", "8080")
	log.Printf("🌐 Server starting on port %s", port)
	log.Printf("📋 Available endpoints:")
	log.Printf("  GET / - App info")
	log.Printf("  GET /health - Health check")
	log.Printf("  GET /config - Configuration details")
	log.Printf("  GET /database - Database connection test")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
