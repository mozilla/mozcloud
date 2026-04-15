// Package cache provides helpers for reading and writing files in ~/.mzcld/.
package cache

import (
	"errors"
	"os"
	"path/filepath"
)

const saveDir = ".mzcld"

// Dir returns the path to the ~/.mzcld cache directory.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, saveDir), nil
}

// Exists reports whether name exists inside the cache directory.
func Exists(name string) bool {
	p, err := FullPath(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return !errors.Is(err, os.ErrNotExist)
}

// Load reads name from the cache directory and returns its contents.
func Load(name string) ([]byte, error) {
	p, err := FullPath(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

// Save writes data to name in the cache directory, creating the directory if
// it does not already exist.
func Save(name string, data []byte) error {
	p, err := FullPath(name)
	if err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	_, err = f.Write(data)
	return err
}

// FullPath returns the absolute path for name inside the cache directory,
// creating the directory if it does not already exist.
func FullPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, saveDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}
