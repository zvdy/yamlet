package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/zvdy/yamlet/internal/auth"
	"github.com/zvdy/yamlet/internal/storage"

	"github.com/gorilla/mux"
)

// Handler contains the dependencies for HTTP handlers
type Handler struct {
	store storage.Store
	auth  auth.Auth
}

// NewHandler creates a new handler instance
func NewHandler(store storage.Store, auth auth.Auth) *Handler {
	return &Handler{
		store: store,
		auth:  auth,
	}
}

// extractToken extracts the token from the Authorization header
func (h *Handler) extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Support both "Bearer token" and just "token" formats
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}

// StoreConfig handles POST /namespaces/{namespace}/configs/{name}
func (h *Handler) StoreConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	// Validate input
	if namespace == "" || name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	// Authenticate
	token := h.extractToken(r)
	if err := h.auth.ValidateToken(namespace, token); err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "Request body cannot be empty", http.StatusBadRequest)
		return
	}

	// Store the config
	if err := h.store.Store(namespace, name, body); err != nil {
		log.Printf("Failed to store config %s/%s: %v", namespace, name, err)
		http.Error(w, fmt.Sprintf("Failed to store config: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Stored config %s/%s (%d bytes)", namespace, name, len(body))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Config stored successfully",
		"namespace": namespace,
		"name":      name,
		"size":      len(body),
	})
}

// GetConfig handles GET /namespaces/{namespace}/configs/{name}
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	// Validate input
	if namespace == "" || name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	// Authenticate
	token := h.extractToken(r)
	if err := h.auth.ValidateToken(namespace, token); err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Get the config
	content, err := h.store.Get(namespace, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf("Config not found: %v", err), http.StatusNotFound)
		} else {
			log.Printf("Failed to get config %s/%s: %v", namespace, name, err)
			http.Error(w, fmt.Sprintf("Failed to get config: %v", err), http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Retrieved config %s/%s (%d bytes)", namespace, name, len(content))

	// Return the content as-is (YAML)
	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// DeleteConfig handles DELETE /namespaces/{namespace}/configs/{name}
func (h *Handler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	// Validate input
	if namespace == "" || name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	// Authenticate
	token := h.extractToken(r)
	if err := h.auth.ValidateToken(namespace, token); err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Delete the config
	if err := h.store.Delete(namespace, name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf("Config not found: %v", err), http.StatusNotFound)
		} else {
			log.Printf("Failed to delete config %s/%s: %v", namespace, name, err)
			http.Error(w, fmt.Sprintf("Failed to delete config: %v", err), http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Deleted config %s/%s", namespace, name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Config deleted successfully",
		"namespace": namespace,
		"name":      name,
	})
}

// ListConfigs handles GET /namespaces/{namespace}/configs
func (h *Handler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	// Validate input
	if namespace == "" {
		http.Error(w, "namespace is required", http.StatusBadRequest)
		return
	}

	// Authenticate
	token := h.extractToken(r)
	if err := h.auth.ValidateToken(namespace, token); err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	// List configs
	configs, err := h.store.List(namespace)
	if err != nil {
		log.Printf("Failed to list configs for namespace %s: %v", namespace, err)
		http.Error(w, fmt.Sprintf("Failed to list configs: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Listed %d configs for namespace %s", len(configs), namespace)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"namespace": namespace,
		"configs":   configs,
		"count":     len(configs),
	})
}
