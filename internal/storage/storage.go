package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrNotFound is returned when a namespace or config does not exist.
var ErrNotFound = errors.New("not found")

// ErrInvalidName is returned when a namespace or config name is rejected
// because it is empty, contains a path separator or NUL byte, or resolves to
// a relative traversal component (".", "..").
var ErrInvalidName = errors.New("invalid name")

// Store interface defines the storage operations
type Store interface {
	Store(namespace, name string, content []byte) error
	Get(namespace, name string) ([]byte, error)
	Delete(namespace, name string) error
	List(namespace string) ([]string, error)
}

// validateName enforces that a namespace/config segment is a single, safe
// filesystem component. Segments may not be empty, contain path separators,
// NUL bytes, or be relative traversal markers.
func validateName(s string) error {
	if s == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}
	if s == "." || s == ".." {
		return fmt.Errorf("%w: %q", ErrInvalidName, s)
	}
	if strings.ContainsAny(s, "/\\\x00") {
		return fmt.Errorf("%w: %q contains path separator or NUL", ErrInvalidName, s)
	}
	return nil
}

func validateNamespaceAndName(namespace, name string) error {
	if err := validateName(namespace); err != nil {
		return err
	}
	return validateName(name)
}

// MemoryStore implements in-memory storage
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]map[string][]byte // namespace -> configName -> content
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]map[string][]byte),
	}
}

func (m *MemoryStore) Store(namespace, name string, content []byte) error {
	if err := validateNamespaceAndName(namespace, name); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data[namespace] == nil {
		m.data[namespace] = make(map[string][]byte)
	}
	m.data[namespace][name] = content
	return nil
}

func (m *MemoryStore) Get(namespace, name string) ([]byte, error) {
	if err := validateNamespaceAndName(namespace, name); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	namespaceData, exists := m.data[namespace]
	if !exists {
		return nil, fmt.Errorf("namespace %s: %w", namespace, ErrNotFound)
	}

	content, exists := namespaceData[name]
	if !exists {
		return nil, fmt.Errorf("config %s in namespace %s: %w", name, namespace, ErrNotFound)
	}

	return content, nil
}

func (m *MemoryStore) Delete(namespace, name string) error {
	if err := validateNamespaceAndName(namespace, name); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	namespaceData, exists := m.data[namespace]
	if !exists {
		return fmt.Errorf("namespace %s: %w", namespace, ErrNotFound)
	}

	if _, exists := namespaceData[name]; !exists {
		return fmt.Errorf("config %s in namespace %s: %w", name, namespace, ErrNotFound)
	}

	delete(namespaceData, name)

	// Clean up empty namespace
	if len(namespaceData) == 0 {
		delete(m.data, namespace)
	}

	return nil
}

func (m *MemoryStore) List(namespace string) ([]string, error) {
	if err := validateName(namespace); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	namespaceData, exists := m.data[namespace]
	if !exists {
		return []string{}, nil
	}

	configs := make([]string, 0, len(namespaceData))
	for name := range namespaceData {
		configs = append(configs, name)
	}

	return configs, nil
}

// FileStore implements file-based storage
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileStore creates a new file-based store
func NewFileStore(baseDir string) *FileStore {
	return &FileStore{
		baseDir: baseDir,
	}
}

func (f *FileStore) getFilePath(namespace, name string) string {
	return filepath.Join(f.baseDir, namespace, name)
}

// resolvePath validates the inputs and returns a cleaned path guaranteed to
// sit under f.baseDir. Any traversal attempt returns ErrInvalidName.
func (f *FileStore) resolvePath(namespace, name string) (string, error) {
	if err := validateNamespaceAndName(namespace, name); err != nil {
		return "", err
	}
	base := filepath.Clean(f.baseDir)
	full := filepath.Clean(filepath.Join(base, namespace, name))
	// Defense in depth: ensure the cleaned path is under base.
	rel, err := filepath.Rel(base, full)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%w: resolves outside base", ErrInvalidName)
	}
	return full, nil
}

func (f *FileStore) resolveNamespaceDir(namespace string) (string, error) {
	if err := validateName(namespace); err != nil {
		return "", err
	}
	base := filepath.Clean(f.baseDir)
	full := filepath.Clean(filepath.Join(base, namespace))
	rel, err := filepath.Rel(base, full)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%w: resolves outside base", ErrInvalidName)
	}
	return full, nil
}

func (f *FileStore) Store(namespace, name string, content []byte) error {
	filePath, err := f.resolvePath(namespace, name)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

func (f *FileStore) Get(namespace, name string) ([]byte, error) {
	filePath, err := f.resolvePath(namespace, name)
	if err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config %s in namespace %s: %w", name, namespace, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return content, nil
}

func (f *FileStore) Delete(namespace, name string) error {
	filePath, err := f.resolvePath(namespace, name)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config %s in namespace %s: %w", name, namespace, ErrNotFound)
		}
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}

	return nil
}

func (f *FileStore) List(namespace string) ([]string, error) {
	namespaceDir, err := f.resolveNamespaceDir(namespace)
	if err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	entries, err := os.ReadDir(namespaceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", namespaceDir, err)
	}

	configs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			configs = append(configs, entry.Name())
		}
	}

	return configs, nil
}
