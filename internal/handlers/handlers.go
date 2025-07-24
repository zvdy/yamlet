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

// Admin endpoints for token management

// CreateToken handles POST /admin/tokens
func (h *Handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	// Extract admin token
	adminToken := h.extractToken(r)
	if adminToken == "" {
		http.Error(w, "Admin token required", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req struct {
		Token     string `json:"token"`
		Namespace string `json:"namespace"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Create the token
	if err := h.auth.CreateToken(adminToken, req.Token, req.Namespace); err != nil {
		if strings.Contains(err.Error(), "admin token required") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	log.Printf("Admin created token for namespace %s", req.Namespace)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Token created successfully",
		"token":     req.Token,
		"namespace": req.Namespace,
	})
}

// RevokeToken handles DELETE /admin/tokens/{token}
func (h *Handler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tokenToRevoke := vars["token"]

	// Extract admin token
	adminToken := h.extractToken(r)
	if adminToken == "" {
		http.Error(w, "Admin token required", http.StatusUnauthorized)
		return
	}

	// Revoke the token
	if err := h.auth.RevokeToken(adminToken, tokenToRevoke); err != nil {
		if strings.Contains(err.Error(), "admin token required") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		} else if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	log.Printf("Admin revoked token: %s", tokenToRevoke)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Token revoked successfully",
		"token":   tokenToRevoke,
	})
}

// ListTokens handles GET /admin/tokens
func (h *Handler) ListTokens(w http.ResponseWriter, r *http.Request) {
	// Extract admin token
	adminToken := h.extractToken(r)
	if adminToken == "" {
		http.Error(w, "Admin token required", http.StatusUnauthorized)
		return
	}

	// List all tokens
	tokens, err := h.auth.ListAllTokens(adminToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	log.Printf("Admin listed %d tokens", len(tokens))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tokens": tokens,
		"count":  len(tokens),
	})
}
