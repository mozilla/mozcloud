package mcperr_test

import (
	"encoding/json"
	"testing"

	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/mcperr"
)

func TestNew(t *testing.T) {
	e := mcperr.New("not_found", "file missing", "check the path")
	if e.Error.Code != "not_found" {
		t.Errorf("code: got %q, want %q", e.Error.Code, "not_found")
	}
	if e.Error.Message != "file missing" {
		t.Errorf("message: got %q, want %q", e.Error.Message, "file missing")
	}
	if e.Error.Remediation != "check the path" {
		t.Errorf("remediation: got %q, want %q", e.Error.Remediation, "check the path")
	}
}

func TestJSON(t *testing.T) {
	e := mcperr.New("some_code", "some message", "do something")
	s := e.JSON()

	var parsed struct {
		Error struct {
			Code        string `json:"code"`
			Message     string `json:"message"`
			Remediation string `json:"remediation"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		t.Fatalf("JSON() returned invalid JSON: %v", err)
	}
	if parsed.Error.Code != "some_code" {
		t.Errorf("code: got %q, want %q", parsed.Error.Code, "some_code")
	}
	if parsed.Error.Remediation != "do something" {
		t.Errorf("remediation: got %q, want %q", parsed.Error.Remediation, "do something")
	}
}
