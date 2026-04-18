package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/zvdy/yamlet/internal/auth"
	"github.com/zvdy/yamlet/internal/storage"

	"github.com/gorilla/mux"
)

// MaxConfigBodyBytes caps the size of a stored YAML config payload.
const MaxConfigBodyBytes = 1 << 20 // 1 MiB

// MaxAdminBodyBytes caps the size of admin JSON payloads.
const MaxAdminBodyBytes = 16 * 1024 // 16 KiB

// Handler contains the dependencies for HTTP handlers
type Handler struct {
	store storage.Store
	auth  auth.Auth
}

// NewHandler creates a new handler instance
func NewHandler(store storage.Store, a auth.Auth) *Handler {
	return &Handler{
		store: store,
		auth:  a,
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

// writeJSON writes v as a JSON response with the given status.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErrorJSON writes an error message as a structured JSON response.
func writeErrorJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// authStatusFor maps an auth error to an HTTP status code.
func authStatusFor(err error) int {
	switch {
	case errors.Is(err, auth.ErrMissingToken), errors.Is(err, auth.ErrInvalidToken):
		return http.StatusUnauthorized
	case errors.Is(err, auth.ErrNamespaceMismatch):
		return http.StatusForbidden
	default:
		return http.StatusUnauthorized
	}
}

// storeStatusFor maps a storage error to an HTTP status code.
func storeStatusFor(err error) int {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, storage.ErrInvalidName):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// StoreConfig handles POST /namespaces/{namespace}/configs/{name}
func (h *Handler) StoreConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	if namespace == "" || name == "" {
		writeErrorJSON(w, http.StatusBadRequest, "namespace and name are required")
		return
	}

	token := h.extractToken(r)
	if err := h.auth.ValidateToken(namespace, token); err != nil {
		writeErrorJSON(w, authStatusFor(err), fmt.Sprintf("Authentication failed: %v", err))
		return
	}

	// Cap body size to prevent memory exhaustion DoS.
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, MaxConfigBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if isMaxBytesError(err) {
			writeErrorJSON(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("Request body exceeds %d bytes", MaxConfigBodyBytes))
			return
		}
		writeErrorJSON(w, http.StatusBadRequest, fmt.Sprintf("Failed to read request body: %v", err))
		return
	}

	if len(body) == 0 {
		writeErrorJSON(w, http.StatusBadRequest, "Request body cannot be empty")
		return
	}

	if err := h.store.Store(namespace, name, body); err != nil {
		log.Printf("Failed to store config %s/%s: %v", namespace, name, err)
		writeErrorJSON(w, storeStatusFor(err), fmt.Sprintf("Failed to store config: %v", err))
		return
	}

	log.Printf("Stored config %s/%s (%d bytes)", namespace, name, len(body))

	writeJSON(w, http.StatusCreated, map[string]interface{}{
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

	if namespace == "" || name == "" {
		writeErrorJSON(w, http.StatusBadRequest, "namespace and name are required")
		return
	}

	token := h.extractToken(r)
	if err := h.auth.ValidateToken(namespace, token); err != nil {
		writeErrorJSON(w, authStatusFor(err), fmt.Sprintf("Authentication failed: %v", err))
		return
	}

	content, err := h.store.Get(namespace, name)
	if err != nil {
		status := storeStatusFor(err)
		if status == http.StatusInternalServerError {
			log.Printf("Failed to get config %s/%s: %v", namespace, name, err)
		}
		writeErrorJSON(w, status, fmt.Sprintf("Failed to get config: %v", err))
		return
	}

	log.Printf("Retrieved config %s/%s (%d bytes)", namespace, name, len(content))

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

// DeleteConfig handles DELETE /namespaces/{namespace}/configs/{name}
func (h *Handler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	if namespace == "" || name == "" {
		writeErrorJSON(w, http.StatusBadRequest, "namespace and name are required")
		return
	}

	token := h.extractToken(r)
	if err := h.auth.ValidateToken(namespace, token); err != nil {
		writeErrorJSON(w, authStatusFor(err), fmt.Sprintf("Authentication failed: %v", err))
		return
	}

	if err := h.store.Delete(namespace, name); err != nil {
		status := storeStatusFor(err)
		if status == http.StatusInternalServerError {
			log.Printf("Failed to delete config %s/%s: %v", namespace, name, err)
		}
		writeErrorJSON(w, status, fmt.Sprintf("Failed to delete config: %v", err))
		return
	}

	log.Printf("Deleted config %s/%s", namespace, name)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Config deleted successfully",
		"namespace": namespace,
		"name":      name,
	})
}

// ListConfigs handles GET /namespaces/{namespace}/configs
func (h *Handler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	if namespace == "" {
		writeErrorJSON(w, http.StatusBadRequest, "namespace is required")
		return
	}

	token := h.extractToken(r)
	if err := h.auth.ValidateToken(namespace, token); err != nil {
		writeErrorJSON(w, authStatusFor(err), fmt.Sprintf("Authentication failed: %v", err))
		return
	}

	configs, err := h.store.List(namespace)
	if err != nil {
		status := storeStatusFor(err)
		if status == http.StatusInternalServerError {
			log.Printf("Failed to list configs for namespace %s: %v", namespace, err)
		}
		writeErrorJSON(w, status, fmt.Sprintf("Failed to list configs: %v", err))
		return
	}

	log.Printf("Listed %d configs for namespace %s", len(configs), namespace)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"namespace": namespace,
		"configs":   configs,
		"count":     len(configs),
	})
}

