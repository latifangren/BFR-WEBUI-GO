package modules_test

import (
	"bytes"
	"testing"

	"bfr-webui-go/internal/modules"
)

func TestListModules(t *testing.T) {
	list, err := modules.ListModules()
	if err != nil {
		t.Fatalf("unexpected error getting modules list: %v", err)
	}
	// On non-android systems, returns empty slice without error
	_ = list
}

func TestToggleModule_Validation(t *testing.T) {
	// Invalid ID with special characters should be rejected by regex
	err := modules.ToggleModule("invalid_mod_id; rm -rf /", true)
	if err == nil {
		t.Errorf("expected error for invalid module ID containing shell injection chars")
	}
}

func TestInstallModule_Validation(t *testing.T) {
	// Non-zip file extension should fail
	r := bytes.NewReader([]byte("dummy content"))
	_, err := modules.InstallModule(r, "bad_file.txt")
	if err == nil {
		t.Errorf("expected error when installing non-zip module file")
	}
}
