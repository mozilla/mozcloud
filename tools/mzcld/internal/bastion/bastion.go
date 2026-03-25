// Package bastion provides types and helpers for working with Mozilla's GCP
// bastion hosts.
package bastion

import (
	"encoding/json"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/cache"
)

// BastionRealms is the ordered list of supported bastion realms.
var BastionRealms = [2]string{"prod", "nonprod"}

// BastionRegions is the ordered list of supported bastion regions.
var BastionRegions = [3]string{"us-west1", "us-central1", "europe-west1"}

// BastionPorts maps "realm:region" keys to SOCKS proxy port strings.
var BastionPorts = map[string]string{
	"prod:europe-west1":    "5001",
	"prod:us-central1":     "5002",
	"prod:us-west1":        "5003",
	"nonprod:europe-west1": "5004",
	"nonprod:us-central1":  "5005",
	"nonprod:us-west1":     "5006",
}

// BastionZone maps a region name to its canonical availability zone.
var BastionZone = map[string]string{
	"europe-west1": "europe-west1-b",
	"us-central1":  "us-central1-a",
	"us-west1":     "us-west1-a",
}

const cacheFileName = "bastion-cache.json"

// BastionCache holds the most-recently-used bastion realm and region.
type BastionCache struct {
	Realm      string `json:"realm"`
	Region     string `json:"region"`
	LastAccess string `json:"last_access"`
}

// Load reads the cached bastion selection from ~/.mops/bastion-cache.json.
// Returns nil, nil if the file does not yet exist.
func Load() (*BastionCache, error) {
	if !cache.Exists(cacheFileName) {
		return nil, nil
	}
	data, err := cache.Load(cacheFileName)
	if err != nil {
		return nil, err
	}
	var bc BastionCache
	if err := json.Unmarshal(data, &bc); err != nil {
		return nil, err
	}
	return &bc, nil
}

// Save writes bc to ~/.mops/bastion-cache.json.
func Save(bc *BastionCache) error {
	data, err := json.Marshal(bc)
	if err != nil {
		return err
	}
	return cache.Save(cacheFileName, data)
}
