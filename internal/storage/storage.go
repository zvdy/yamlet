package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store interface defines the storage operations
type Store interface {
	Store(namespace, name string, content []byte) error
	Get(namespace, name string) ([]byte, error)
	Delete(namespace, name string) error
	List(namespace string) ([]string, error)
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
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data[namespace] == nil {
		m.data[namespace] = make(map[string][]byte)
	}
	m.data[namespace][name] = content
	return nil
}

func (m *MemoryStore) Get(namespace, name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	namespaceData, exists := m.data[namespace]
	if !exists {
		return nil, fmt.Errorf("namespace %s not found", namespace)
	}

	content, exists := namespaceData[name]
	if !exists {
		return nil, fmt.Errorf("config %s not found in namespace %s", name, namespace)
	}

	return content, nil
}

func (m *MemoryStore) Delete(namespace, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	namespaceData, exists := m.data[namespace]
	if !exists {
		return fmt.Errorf("namespace %s not found", namespace)
	}

	if _, exists := namespaceData[name]; !exists {
		return fmt.Errorf("config %s not found in namespace %s", name, namespace)
	}

	delete(namespaceData, name)

	// Clean up empty namespace
	if len(namespaceData) == 0 {
		delete(m.data, namespace)
	}

	return nil
}

func (m *MemoryStore) List(namespace string) ([]string, error) {
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

func (f *FileStore) Store(namespace, name string, content []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	filePath := f.getFilePath(namespace, name)

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

func (f *FileStore) Get(namespace, name string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	filePath := f.getFilePath(namespace, name)

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config %s not found in namespace %s", name, namespace)
		}
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return content, nil
}

func (f *FileStore) Delete(namespace, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	filePath := f.getFilePath(namespace, name)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config %s not found in namespace %s", name, namespace)
		}
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}

	return nil
}

func (f *FileStore) List(namespace string) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	namespaceDir := filepath.Join(f.baseDir, namespace)

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