// Admin endpoints for token management

// authAdminStatusFor maps an auth error from admin operations to an HTTP
// status code.
func authAdminStatusFor(err error) int {
	switch {
	case errors.Is(err, auth.ErrAdminRequired):
		return http.StatusUnauthorized
	case errors.Is(err, auth.ErrTokenNotFound):
		return http.StatusNotFound
	case errors.Is(err, auth.ErrTokenExists), errors.Is(err, auth.ErrInvalidInput):
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}

// CreateToken handles POST /admin/tokens
func (h *Handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	adminToken := h.extractToken(r)
	if adminToken == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "Admin token required")
		return
	}

	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, MaxAdminBodyBytes)

	var req struct {
		Token     string `json:"token"`
		Namespace string `json:"namespace"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if isMaxBytesError(err) {
			writeErrorJSON(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("Request body exceeds %d bytes", MaxAdminBodyBytes))
			return
		}
		writeErrorJSON(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if err := h.auth.CreateToken(adminToken, req.Token, req.Namespace); err != nil {
		writeErrorJSON(w, authAdminStatusFor(err), err.Error())
		return
	}

	log.Printf("Admin created token for namespace %s", req.Namespace)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":   "Token created successfully",
		"token":     req.Token,
		"namespace": req.Namespace,
	})
}

// RevokeToken handles DELETE /admin/tokens/{token}
func (h *Handler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tokenToRevoke := vars["token"]

	adminToken := h.extractToken(r)
	if adminToken == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "Admin token required")
		return
	}

	if err := h.auth.RevokeToken(adminToken, tokenToRevoke); err != nil {
		writeErrorJSON(w, authAdminStatusFor(err), err.Error())
		return
	}

	log.Printf("Admin revoked token: %s", tokenToRevoke)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Token revoked successfully",
		"token":   tokenToRevoke,
	})
}

// ListTokens handles GET /admin/tokens
func (h *Handler) ListTokens(w http.ResponseWriter, r *http.Request) {
	adminToken := h.extractToken(r)
	if adminToken == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "Admin token required")
		return
	}

	tokens, err := h.auth.ListAllTokens(adminToken)
	if err != nil {
		writeErrorJSON(w, authAdminStatusFor(err), err.Error())
		return
	}

	log.Printf("Admin listed %d tokens", len(tokens))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

// isMaxBytesError reports whether err originated from http.MaxBytesReader.
func isMaxBytesError(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}
