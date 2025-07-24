package auth

import (
	"os"
	"testing"
)

func TestTokenAuth(t *testing.T) {
	auth := NewTokenAuth()

	// Test default tokens (updated for new system)
	err := auth.ValidateToken("dev", "dev-token")
	if err != nil {
		t.Fatalf("Expected valid token: %v", err)
	}

	// Test invalid token
	err = auth.ValidateToken("dev", "invalidtoken")
	if err == nil {
		t.Fatal("Expected invalid token error")
	}

	// Test wrong namespace
	err = auth.ValidateToken("production", "dev-token")
	if err == nil {
		t.Fatal("Expected namespace mismatch error")
	}

	// Test bearer prefix
	err = auth.ValidateToken("dev", "Bearer dev-token")
	if err != nil {
		t.Fatalf("Expected bearer token to work: %v", err)
	}
}

func TestTokenAuthFromEnv(t *testing.T) {
	// Set environment variable
	os.Setenv("YAMLET_TOKENS", "custom123:custom,test456:testing")
	defer os.Unsetenv("YAMLET_TOKENS")

	auth := NewTokenAuth()

	// Test custom tokens
	err := auth.ValidateToken("custom", "custom123")
	if err != nil {
		t.Fatalf("Expected custom token to work: %v", err)
	}

	err = auth.ValidateToken("testing", "test456")
	if err != nil {
		t.Fatalf("Expected test token to work: %v", err)
	}
}

func TestGetNamespaceForToken(t *testing.T) {
	auth := NewTokenAuth()

	namespace, err := auth.GetNamespaceForToken("dev-token")
	if err != nil {
		t.Fatalf("Failed to get namespace: %v", err)
	}

	if namespace != "dev" {
		t.Fatalf("Expected namespace 'dev', got '%s'", namespace)
	}

	// Test with Bearer prefix
	namespace, err = auth.GetNamespaceForToken("Bearer dev-token")
	if err != nil {
		t.Fatalf("Failed to get namespace with Bearer: %v", err)
	}

	if namespace != "dev" {
		t.Fatalf("Expected namespace 'dev', got '%s'", namespace)
	}
}

func TestAddRemoveToken(t *testing.T) {
	auth := NewTokenAuth()

	// Add custom token
	auth.AddToken("newtoken", "newnamespace")

	err := auth.ValidateToken("newnamespace", "newtoken")
	if err != nil {
		t.Fatalf("Failed to validate new token: %v", err)
	}

	// Remove token
	auth.RemoveToken("newtoken")

	err = auth.ValidateToken("newnamespace", "newtoken")
	if err == nil {
		t.Fatal("Token should have been removed")
	}
}

// Test admin functionality
func TestAdminTokenManagement(t *testing.T) {
	auth := NewTokenAuth()

	// Test admin token recognition
	if !auth.IsAdminToken("admin-secret-token-change-me") {
		t.Fatal("Default admin token should be recognized")
	}

	if auth.IsAdminToken("dev-token") {
		t.Fatal("Regular token should not be admin")
	}

	// Test creating new token
	err := auth.CreateToken("admin-secret-token-change-me", "new-prod-token", "production")
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Test new token works
	err = auth.ValidateToken("production", "new-prod-token")
	if err != nil {
		t.Fatalf("New token should work: %v", err)
	}

	// Test non-admin cannot create tokens
	err = auth.CreateToken("dev-token", "hacker-token", "evil")
	if err == nil {
		t.Fatal("Non-admin should not be able to create tokens")
	}

	// Test listing tokens
	tokens, err := auth.ListAllTokens("admin-secret-token-change-me")
	if err != nil {
		t.Fatalf("Failed to list tokens: %v", err)
	}

	if _, exists := tokens["new-prod-token"]; !exists {
		t.Fatal("New token should appear in list")
	}

	// Test revoking token
	err = auth.RevokeToken("admin-secret-token-change-me", "new-prod-token")
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// Test revoked token no longer works
	err = auth.ValidateToken("production", "new-prod-token")
	if err == nil {
		t.Fatal("Revoked token should not work")
	}
}
