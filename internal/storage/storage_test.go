package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	// Test store and get
	content := []byte("test: yaml\nvalue: 123")
	err := store.Store("dev", "test.yaml", content)
	if err != nil {
		t.Fatalf("Failed to store: %v", err)
	}

	retrieved, err := store.Get("dev", "test.yaml")
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if string(retrieved) != string(content) {
		t.Fatalf("Content mismatch: got %s, want %s", retrieved, content)
	}

	// Test list
	configs, err := store.List("dev")
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}

	if len(configs) != 1 || configs[0] != "test.yaml" {
		t.Fatalf("List mismatch: got %v, want [test.yaml]", configs)
	}

	// Test delete
	err = store.Delete("dev", "test.yaml")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Should be empty now
	configs, err = store.List("dev")
	if err != nil {
		t.Fatalf("Failed to list after delete: %v", err)
	}

	if len(configs) != 0 {
		t.Fatalf("List should be empty after delete: got %v", configs)
	}
}

func TestFileStore(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "yamlet-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewFileStore(tempDir)

	// Test store and get
	content := []byte("test: yaml\nvalue: 456")
	err = store.Store("staging", "test.yaml", content)
	if err != nil {
		t.Fatalf("Failed to store: %v", err)
	}

	// Check file exists
	filePath := filepath.Join(tempDir, "staging", "test.yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("File was not created: %s", filePath)
	}

	retrieved, err := store.Get("staging", "test.yaml")
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if string(retrieved) != string(content) {
		t.Fatalf("Content mismatch: got %s, want %s", retrieved, content)
	}

	// Test list
	configs, err := store.List("staging")
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}

	if len(configs) != 1 || configs[0] != "test.yaml" {
		t.Fatalf("List mismatch: got %v, want [test.yaml]", configs)
	}

	// Test delete
	err = store.Delete("staging", "test.yaml")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// File should be gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("File should be deleted: %s", filePath)
	}
}

func TestStoreNotFound(t *testing.T) {
	store := NewMemoryStore()

	// Try to get non-existent config
	_, err := store.Get("nonexistent", "test.yaml")
	if err == nil {
		t.Fatal("Expected error for non-existent namespace")
	}

	// Try to get non-existent config in existing namespace
	store.Store("dev", "exists.yaml", []byte("content"))
	_, err = store.Get("dev", "nonexistent.yaml")
	if err == nil {
		t.Fatal("Expected error for non-existent config")
	}
}
