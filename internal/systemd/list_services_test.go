package systemd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_ListServices_Optimized(t *testing.T) {
	tmpDir := t.TempDir()
	mockSystemctl := filepath.Join(tmpDir, "mock-systemctl")
	
	// Create a mock systemctl that handles both list-unit-files and list-units
	mockScript := `#!/bin/bash
if [[ "$*" == *"--user list-unit-files"* ]]; then
    echo "rclone-mount-gdrive.service enabled"
    echo "rclone-mount-dropbox.service disabled"
    echo "rclone-sync-backup.service enabled"
    echo "other-service.service enabled"
    exit 0
fi

if [[ "$*" == *"--user list-units"* ]]; then
    # Format: UNIT LOAD ACTIVE SUB DESCRIPTION
    echo "rclone-mount-gdrive.service loaded active running Rclone mount: gdrive"
    echo "rclone-sync-backup.service  loaded active exited  Rclone sync: backup"
    # dropbox is missing from list-units because it's inactive
    exit 0
fi
exit 1
`
	if err := os.WriteFile(mockSystemctl, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock systemctl: %v", err)
	}

	m := &Manager{systemctlPath: mockSystemctl}
	services, err := m.ListServices()
	if err != nil {
		t.Fatalf("ListServices() error = %v", err)
	}

	if len(services) != 3 {
		t.Errorf("ListServices() returned %d services, want 3 (rclone services only)", len(services))
	}

	foundGdrive := false
	foundDropbox := false
	foundBackup := false

	for _, s := range services {
		switch s.Name {
		case "rclone-mount-gdrive":
			foundGdrive = true
			if !s.Active {
				t.Errorf("gdrive should be active")
			}
			if s.State != "active" {
				t.Errorf("gdrive state should be active, got %q", s.State)
			}
			if s.SubState != "running" {
				t.Errorf("gdrive substate should be running, got %q", s.SubState)
			}
		case "rclone-mount-dropbox":
			foundDropbox = true
			if s.Active {
				t.Errorf("dropbox should not be active")
			}
			if s.State != "inactive" {
				t.Errorf("dropbox state should be inactive, got %q", s.State)
			}
		case "rclone-sync-backup":
			foundBackup = true
			if !s.Active {
				t.Errorf("backup should be active")
			}
			if s.State != "active" {
				t.Errorf("backup state should be active, got %q", s.State)
			}
			if s.SubState != "exited" {
				t.Errorf("backup substate should be exited, got %q", s.SubState)
			}
		}
	}

	if !foundGdrive || !foundDropbox || !foundBackup {
		t.Errorf("Did not find all expected services: gdrive=%v, dropbox=%v, backup=%v", 
			foundGdrive, foundDropbox, foundBackup)
	}
}
