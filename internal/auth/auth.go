package auth

import (
	"fmt"
	"os"
	"strings"
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

// ValidateToken validates if a token is valid for the given namespace
func (t *TokenAuth) ValidateToken(namespace, token string) error {
	if token == "" {
		return fmt.Errorf("authorization token is required")
	}

	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	allowedNamespace, exists := t.tokens[token]
	if !exists {
		return fmt.Errorf("invalid token")
	}

	if allowedNamespace != namespace {
		return fmt.Errorf("token not authorized for namespace %s", namespace)
	}

	return nil
}

// GetNamespaceForToken returns the namespace associated with a token
func (t *TokenAuth) GetNamespaceForToken(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("authorization token is required")
	}

	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	namespace, exists := t.tokens[token]
	if !exists {
		return "", fmt.Errorf("invalid token")
	}

	return namespace, nil
}

// AddToken adds a new token for a namespace (useful for testing)
func (t *TokenAuth) AddToken(token, namespace string) {
	t.tokens[token] = namespace
}

// RemoveToken removes a token (useful for testing)
func (t *TokenAuth) RemoveToken(token string) {
	delete(t.tokens, token)
}

// ListTokens returns all configured tokens (for debugging, remove in production)
func (t *TokenAuth) ListTokens() map[string]string {
	tokens := make(map[string]string)
	for token, namespace := range t.tokens {
		tokens[token] = namespace
	}
	return tokens
}

// Admin operations

// IsAdminToken checks if the provided token is the admin token
func (t *TokenAuth) IsAdminToken(token string) bool {
	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	return token == t.adminToken
}

// CreateToken creates a new namespace token (admin only)
func (t *TokenAuth) CreateToken(adminToken, newToken, namespace string) error {
	if !t.IsAdminToken(adminToken) {
		return fmt.Errorf("admin token required for token creation")
	}

	if newToken == "" || namespace == "" {
		return fmt.Errorf("token and namespace cannot be empty")
	}

	// Check if token already exists
	if _, exists := t.tokens[newToken]; exists {
		return fmt.Errorf("token already exists")
	}

	// Prevent creating admin token as namespace token
	if newToken == t.adminToken {
		return fmt.Errorf("cannot create token that conflicts with admin token")
	}

	t.tokens[newToken] = namespace
	return nil
}

// RevokeToken removes a namespace token (admin only)
func (t *TokenAuth) RevokeToken(adminToken, tokenToRevoke string) error {
	if !t.IsAdminToken(adminToken) {
		return fmt.Errorf("admin token required for token revocation")
	}

	if tokenToRevoke == "" {
		return fmt.Errorf("token to revoke cannot be empty")
	}

	// Prevent revoking admin token
	if tokenToRevoke == t.adminToken {
		return fmt.Errorf("cannot revoke admin token")
	}

	if _, exists := t.tokens[tokenToRevoke]; !exists {
		return fmt.Errorf("token not found")
	}

	delete(t.tokens, tokenToRevoke)
	return nil
}

// ListAllTokens returns all configured tokens (admin only)
func (t *TokenAuth) ListAllTokens(adminToken string) (map[string]string, error) {
	if !t.IsAdminToken(adminToken) {
		return nil, fmt.Errorf("admin token required for listing tokens")
	}

	tokens := make(map[string]string)
	for token, namespace := range t.tokens {
		tokens[token] = namespace
	}
	return tokens, nil
}
