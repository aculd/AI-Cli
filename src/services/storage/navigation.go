// services/storage/navigation.go - Navigation state persistence
// MIGRATION TARGET: refactoring.md (storage service integration)

package storage

import (
	"os"
	"path/filepath"
)

type NavigationStorage interface {
	SaveNavigationState(data []byte) error
	LoadNavigationState() ([]byte, error)
}

type JSONNavigationStorage struct {
	path string
}

func NewNavigationStorage(appDir string) *JSONNavigationStorage {
	return &JSONNavigationStorage{
		path: filepath.Join(appDir, "navigation_state.json"),
	}
}

func (s *JSONNavigationStorage) SaveNavigationState(data []byte) error {
	return atomicWrite(s.path, data)
}

func (s *JSONNavigationStorage) LoadNavigationState() ([]byte, error) {
	return os.ReadFile(s.path)
}

// atomicWrite ensures safe file writes
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
