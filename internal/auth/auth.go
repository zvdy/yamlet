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
	tokens map[string]string // token -> namespace
}

// NewTokenAuth creates a new token-based auth service
func NewTokenAuth() *TokenAuth {
	auth := &TokenAuth{
		tokens: make(map[string]string),
	}

	// Load tokens from environment variables
	auth.loadTokensFromEnv()

	// If no tokens loaded, set up default tokens
	if len(auth.tokens) == 0 {
		auth.setDefaultTokens()
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

// setDefaultTokens sets up default tokens for development
func (t *TokenAuth) setDefaultTokens() {
	defaultTokens := map[string]string{
		"devtoken123":     "dev",
		"stagingtoken456": "staging",
		"prodtoken789":    "production",
		"testtoken000":    "test",
	}

	for token, namespace := range defaultTokens {
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
