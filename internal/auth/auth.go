package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Sentinel errors for callers to classify via errors.Is.
var (
	ErrMissingToken      = errors.New("authorization token is required")
	ErrInvalidToken      = errors.New("invalid token")
	ErrNamespaceMismatch = errors.New("token not authorized for namespace")
	ErrAdminRequired     = errors.New("admin token required")
	ErrTokenExists       = errors.New("token already exists")
	ErrTokenNotFound     = errors.New("token not found")
	ErrInvalidInput      = errors.New("invalid input")
)

// Auth interface defines authentication operations
type Auth interface {
	ValidateToken(namespace, token string) error
	GetNamespaceForToken(token string) (string, error)
	// Admin operations
	IsAdminToken(token string) bool
	CreateToken(adminToken, newToken, namespace string) error
	RevokeToken(adminToken, tokenToRevoke string) error
	ListAllTokens(adminToken string) (map[string]string, error)
}

// TokenAuth implements simple token-based authentication
type TokenAuth struct {
	mu         sync.RWMutex
	tokens     map[string]string // token -> namespace
	adminToken string            // special admin token
}

// NewTokenAuth creates a new token-based auth service
func NewTokenAuth() *TokenAuth {
	auth := &TokenAuth{
		tokens: make(map[string]string),
	}

	// Set admin token from environment or use default
	auth.adminToken = os.Getenv("YAMLET_ADMIN_TOKEN")
	if auth.adminToken == "" {
		auth.adminToken = "admin-secret-token-change-me" // Default for development
	}

	// Load tokens from environment variables
	auth.loadTokensFromEnv()

	// If no tokens loaded, set up minimal default tokens for development
	if len(auth.tokens) == 0 {
		auth.setDevelopmentTokens()
	}

	return auth
}

// loadTokensFromEnv loads tokens from environment variables
// Expected format: YAMLET_TOKENS=token1:namespace1,token2:namespace2
func (t *TokenAuth) loadTokensFromEnv() {
	tokensEnv := os.Getenv("YAMLET_TOKENS")
	if tokensEnv == "" {
		return
	}

	pairs := strings.Split(tokensEnv, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			token := strings.TrimSpace(parts[0])
			namespace := strings.TrimSpace(parts[1])
			if token != "" && namespace != "" {
				t.tokens[token] = namespace
			}
		}
	}
}

// setDevelopmentTokens sets up minimal tokens for development only
func (t *TokenAuth) setDevelopmentTokens() {
	// Only add a few basic tokens for development
	// In production, tokens should be created via admin API
	developmentTokens := map[string]string{
		"dev-token":  "dev",
		"test-token": "test",
	}

	for token, namespace := range developmentTokens {
		t.tokens[token] = namespace
	}
}

// stripBearer removes a leading "Bearer " prefix if present.
func stripBearer(token string) string {
	return strings.TrimPrefix(token, "Bearer ")
}

// ValidateToken validates if a token is valid for the given namespace
func (t *TokenAuth) ValidateToken(namespace, token string) error {
	if token == "" {
		return ErrMissingToken
	}
	token = stripBearer(token)

	t.mu.RLock()
	allowedNamespace, exists := t.tokens[token]
	t.mu.RUnlock()

	if !exists {
		return ErrInvalidToken
	}
	if allowedNamespace != namespace {
		return fmt.Errorf("%w: %s", ErrNamespaceMismatch, namespace)
	}
	return nil
}

// GetNamespaceForToken returns the namespace associated with a token
func (t *TokenAuth) GetNamespaceForToken(token string) (string, error) {
	if token == "" {
		return "", ErrMissingToken
	}
	token = stripBearer(token)

	t.mu.RLock()
	namespace, exists := t.tokens[token]
	t.mu.RUnlock()

	if !exists {
		return "", ErrInvalidToken
	}
	return namespace, nil
}

// AddToken adds a new token for a namespace (useful for testing)
func (t *TokenAuth) AddToken(token, namespace string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tokens[token] = namespace
}

// RemoveToken removes a token (useful for testing)
func (t *TokenAuth) RemoveToken(token string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.tokens, token)
}

// ListTokens returns all configured tokens (for debugging, remove in production)
func (t *TokenAuth) ListTokens() map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	tokens := make(map[string]string, len(t.tokens))
	for token, namespace := range t.tokens {
		tokens[token] = namespace
	}
	return tokens
}

// Admin operations

// IsAdminToken checks if the provided token is the admin token
func (t *TokenAuth) IsAdminToken(token string) bool {
	return stripBearer(token) == t.adminToken
}

// CreateToken creates a new namespace token (admin only)
func (t *TokenAuth) CreateToken(adminToken, newToken, namespace string) error {
	if !t.IsAdminToken(adminToken) {
		return ErrAdminRequired
	}
	if newToken == "" || namespace == "" {
		return fmt.Errorf("%w: token and namespace cannot be empty", ErrInvalidInput)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if newToken == t.adminToken {
		return fmt.Errorf("%w: cannot create token conflicting with admin token", ErrInvalidInput)
	}
	if _, exists := t.tokens[newToken]; exists {
		return ErrTokenExists
	}

	t.tokens[newToken] = namespace
	return nil
}

// RevokeToken removes a namespace token (admin only)
func (t *TokenAuth) RevokeToken(adminToken, tokenToRevoke string) error {
	if !t.IsAdminToken(adminToken) {
		return ErrAdminRequired
	}
	if tokenToRevoke == "" {
		return fmt.Errorf("%w: token to revoke cannot be empty", ErrInvalidInput)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if tokenToRevoke == t.adminToken {
		return fmt.Errorf("%w: cannot revoke admin token", ErrInvalidInput)
	}
	if _, exists := t.tokens[tokenToRevoke]; !exists {
		return ErrTokenNotFound
	}

	delete(t.tokens, tokenToRevoke)
	return nil
}

// ListAllTokens returns all configured tokens (admin only)
func (t *TokenAuth) ListAllTokens(adminToken string) (map[string]string, error) {
	if !t.IsAdminToken(adminToken) {
		return nil, ErrAdminRequired
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	tokens := make(map[string]string, len(t.tokens))
	for token, namespace := range t.tokens {
		tokens[token] = namespace
	}
	return tokens, nil
}
