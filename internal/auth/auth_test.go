package auth

import (
	"os"
	"testing"
)

func TestTokenAuth(t *testing.T) {
	auth := NewTokenAuth()

	// Test default tokens
	err := auth.ValidateToken("dev", "devtoken123")
	if err != nil {
		t.Fatalf("Expected valid token: %v", err)
	}

	// Test invalid token
	err = auth.ValidateToken("dev", "invalidtoken")
	if err == nil {
		t.Fatal("Expected invalid token error")
	}

	// Test wrong namespace
	err = auth.ValidateToken("prod", "devtoken123")
	if err == nil {
		t.Fatal("Expected namespace mismatch error")
	}

	// Test bearer prefix
	err = auth.ValidateToken("dev", "Bearer devtoken123")
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

	namespace, err := auth.GetNamespaceForToken("devtoken123")
	if err != nil {
		t.Fatalf("Failed to get namespace: %v", err)
	}

	if namespace != "dev" {
		t.Fatalf("Expected namespace 'dev', got '%s'", namespace)
	}

	// Test with Bearer prefix
	namespace, err = auth.GetNamespaceForToken("Bearer devtoken123")
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
