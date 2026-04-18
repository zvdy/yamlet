package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected errors.Is(err, ErrNotFound), got %v", err)
	}

	// Try to get non-existent config in existing namespace
	store.Store("dev", "exists.yaml", []byte("content"))
	_, err = store.Get("dev", "nonexistent.yaml")
	if err == nil {
		t.Fatal("Expected error for non-existent config")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected errors.Is(err, ErrNotFound), got %v", err)
	}

	// Delete non-existent config should also be ErrNotFound
	err = store.Delete("dev", "missing.yaml")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected errors.Is(err, ErrNotFound) from Delete, got %v", err)
	}
}

func TestFileStoreNotFoundIsSentinel(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "yamlet-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewFileStore(tempDir)

	_, err = store.Get("missing-ns", "missing.yaml")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected ErrNotFound from Get, got %v", err)
	}

	err = store.Delete("missing-ns", "missing.yaml")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected ErrNotFound from Delete, got %v", err)
	}
}

// TestFileStorePathTraversalRejected verifies that namespace/name values
// containing path-traversal components cannot escape the base dir.
func TestFileStorePathTraversalRejected(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "yamlet-traversal")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a sentinel file outside baseDir that must never be touched.
	outsideDir, err := os.MkdirTemp("", "yamlet-outside")
	if err != nil {
		t.Fatalf("Failed to create outside dir: %v", err)
	}
	defer os.RemoveAll(outsideDir)

	sentinel := filepath.Join(outsideDir, "sentinel.yaml")
	if err := os.WriteFile(sentinel, []byte("original"), 0600); err != nil {
		t.Fatalf("Failed to write sentinel: %v", err)
	}

	store := NewFileStore(tempDir)

	badInputs := []struct {
		ns   string
		name string
	}{
		{"..", "whatever.yaml"},
		{"../etc", "passwd"},
		{"../../etc", "passwd"},
		{"ns", ".."},
		{"ns", "../escape.yaml"},
		{"ns", "sub/nested.yaml"},
		{"ns/sub", "x.yaml"},
		{"", "x.yaml"},
		{"ns", ""},
		{".", "x.yaml"},
		{"ns", "."},
		{"ns\x00", "x.yaml"},
	}

	for _, bi := range badInputs {
		t.Run("store/"+bi.ns+"/"+bi.name, func(t *testing.T) {
			if err := store.Store(bi.ns, bi.name, []byte("pwn")); err == nil {
				t.Fatalf("Store(%q, %q) should have returned an error", bi.ns, bi.name)
			}
			if _, err := store.Get(bi.ns, bi.name); err == nil {
				t.Fatalf("Get(%q, %q) should have returned an error", bi.ns, bi.name)
			}
			if err := store.Delete(bi.ns, bi.name); err == nil {
				t.Fatalf("Delete(%q, %q) should have returned an error", bi.ns, bi.name)
			}
		})
	}

	// Sentinel outside the store must be untouched.
	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("Sentinel file missing after traversal attempts: %v", err)
	}
	if string(got) != "original" {
		t.Fatalf("Sentinel contents changed: %q", got)
	}
}

// TestFileStoreListPathTraversalRejected ensures List cannot escape the base.
func TestFileStoreListPathTraversalRejected(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "yamlet-list-traversal")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewFileStore(tempDir)

	for _, ns := range []string{"..", "../etc", "ns/sub", "", ".", "ns\x00"} {
		if _, err := store.List(ns); err == nil {
			t.Fatalf("List(%q) should have returned an error", ns)
		}
	}
}

// TestMemoryStoreRejectsInvalidNames ensures MemoryStore rejects obviously
// invalid namespace/name values for consistency with FileStore.
func TestMemoryStoreRejectsInvalidNames(t *testing.T) {
	store := NewMemoryStore()

	for _, bi := range []struct{ ns, name string }{
		{"", "x"}, {"ns", ""}, {".", "x"}, {"..", "x"}, {"ns", "."}, {"ns", ".."},
		{"ns/sub", "x"}, {"ns", "sub/x"}, {"ns\x00", "x"}, {"ns", "x\x00"},
	} {
		if err := store.Store(bi.ns, bi.name, []byte("x")); err == nil {
			t.Errorf("MemoryStore.Store(%q, %q) should error", bi.ns, bi.name)
		}
	}
}

// TestStoreConcurrent exercises concurrent access under -race.
func TestStoreConcurrent(t *testing.T) {
	store := NewMemoryStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := "cfg.yaml"
			ns := "ns"
			_ = store.Store(ns, name, []byte(strings.Repeat("a", i)))
			_, _ = store.Get(ns, name)
			_, _ = store.List(ns)
		}(i)
	}
	wg.Wait()
}
