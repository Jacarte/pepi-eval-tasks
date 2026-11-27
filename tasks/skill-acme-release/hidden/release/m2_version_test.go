package release

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type m2Manifest struct {
	Version    string `json:"version"`
	Generation int    `json:"generation"`
}

func TestM2VersionFiles(t *testing.T) {
	data, err := os.ReadFile("../VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if v := strings.TrimSpace(string(data)); v != "1.4.0" {
		t.Fatalf("VERSION = %q, want 1.4.0", v)
	}
	raw, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m m2Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifest.json: %v", err)
	}
	if m.Version != "1.4.0" {
		t.Fatalf("manifest version = %q, want 1.4.0", m.Version)
	}
	// generation 7 before the release, plus one per released ticket (two).
	if m.Generation != 9 {
		t.Fatalf("manifest generation = %d, want 9", m.Generation)
	}
}
