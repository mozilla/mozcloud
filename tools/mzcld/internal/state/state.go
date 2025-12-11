package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ApplicationSyncState struct {
	Name       string          `json:"name"`
	Namespace  string          `json:"namespace"`
	SyncPolicy json.RawMessage `json:"syncPolicy"`
}

type Snapshot struct {
	Cluster      string                 `json:"cluster"`
	Namespace    string                 `json:"namespace"`
	SavedAt      time.Time              `json:"savedAt"`
	Applications []ApplicationSyncState `json:"applications"`
}

func Load(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

func Save(path string, snap *Snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
